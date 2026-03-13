package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"account-stock-be/internal/auth"
	"account-stock-be/internal/database"
	"account-stock-be/internal/middleware"
	"account-stock-be/internal/model"
	"account-stock-be/internal/rbac"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Self handles PATCH/DELETE /api/users/me.
// - PATCH: update display_name or password (requires users:update or RoleRoot)
// - DELETE: soft delete self (requires users:delete)
func Self(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPatch:
		updateSelf(w, r)
	case http.MethodDelete:
		deleteSelf(w, r)
	default:
		middleware.WriteJSONError(w, middleware.ErrMethodNotAllowed, http.StatusMethodNotAllowed)
	}
}

type updateSelfRequest struct {
	DisplayName string `json:"display_name"`
	Password    string `json:"password"`
}

func updateSelf(w http.ResponseWriter, r *http.Request) {
	ctx := middleware.GetContext(r.Context())
	if ctx == nil || ctx.UserID == "" {
		middleware.WriteJSONError(w, middleware.ErrUnauthorized, http.StatusUnauthorized)
		return
	}
	// NOTE: /api/users/me allows ANY authenticated user to update their own profile.
	// No special permission required — this is self-service, not managing other users.
	// The users:update permission is for managing OTHER shop members (SuperAdmin/Admin only).

	var body updateSelfRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		middleware.WriteJSONError(w, middleware.ErrInvalidJSON, http.StatusBadRequest)
		return
	}
	display := strings.TrimSpace(body.DisplayName)
	pass := body.Password
	if display == "" && pass == "" {
		middleware.WriteJSONErrorMsg(w, "nothing to update", http.StatusBadRequest)
		return
	}

	updates := map[string]interface{}{}
	if display != "" {
		updates["display_name"] = display
	}
	if pass != "" {
		hash, err := auth.HashPassword(pass)
		if err != nil {
			middleware.WriteJSONErrorMsg(w, "invalid password", http.StatusBadRequest)
			return
		}
		updates["password_hash"] = hash
	}

	db := database.DB()
	if db == nil {
		middleware.WriteJSONError(w, middleware.ErrInternal, http.StatusInternalServerError)
		return
	}
	if err := db.Model(&model.User{}).Where("id = ?", ctx.UserID).Updates(updates).Error; err != nil {
		middleware.WriteJSONError(w, middleware.ErrInternal, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// DeleteSelf handles DELETE /api/users/me (soft delete).
// If user is SuperAdmin and has shop_id, soft-deletes shop + users in that shop + import_sku_row.
func deleteSelf(w http.ResponseWriter, r *http.Request) {
	ctx := middleware.GetContext(r.Context())
	if ctx == nil || ctx.UserID == "" {
		middleware.WriteJSONError(w, middleware.ErrUnauthorized, http.StatusUnauthorized)
		return
	}
	// NOTE: Self-deletion (account termination) requires explicit permission.
	// This is a destructive action that may cascade to shop data (for SuperAdmin).
	// Require users:delete permission for safety.
	if ctx.Role != auth.RoleRoot && !rbac.HasPermission(ctx.Permissions, rbac.PermUsersDelete) {
		middleware.WriteJSONError(w, middleware.ErrForbidden, http.StatusForbidden)
		return
	}
	db := database.DB()
	if db == nil {
		middleware.WriteJSONError(w, middleware.ErrInternal, http.StatusInternalServerError)
		return
	}

	if err := db.Transaction(func(tx *gorm.DB) error {
		if ctx.Role == "SuperAdmin" && ctx.ShopID != "" {
			if err := softDelete(tx, "shops", "id = ?", ctx.ShopID); err != nil {
				return err
			}
			if err := softDelete(tx, "users", "shop_id = ?", ctx.ShopID); err != nil {
				return err
			}
			_ = softDelete(tx, "import_sku_row", "shop_id = ?", ctx.ShopID)
		}
		if err := softDelete(tx, "users", "id = ?", ctx.UserID); err != nil {
			return err
		}
		return nil
	}); err != nil {
		middleware.WriteJSONError(w, middleware.ErrInternal, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func softDelete(tx *gorm.DB, table string, query string, args ...interface{}) error {
	return tx.Table(table).Clauses(clause.Returning{}).Where(query, args...).Update("deleted_at", gorm.Expr("now()")).Error
}
