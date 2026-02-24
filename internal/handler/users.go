package handler

import (
	"encoding/json"
	"net/http"
)

// UsersListResponse placeholder for GET /api/users (SuperAdmin only via RequirePermission).
type UsersListResponse struct {
	Users []interface{} `json:"users"`
}

// UsersList returns an empty list until user store is connected (RBAC: users:read enforced by middleware).
func UsersList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(UsersListResponse{Users: []interface{}{}})
}
