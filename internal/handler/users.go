package handler

import (
	"encoding/json"
	"net/http"

	"account-stock-be/internal/database"
	"account-stock-be/internal/middleware"
	"account-stock-be/internal/model"
)

// UsersListResponse placeholder for GET /api/users (SuperAdmin only via RequirePermission).
type UsersListResponse struct {
	Users []UserItem `json:"users"`
}

type UserItem struct {
	ID          string  `json:"id"`
	CompanyID   string  `json:"company_id"`
	ShopID      *string `json:"shop_id,omitempty"`
	Email       string  `json:"email"`
	DisplayName string  `json:"display_name"`
	Role        string  `json:"role"`
	Tier        string  `json:"tier"`
}

// UsersList returns users in the caller's shop/company (RBAC: users:read enforced by middleware).
func UsersList(w http.ResponseWriter, r *http.Request) {
	ctx := middleware.GetContext(r.Context())
	if ctx == nil {
		middleware.WriteJSONError(w, middleware.ErrUnauthorized, http.StatusUnauthorized)
		return
	}
	db := database.DB()
	if db == nil {
		middleware.WriteJSONError(w, "service unavailable", http.StatusServiceUnavailable)
		return
	}
	if ctx.CompanyID == "" {
		// Root user has no tenant scope; return empty list for this endpoint.
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(UsersListResponse{Users: []UserItem{}})
		return
	}
	var users []model.User
	q := db.Model(&model.User{}).Where("company_id = ?", ctx.CompanyID)
	if err := q.Find(&users).Error; err != nil {
		middleware.WriteJSONError(w, middleware.ErrInternal, http.StatusInternalServerError)
		return
	}
	items := make([]UserItem, 0, len(users))
	for _, u := range users {
		items = append(items, UserItem{
			ID:          u.ID,
			CompanyID:   u.CompanyID,
			ShopID:      u.ShopID,
			Email:       u.Email,
			DisplayName: u.DisplayName,
			Role:        u.Role,
			Tier:        u.Tier,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(UsersListResponse{Users: items})
}
