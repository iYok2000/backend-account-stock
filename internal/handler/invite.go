package handler

import (
	"crypto/rand"
	"encoding/json"
	"net/http"
	"time"

	"account-stock-be/internal/auth"
	"account-stock-be/internal/database"
	"account-stock-be/internal/middleware"
	"account-stock-be/internal/model"

	"github.com/google/uuid"
)

// POST /api/invite/validate — Validate invite code (public, before registration)
func ValidateInviteCode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		middleware.WriteJSONError(w, middleware.ErrMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Code == "" {
		middleware.WriteJSONError(w, "invalid request", http.StatusBadRequest)
		return
	}

	db := database.DB()
	var invite model.InviteCode
	if err := db.Where("code = ? AND deleted_at IS NULL", req.Code).First(&invite).Error; err != nil {
		middleware.WriteJSONError(w, "invalid or expired code", http.StatusNotFound)
		return
	}

	// Check validity
	if !invite.IsActive {
		middleware.WriteJSONError(w, "code is deactivated", http.StatusBadRequest)
		return
	}
	if invite.ExpiresAt != nil && invite.ExpiresAt.Before(time.Now()) {
		middleware.WriteJSONError(w, "code has expired", http.StatusBadRequest)
		return
	}
	if invite.UsedCount >= invite.MaxUses {
		middleware.WriteJSONError(w, "code has reached max uses", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"valid":      true,
		"grant_tier": invite.GrantTier,
		"message":    "Code is valid",
	})
}

// GET /api/invite/check-required — Check if invite code is required (public)
func CheckInviteRequired(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		middleware.WriteJSONError(w, middleware.ErrMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}

	db := database.DB()
	var config model.SystemConfig
	if err := db.Where("key = ?", "require_invite_code").First(&config).Error; err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"required": false})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"required": config.Value == "true"})
}

// POST /api/invite/use — Use invite code (authenticated user, after registration)
func UseInviteCode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		middleware.WriteJSONError(w, middleware.ErrMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}

	userCtx := middleware.GetContext(r.Context())
	if userCtx == nil || userCtx.UserID == "" {
		middleware.WriteJSONError(w, middleware.ErrUnauthorized, http.StatusUnauthorized)
		return
	}

	var req struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Code == "" {
		middleware.WriteJSONError(w, "invalid request", http.StatusBadRequest)
		return
	}

	db := database.DB()
	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Lock invite code
	var invite model.InviteCode
	if err := tx.Set("gorm:query_option", "FOR UPDATE").Where("code = ? AND deleted_at IS NULL", req.Code).First(&invite).Error; err != nil {
		tx.Rollback()
		middleware.WriteJSONError(w, "invalid or expired code", http.StatusNotFound)
		return
	}

	// Validate
	if !invite.IsActive || invite.UsedCount >= invite.MaxUses {
		tx.Rollback()
		middleware.WriteJSONError(w, "code is not available", http.StatusBadRequest)
		return
	}
	if invite.ExpiresAt != nil && invite.ExpiresAt.Before(time.Now()) {
		tx.Rollback()
		middleware.WriteJSONError(w, "code has expired", http.StatusBadRequest)
		return
	}

	// Get user
	var user model.User
	if err := tx.Where("id = ?", userCtx.UserID).First(&user).Error; err != nil {
		tx.Rollback()
		middleware.WriteJSONError(w, middleware.ErrInternal, http.StatusInternalServerError)
		return
	}

	// Update user tier
	oldTier := user.Tier
	user.Tier = invite.GrantTier
	user.TierStartedAt = timePtr(time.Now())
	if invite.TierDurationDays != nil {
		user.TierExpiresAt = timePtr(time.Now().AddDate(0, 0, *invite.TierDurationDays))
	} else {
		user.TierExpiresAt = nil // unlimited
	}
	user.InviteCodeUsed = invite.Code

	if err := tx.Save(&user).Error; err != nil {
		tx.Rollback()
		middleware.WriteJSONError(w, middleware.ErrInternal, http.StatusInternalServerError)
		return
	}

	// Increment used count
	invite.UsedCount++
	if err := tx.Save(&invite).Error; err != nil {
		tx.Rollback()
		middleware.WriteJSONError(w, middleware.ErrInternal, http.StatusInternalServerError)
		return
	}

	// Log tier change
	history := model.TierHistory{
		ID:           uuid.New().String(),
		UserID:       user.ID,
		OldTier:      oldTier,
		NewTier:      user.Tier,
		Reason:       "invite_code",
		InviteCodeID: &invite.ID,
		StartedAt:    time.Now(),
		ExpiresAt:    user.TierExpiresAt,
		CreatedAt:    time.Now(),
	}
	if err := tx.Create(&history).Error; err != nil {
		tx.Rollback()
		middleware.WriteJSONError(w, middleware.ErrInternal, http.StatusInternalServerError)
		return
	}

	tx.Commit()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Tier granted",
		"tier":    user.Tier,
	})
}

