package handler

import (
	"encoding/json"
	"net/http"

	"account-stock-be/internal/middleware"
)

// MeResponse matches frontend AuthContext / useUserContext expectations.
// See project-specific_context.md and fe docs/USER_SPEC.md.
type MeResponse struct {
	User struct {
		ID          string `json:"id"`
		DisplayName string `json:"displayName,omitempty"`
	} `json:"user"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
	Tier        string   `json:"tier,omitempty"`
	CompanyID   string   `json:"company_id,omitempty"`
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
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(res)
}
