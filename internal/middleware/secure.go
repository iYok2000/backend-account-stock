package middleware

import (
	"encoding/json"
	"net/http"
)

// Predefined error messages only — never pass user input to avoid injection (OWASP A03).
const (
	ErrUnauthorized = "unauthorized"
	ErrForbidden    = "forbidden"
	ErrInvalidToken = "invalid or expired token"
	ErrMethodNotAllowed = "method not allowed"
)

type errorResponse struct {
	Error string `json:"error"`
}

// WriteJSONError sends a JSON body {"error": "<msg>"} using encoding/json (injection-safe).
// Only use predefined msg constants above; do not pass user input.
func WriteJSONError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(errorResponse{Error: msg})
}

// writeJSONError is the internal version used by middleware (same implementation).
func writeJSONError(w http.ResponseWriter, msg string, code int) {
	WriteJSONError(w, msg, code)
}