// GET /api/admin/invites — List invite codes (Root/SuperAdmin only)
func ListInviteCodes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		middleware.WriteJSONError(w, middleware.ErrMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}

	userCtx := middleware.GetContext(r.Context())
	if userCtx == nil || (userCtx.Role != auth.RoleRoot && userCtx.Role != auth.RoleSuperAdmin) {
		middleware.WriteJSONError(w, middleware.ErrForbidden, http.StatusForbidden)
		return
	}

	db := database.DB()
	var codes []model.InviteCode
	if err := db.Where("deleted_at IS NULL").Order("created_at DESC").Find(&codes).Error; err != nil {
		middleware.WriteJSONError(w, middleware.ErrInternal, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"codes": codes})
}

// POST /api/admin/invites — Create invite code (Root/SuperAdmin only)
func CreateInviteCode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		middleware.WriteJSONError(w, middleware.ErrMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}

	userCtx := middleware.GetContext(r.Context())
	if userCtx == nil || (userCtx.Role != auth.RoleRoot && userCtx.Role != auth.RoleSuperAdmin) {
		middleware.WriteJSONError(w, middleware.ErrForbidden, http.StatusForbidden)
		return
	}

	var req struct {
		Code             string     `json:"code"`
		GrantTier        string     `json:"grantTier"`
		TierDurationDays *int       `json:"tierDurationDays"`
		MaxUses          int        `json:"maxUses"`
		ExpiresAt        *time.Time `json:"expiresAt"`
		Note             string     `json:"note"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.GrantTier == "" || req.MaxUses <= 0 {
		middleware.WriteJSONError(w, "invalid request", http.StatusBadRequest)
		return
	}

	// Generate code if not provided
	code := req.Code
	if code == "" {
		code = "STOCK-" + generateRandomCode(6)
	}

	db := database.DB()
	invite := model.InviteCode{
		ID:               uuid.New().String(),
		Code:             code,
		GrantTier:        req.GrantTier,
		TierDurationDays: req.TierDurationDays,
		MaxUses:          req.MaxUses,
		UsedCount:        0,
		IsActive:         true,
		ExpiresAt:        req.ExpiresAt,
		Note:             req.Note,
		CreatedBy:        &userCtx.UserID,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	if err := db.Create(&invite).Error; err != nil {
		middleware.WriteJSONError(w, "code already exists or invalid", http.StatusConflict)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{"code": invite})
}

// PUT /api/admin/invites/:id — Update invite code (Root/SuperAdmin only)
func UpdateInviteCode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut && r.Method != http.MethodPatch {
		middleware.WriteJSONError(w, middleware.ErrMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}

	userCtx := middleware.GetContext(r.Context())
	if userCtx == nil || (userCtx.Role != auth.RoleRoot && userCtx.Role != auth.RoleSuperAdmin) {
		middleware.WriteJSONError(w, middleware.ErrForbidden, http.StatusForbidden)
		return
	}

	// Extract ID from path (/api/admin/invites/:id)
	path := r.URL.Path
	id := path[len("/api/admin/invites/"):]
	if id == "" {
		middleware.WriteJSONError(w, "invalid request", http.StatusBadRequest)
		return
	}

	var req struct {
		IsActive *bool `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		middleware.WriteJSONError(w, "invalid request", http.StatusBadRequest)
		return
	}

	db := database.DB()
	var invite model.InviteCode
	if err := db.Where("id = ? AND deleted_at IS NULL", id).First(&invite).Error; err != nil {
		middleware.WriteJSONError(w, "code not found", http.StatusNotFound)
		return
	}

	if req.IsActive != nil {
		invite.IsActive = *req.IsActive
	}
	invite.UpdatedAt = time.Now()

	if err := db.Save(&invite).Error; err != nil {
		middleware.WriteJSONError(w, middleware.ErrInternal, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"code": invite})
}

