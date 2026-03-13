package handler

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"account-stock-be/internal/auth"
	"account-stock-be/internal/database"
	"account-stock-be/internal/middleware"
	"account-stock-be/internal/model"
)

const defaultRootCompanyID = "YPC"
const defaultRootShopID = "00000000-0000-0000-0000-000000000001" // UUID string to match import_sku_row.shop_id (uuid)
const defaultRootShopName = "YPC Affiliate"

// MeResponse matches frontend AuthContext / useUserContext (SHOPS_AND_ROLES_SPEC, USER_SPEC).
type MeResponse struct {
	User struct {
		ID          string `json:"id"`
		DisplayName string `json:"displayName,omitempty"`
	} `json:"user"`
	Roles          []string `json:"roles"`
	Permissions    []string `json:"permissions"`
	Tier           string   `json:"tier,omitempty"`
	TierStartedAt  *string  `json:"tier_started_at,omitempty"`
	TierExpiresAt  *string  `json:"tier_expires_at,omitempty"`
	InviteCodeUsed *string  `json:"invite_code_used,omitempty"`
	InviteSlots    int      `json:"invite_slots"`
	CompanyID      string   `json:"company_id,omitempty"`
	ShopID         *string  `json:"shop_id,omitempty"`
	ShopName       string   `json:"shop_name,omitempty"`
}

// Me handles GET /api/auth/me — returns current user context for frontend AuthContext.
func Me(w http.ResponseWriter, r *http.Request) {
	ctx := middleware.GetContext(r.Context())
	if ctx == nil {
		middleware.WriteJSONError(w, middleware.ErrUnauthorized, http.StatusUnauthorized)
		return
	}
	res := MeResponse{}
	res.User.ID = ctx.UserID
	res.User.DisplayName = ctx.DisplayName
	res.Roles = []string{string(ctx.Role)}
	res.Permissions = ctx.Permissions
	res.Tier = string(ctx.Tier)
	res.CompanyID = ctx.CompanyID
	if ctx.ShopID != "" {
		res.ShopID = &ctx.ShopID
	}
	res.ShopName = ctx.ShopName

	// Look up user from DB for tier tracking fields (USER_SPEC)
	if ctx.UserID != "" && ctx.UserID != "root" {
		db := database.DB()
		if db != nil {
			var user model.User
			if err := db.Where("id = ?", ctx.UserID).First(&user).Error; err == nil {
				if user.TierStartedAt != nil {
					ts := user.TierStartedAt.Format("2006-01-02T15:04:05Z07:00")
					res.TierStartedAt = &ts
				}
				if user.TierExpiresAt != nil {
					ts := user.TierExpiresAt.Format("2006-01-02T15:04:05Z07:00")
					res.TierExpiresAt = &ts
				}
				if user.InviteCodeUsed != "" {
					res.InviteCodeUsed = &user.InviteCodeUsed
				}
				res.InviteSlots = user.InviteSlots
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(res)
}

// LoginRequest body for POST /api/auth/login.
type LoginRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	ConfirmCode string `json:"confirm_code,omitempty"`
}

// LoginResponse body on success.
type LoginResponse struct {
	Token string `json:"token"`
}

// Login handles POST /api/auth/login. Root: env ROOT_EMAIL, ROOT_PASSWORD, ROOT_CONFIRM_CODE. Others: DB user + bcrypt.
func Login(w http.ResponseWriter, r *http.Request, jwtCfg auth.JWTConfig) {
	if r.Method != http.MethodPost {
		middleware.WriteJSONError(w, middleware.ErrMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}
	var body LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		middleware.WriteJSONError(w, middleware.ErrInvalidJSON, http.StatusBadRequest)
		return
	}
	email := strings.TrimSpace(body.Email)
	password := body.Password
	confirmCode := strings.TrimSpace(body.ConfirmCode)

	// appEnv := strings.ToLower(os.Getenv("APP_ENV"))
	rootEmail := strings.TrimSpace(os.Getenv("ROOT_EMAIL"))
	rootPassword := os.Getenv("ROOT_PASSWORD")
	rootConfirm := strings.TrimSpace(os.Getenv("ROOT_CONFIRM_CODE"))

	// Dev fallback - use default values if not set
	if rootEmail == "" {
		rootEmail = "superadmin"
	}
	if rootPassword == "" {
		rootPassword = "pass@1congrate"
	}
	if rootConfirm == "" {
		rootConfirm = "YIM2021"
	}

	if email == rootEmail && password == rootPassword {
		if confirmCode != rootConfirm {
			middleware.WriteJSONError(w, middleware.ErrUnauthorized, http.StatusUnauthorized)
			return
		}
		// Root has NO shop_id — platform-only account (SHOPS_AND_ROLES_SPEC §1, §6)
		claims := &auth.Claims{}
		claims.Subject = "root"
		claims.Role = string(auth.RoleRoot)
		claims.Tier = string(auth.TierFree)
		claims.CompanyID = ""
		claims.ShopID = ""
		claims.ShopName = ""
		claims.DisplayName = "Root"
		token, err := auth.IssueToken(jwtCfg, claims)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(LoginResponse{Token: token})
		return
	}

	db := database.DB()
	if db == nil {
		middleware.WriteJSONError(w, "service unavailable", http.StatusServiceUnavailable)
		return
	}
	var user model.User
	if err := db.Where("email = ?", email).First(&user).Error; err != nil {
		middleware.WriteJSONError(w, middleware.ErrUnauthorized, http.StatusUnauthorized)
		return
	}
	if user.PasswordHash == "" || !auth.ComparePassword(user.PasswordHash, password) {
		middleware.WriteJSONError(w, middleware.ErrUnauthorized, http.StatusUnauthorized)
		return
	}
	role, ok := auth.ValidRole(user.Role)
	if !ok {
		middleware.WriteJSONError(w, middleware.ErrUnauthorized, http.StatusUnauthorized)
		return
	}
	shopID := ""
	shopName := ""
	if user.ShopID != nil && *user.ShopID != "" {
		shopID = *user.ShopID
		var shop model.Shop
		if err := db.Where("id = ?", shopID).First(&shop).Error; err == nil {
			shopName = shop.Name
		}
	}
	companyID := strings.TrimSpace(user.CompanyID)
	if companyID == "" {
		companyID = shopID // fallback: align company with shop when legacy data missing
	}
	claims := &auth.Claims{}
	claims.Subject = user.ID
	claims.Role = string(role)
	claims.Tier = user.Tier
	claims.ShopID = shopID
	claims.ShopName = shopName
	claims.DisplayName = user.DisplayName
	claims.CompanyID = companyID
	token, err := auth.IssueToken(jwtCfg, claims)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(LoginResponse{Token: token})
}
