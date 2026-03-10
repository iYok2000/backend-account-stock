package middleware

import (
	"net/http"
	"os"
	"strings"
)

// CORS wraps a handler and adds Access-Control-Allow-* headers for browser requests.
// Allowed origin from env CORS_ORIGIN (comma-separated). Default: localhost:3000 and 127.0.0.1:3000 for dev.
func CORS(next http.Handler) http.Handler {
	allowed := os.Getenv("CORS_ORIGIN")
	if allowed == "" {
		allowed = "http://localhost:3000,http://127.0.0.1:3000"
	}
	// Remove surrounding quotes if present (Railway/other platforms may include them)
	allowed = strings.Trim(allowed, `"'`)
	origins := strings.Split(allowed, ",")
	for i := range origins {
		origins[i] = strings.TrimSpace(origins[i])
		origins[i] = strings.Trim(origins[i], `"'`) // Also trim quotes from individual origins
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		allowedOrigin := ""
		for _, o := range origins {
			if o != "" && (o == "*" || o == origin) {
				allowedOrigin = o
				break
			}
		}
		// Dev: if CORS_ORIGIN unset and request from localhost/127.0.0.1, allow it (any port)
		if allowedOrigin == "" && origin != "" && (strings.HasPrefix(origin, "http://localhost:") || strings.HasPrefix(origin, "http://127.0.0.1:")) {
			allowedOrigin = origin
		}
		if allowedOrigin != "" {
			w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Max-Age", "86400")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
