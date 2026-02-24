package middleware

import (
	"context"
	"net/http"

	"account-stock-be/internal/auth"
	"account-stock-be/internal/rbac"
)

type contextKey string

const authContextKey contextKey = "auth"

// Auth extracts user context from JWT (Bearer token) and sets it on the request context.
// Returns 401 if Authorization header is missing or token is invalid/expired.
func Auth(cfg auth.JWTConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := auth.ParseBearer(r.Header.Get("Authorization"))
			if !ok || token == "" {
				writeJSONError(w, ErrUnauthorized, http.StatusUnauthorized)
				return
			}
			claims, err := auth.ValidateToken(token, cfg)
			if err != nil {
				writeJSONError(w, ErrInvalidToken, http.StatusUnauthorized)
				return
			}
			role, ok := auth.ValidRole(claims.Role)
			if !ok {
				writeJSONError(w, ErrInvalidToken, http.StatusUnauthorized)
				return
			}
			if err := auth.ValidateClaimLengths(claims); err != nil {
				writeJSONError(w, ErrInvalidToken, http.StatusUnauthorized)
				return
			}
			permissions := rbac.PermissionsForRole(role)
			tier := auth.ValidTier(claims.Tier)
			ctx := &auth.Context{
				UserID:      claims.Subject,
				Role:        role,
				Tier:        tier,
				CompanyID:   claims.CompanyID,
				DisplayName: claims.DisplayName,
				Permissions: permissions,
			}
			if ctx.UserID == "" {
				ctx.UserID = "unknown"
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), authContextKey, ctx)))
		})
	}
}

// GetContext returns the auth context from the request context.
// Must be called only after Auth middleware.
func GetContext(ctx context.Context) *auth.Context {
	v := ctx.Value(authContextKey)
	if v == nil {
		return nil
	}
	return v.(*auth.Context)
}
