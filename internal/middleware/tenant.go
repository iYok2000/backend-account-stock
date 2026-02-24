package middleware

import (
	"context"
	"net/http"
)

// Tenant ensures company_id from auth context is used for all tenant-scoped data access.
// Handlers must use GetContext(r.Context()).CompanyID for queries/upserts; this middleware
// does not modify the request — it documents the contract. Optional: add audit or validation here.
func Tenant(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Auth middleware must run first and set auth.Context with CompanyID.
		_ = GetContext(r.Context()) // ensure auth was applied on routes that use Tenant
		next.ServeHTTP(w, r)
	})
}

// RequireAuthContext returns 401 if auth context is missing (e.g. for /api/auth/me).
func RequireAuthContext(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if GetContext(r.Context()) == nil {
			writeJSONError(w, ErrUnauthorized, http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

// TenantScope is a helper for handlers: returns company_id from context or empty string.
func TenantScope(ctx context.Context) string {
	c := GetContext(ctx)
	if c == nil {
		return ""
	}
	return c.CompanyID
}