// DELETE /api/admin/invites/:id — Deactivate invite code (Root/SuperAdmin only)
func DeleteInviteCode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		middleware.WriteJSONError(w, middleware.ErrMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}

	userCtx := middleware.GetContext(r.Context())
	if userCtx == nil || (userCtx.Role != auth.RoleRoot && userCtx.Role != auth.RoleSuperAdmin) {
		middleware.WriteJSONError(w, middleware.ErrForbidden, http.StatusForbidden)
		return
	}

	path := r.URL.Path
	id := path[len("/api/admin/invites/"):]
	if id == "" {
		middleware.WriteJSONError(w, "invalid request", http.StatusBadRequest)
		return
	}

	db := database.DB()
	if err := db.Model(&model.InviteCode{}).Where("id = ?", id).Update("is_active", false).Error; err != nil {
		middleware.WriteJSONError(w, middleware.ErrInternal, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Code deactivated"})
}

// GET /api/admin/system-config — Get system config (Root/SuperAdmin only)
func GetSystemConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		middleware.WriteJSONError(w, middleware.ErrMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}

	userCtx := middleware.GetContext(r.Context())
	if userCtx == nil || (userCtx.Role != auth.RoleRoot && userCtx.Role != auth.RoleSuperAdmin) {
		middleware.WriteJSONError(w, middleware.ErrForbidden, http.StatusForbidden)
		return
	}

	db := database.DB()
	var configs []model.SystemConfig
	if err := db.Find(&configs).Error; err != nil {
		middleware.WriteJSONError(w, middleware.ErrInternal, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"configs": configs})
}

// PUT /api/admin/system-config — Update system config (Root only)
func UpdateSystemConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut && r.Method != http.MethodPatch {
		middleware.WriteJSONError(w, middleware.ErrMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}

	userCtx := middleware.GetContext(r.Context())
	if userCtx == nil || userCtx.Role != auth.RoleRoot {
		middleware.WriteJSONError(w, middleware.ErrForbidden, http.StatusForbidden)
		return
	}

	var req struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Key == "" || req.Value == "" {
		middleware.WriteJSONError(w, "invalid request", http.StatusBadRequest)
		return
	}

	db := database.DB()
	var config model.SystemConfig
	if err := db.Where("key = ?", req.Key).First(&config).Error; err != nil {
		// Create if not exists
		config = model.SystemConfig{
			ID:        uuid.New().String(),
			Key:       req.Key,
			Value:     req.Value,
			UpdatedBy: &userCtx.UserID,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := db.Create(&config).Error; err != nil {
			middleware.WriteJSONError(w, middleware.ErrInternal, http.StatusInternalServerError)
			return
		}
	} else {
		config.Value = req.Value
		config.UpdatedBy = &userCtx.UserID
		config.UpdatedAt = time.Now()
		if err := db.Save(&config).Error; err != nil {
			middleware.WriteJSONError(w, middleware.ErrInternal, http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"config": config})
}

// Helper: generate random alphanumeric code
func generateRandomCode(length int) string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	rand.Read(b)
	for i := range b {
		b[i] = charset[int(b[i])%len(charset)]
	}
	return string(b)
}

// Helper: convert time to pointer
func timePtr(t time.Time) *time.Time {
	return &t
}
