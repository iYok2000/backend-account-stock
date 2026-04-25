package handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"account-stock-be/internal/auth"
	"account-stock-be/internal/database"
	"account-stock-be/internal/middleware"
	"account-stock-be/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm/clause"
)

// RegisterRequest body for POST /api/auth/register.
// invite_code is required when system config require_invite_code = "true".
type RegisterRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
	InviteCode  string `json:"invite_code"`
}

// Register handles POST /api/auth/register (public endpoint).
// Flow:
//  1. Validate input (email, password length)
//  2. Check system config require_invite_code — if true, invite_code must be valid
//  3. Create user with role=Affiliate, tier=free, no shop (standalone user)
//  4. If invite code provided and valid, apply tier grant atomically
//  5. Return JWT token (auto-login after register)
func Register(w http.ResponseWriter, r *http.Request, jwtCfg auth.JWTConfig) {
	if r.Method != http.MethodPost {
		middleware.WriteJSONError(w, middleware.ErrMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}

	var body RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		middleware.WriteJSONError(w, middleware.ErrInvalidJSON, http.StatusBadRequest)
		return
	}

	email := strings.TrimSpace(strings.ToLower(body.Email))
	password := body.Password
	displayName := strings.TrimSpace(body.DisplayName)
	inviteCode := strings.TrimSpace(body.InviteCode)

	// Basic validation
	if email == "" || !strings.Contains(email, "@") {
		middleware.WriteJSONError(w, "valid email is required", http.StatusBadRequest)
		return
	}
	if len(password) < 8 {
		middleware.WriteJSONError(w, "password must be at least 8 characters", http.StatusBadRequest)
		return
	}

	db := database.DB()
	if db == nil {
		middleware.WriteJSONError(w, "service unavailable", http.StatusServiceUnavailable)
		return
	}

	// Check if invite code is required
	var config model.SystemConfig
	requireInvite := false
	if err := db.Where("key = ?", "require_invite_code").First(&config).Error; err == nil {
		requireInvite = config.Value == "true"
	}
	if requireInvite && inviteCode == "" {
		middleware.WriteJSONError(w, "invite code is required to register", http.StatusForbidden)
		return
	}

	// Validate invite code if provided
	var invite model.InviteCode
	if inviteCode != "" {
		if err := db.Where("code = ? AND deleted_at IS NULL", inviteCode).First(&invite).Error; err != nil {
			middleware.WriteJSONError(w, "invalid or expired invite code", http.StatusBadRequest)
			return
		}
		if !invite.IsActive || invite.UsedCount >= invite.MaxUses {
			middleware.WriteJSONError(w, "invite code is not available", http.StatusBadRequest)
			return
		}
		if invite.ExpiresAt != nil && invite.ExpiresAt.Before(time.Now()) {
			middleware.WriteJSONError(w, "invite code has expired", http.StatusBadRequest)
			return
		}
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		middleware.WriteJSONError(w, middleware.ErrInternal, http.StatusInternalServerError)
		return
	}

	userID := uuid.New().String()
	now := time.Now()

	// Determine tier from invite code
	tier := string(auth.TierFree)
	var tierStartedAt *time.Time
	var tierExpiresAt *time.Time
	if inviteCode != "" {
		tier = invite.GrantTier
		tierStartedAt = &now
		if invite.TierDurationDays != nil {
			exp := now.AddDate(0, 0, *invite.TierDurationDays)
			tierExpiresAt = &exp
		}
	}

	newUser := model.User{
		ID:             userID,
		Email:          email,
		PasswordHash:   hash,
		DisplayName:    displayName,
		Role:           string(auth.RoleAffiliate),
		Tier:           tier,
		TierStartedAt:  tierStartedAt,
		TierExpiresAt:  tierExpiresAt,
		InviteCodeUsed: inviteCode,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	// Use raw gorm transaction
	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Create(&newUser).Error; err != nil {
		tx.Rollback()
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			middleware.WriteJSONError(w, "email already registered", http.StatusConflict)
			return
		}
		middleware.WriteJSONError(w, middleware.ErrInternal, http.StatusInternalServerError)
		return
	}

	// Apply invite code: increment used_count and record tier history
	if inviteCode != "" {
		var lockedInvite model.InviteCode
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", invite.ID).First(&lockedInvite).Error; err != nil {
			tx.Rollback()
			middleware.WriteJSONError(w, middleware.ErrInternal, http.StatusInternalServerError)
			return
		}
		// Re-check availability under lock
		if !lockedInvite.IsActive || lockedInvite.UsedCount >= lockedInvite.MaxUses {
			tx.Rollback()
			middleware.WriteJSONError(w, "invite code is no longer available", http.StatusConflict)
			return
		}
		lockedInvite.UsedCount++
		if err := tx.Save(&lockedInvite).Error; err != nil {
			tx.Rollback()
			middleware.WriteJSONError(w, middleware.ErrInternal, http.StatusInternalServerError)
			return
		}
		history := model.TierHistory{
			ID:           uuid.New().String(),
			UserID:       userID,
			OldTier:      string(auth.TierFree),
			NewTier:      tier,
			Reason:       "invite_code",
			InviteCodeID: &invite.ID,
			StartedAt:    now,
			ExpiresAt:    tierExpiresAt,
			CreatedAt:    now,
		}
		if err := tx.Create(&history).Error; err != nil {
			tx.Rollback()
			middleware.WriteJSONError(w, middleware.ErrInternal, http.StatusInternalServerError)
			return
		}
	}

	if err := tx.Commit().Error; err != nil {
		middleware.WriteJSONError(w, middleware.ErrInternal, http.StatusInternalServerError)
		return
	}

	// Issue JWT (auto-login)
	claims := &auth.Claims{}
	claims.Subject = userID
	claims.Role = string(auth.RoleAffiliate)
	claims.Tier = tier
	claims.CompanyID = ""
	claims.ShopID = ""
	claims.DisplayName = displayName
	token, err := auth.IssueToken(jwtCfg, claims)
	if err != nil {
		middleware.WriteJSONError(w, middleware.ErrInternal, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(LoginResponse{Token: token})
}
