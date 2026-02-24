package auth

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims holds JWT claims for user context (USER_SPEC, RBAC_SPEC).
type Claims struct {
	jwt.RegisteredClaims
	Role        string `json:"role"`
	Tier        string `json:"tier"`
	CompanyID   string `json:"company_id"`
	DisplayName string `json:"display_name,omitempty"`
}

// JWTConfig for validation (secret from env).
type JWTConfig struct {
	Secret     []byte
	Issuer     string
	Audience   string
	ValidAlgs  []string
	LeewaySec  time.Duration
}

// DefaultJWTConfig reads JWT_SECRET from env; optional JWT_ISSUER, JWT_AUDIENCE.
func DefaultJWTConfig() JWTConfig {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "dev-secret-change-in-production"
	}
	cfg := JWTConfig{
		Secret:    []byte(secret),
		ValidAlgs: []string{"HS256"},
		LeewaySec: 30 * time.Second,
	}
	if v := os.Getenv("JWT_ISSUER"); v != "" {
		cfg.Issuer = v
	}
	if v := os.Getenv("JWT_AUDIENCE"); v != "" {
		cfg.Audience = v
	}
	return cfg
}

// ValidateToken parses and validates the Bearer token string and returns claims.
// Returns error if token is invalid or expired.
func ValidateToken(tokenString string, cfg JWTConfig) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if t.Method.Alg() != "HS256" {
			return nil, fmt.Errorf("unexpected alg: %s", t.Method.Alg())
		}
		return cfg.Secret, nil
	}, jwt.WithValidMethods(cfg.ValidAlgs), jwt.WithLeeway(cfg.LeewaySec))
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}
	if cfg.Issuer != "" && claims.Issuer != cfg.Issuer {
		return nil, fmt.Errorf("invalid issuer")
	}
	if cfg.Audience != "" {
		aud, _ := claims.GetAudience()
		if len(aud) == 0 || aud[0] != cfg.Audience {
			return nil, fmt.Errorf("invalid audience")
		}
	}
	return claims, nil
}

// Max claim lengths to prevent DoS / oversized payloads (OWASP A04, injection surface).
const (
	MaxClaimSubjectLen     = 256
	MaxClaimCompanyIDLen   = 256
	MaxClaimDisplayNameLen = 256
)

// ValidateClaimLengths returns an error if any claim exceeds safe length.
func ValidateClaimLengths(claims *Claims) error {
	if len(claims.Subject) > MaxClaimSubjectLen {
		return fmt.Errorf("subject too long")
	}
	if len(claims.CompanyID) > MaxClaimCompanyIDLen {
		return fmt.Errorf("company_id too long")
	}
	if len(claims.DisplayName) > MaxClaimDisplayNameLen {
		return fmt.Errorf("display_name too long")
	}
	return nil
}

// MaxTokenLen prevents DoS from oversized Authorization header (OWASP A04).
const MaxTokenLen = 8192

// ParseBearer extracts "Bearer <token>" from Authorization header.
// Rejects tokens longer than MaxTokenLen to prevent DoS.
func ParseBearer(authHeader string) (string, bool) {
	const prefix = "Bearer "
	if authHeader == "" || !strings.HasPrefix(authHeader, prefix) {
		return "", false
	}
	token := strings.TrimSpace(authHeader[len(prefix):])
	if len(token) > MaxTokenLen {
		return "", false
	}
	return token, true
}
