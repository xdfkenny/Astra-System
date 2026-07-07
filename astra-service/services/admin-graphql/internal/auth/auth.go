// Package auth implements JWT validation and admin authorization for the
// admin-graphql service.
package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// Context keys.
// ContextKey is the key type used to store claims in a context.
type ContextKey int

const (
	claimsKey ContextKey = iota
)

// NewContextWithClaims returns a context containing the supplied claims.
func NewContextWithClaims(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, claimsKey, claims)
}

// Claims contains the validated JWT claims used by resolvers.
type Claims struct {
	Subject  string
	TenantID string
	Role     string
	IsAdmin  bool
}

// Config holds the JWT validation configuration.
type Config struct {
	Secret   []byte
	Issuer   string
	Audience string
}

// Middleware returns an HTTP middleware that validates JWTs and requires an
// admin role.
func Middleware(cfg Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/healthz" {
				next.ServeHTTP(w, r)
				return
			}

			auth := r.Header.Get("Authorization")
			const prefix = "Bearer "
			if !strings.HasPrefix(auth, prefix) {
				writeError(w, http.StatusUnauthorized, "missing bearer token")
				return
			}
			tokenStr := strings.TrimPrefix(auth, prefix)

			claims, err := Parse(tokenStr, cfg)
			if err != nil {
				writeError(w, http.StatusUnauthorized, fmt.Sprintf("invalid token: %v", err))
				return
			}
			if !claims.IsAdmin {
				writeError(w, http.StatusForbidden, "admin access required")
				return
			}

			ctx := context.WithValue(r.Context(), claimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// Parse validates a raw JWT string and extracts admin claims.
func Parse(tokenStr string, cfg Config) (*Claims, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return cfg.Secret, nil
	},
		jwt.WithIssuer(cfg.Issuer),
		jwt.WithAudience(cfg.Audience),
		jwt.WithValidMethods([]string{"HS256"}),
	)
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, fmt.Errorf("token is invalid")
	}

	mapClaims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("unexpected claims type")
	}

	claims := &Claims{
		Subject:  stringValue(mapClaims, "sub"),
		TenantID: stringValue(mapClaims, "tenant_id"),
		Role:     stringValue(mapClaims, "role"),
		IsAdmin:  boolValue(mapClaims, "is_admin"),
	}
	if claims.Role == "admin" {
		claims.IsAdmin = true
	}
	return claims, nil
}

// ClaimsFromContext extracts validated claims from the request context.
func ClaimsFromContext(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value(claimsKey).(*Claims)
	return claims, ok
}

func stringValue(claims jwt.MapClaims, key string) string {
	v, ok := claims[key].(string)
	if !ok {
		return ""
	}
	return v
}

func boolValue(claims jwt.MapClaims, key string) bool {
	switch v := claims[key].(type) {
	case bool:
		return v
	case string:
		return v == "true"
	default:
		return false
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(`{"errors":[{"message":"` + message + `"}]}`))
}
