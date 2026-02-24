package middleware

import (
	"net/http"

	"account-stock-be/internal/rbac"
)

// RequirePermission returns a middleware that allows the request only if the
// authenticated user has the required permission (resource:action per RBAC_SPEC).
// Must run after Auth middleware. Returns 403 if permission is missing.
func RequirePermission(permission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := GetContext(r.Context())
			if ctx == nil {
				writeJSONError(w, ErrUnauthorized, http.StatusUnauthorized)
				return
			}
			if !rbac.HasPermission(ctx.Permissions, permission) {
				writeJSONError(w, ErrForbidden, http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
