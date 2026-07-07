// Package auth implements JWT validation for the gateway. EdDSA is the primary
// algorithm; RS256 is supported as a fallback for legacy issuers.
package auth

import (
	"fmt"
	"strings"

	"github.com/astra-systems/astra-service/services/gateway/internal/config"
	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
)

const (
	claimsKey  = "claims"
	subjectKey = "subject"
)

// SkipAuth returns true for public health/readiness/metrics endpoints and
// CORS preflight requests.
func SkipAuth(c fiber.Ctx) bool {
	if c.Method() == fiber.MethodOptions {
		return true
	}
	path := c.Path()
	return path == "/health" || path == "/live" || path == "/ready" || path == "/metrics" || path == "/docs" || strings.HasPrefix(path, "/docs/")
}

// Middleware returns a Fiber handler that validates JWTs using EdDSA or RS256.
func Middleware(cfg *config.Config) fiber.Handler {
	return func(c fiber.Ctx) error {
		if SkipAuth(c) {
			return c.Next()
		}

		auth := c.Get("Authorization")
		const prefix = "Bearer "
		if !strings.HasPrefix(auth, prefix) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing_bearer_token"})
		}
		tokenStr := strings.TrimPrefix(auth, prefix)

		claims, err := Parse(tokenStr, cfg)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid_token", "detail": err.Error()})
		}

		c.Locals(claimsKey, claims)
		if sub, ok := claims["sub"].(string); ok {
			c.Locals(subjectKey, sub)
		}
		return c.Next()
	}
}

// Parse validates a raw JWT string against the configured public keys.
func Parse(tokenStr string, cfg *config.Config) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenStr, keyFunc(cfg),
		jwt.WithIssuer(cfg.JWTIssuer),
		jwt.WithAudience(cfg.JWTAudience),
		jwt.WithValidMethods([]string{"EdDSA", "RS256"}),
	)
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, fmt.Errorf("token is invalid")
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("unexpected claims type")
	}
	return claims, nil
}

func keyFunc(cfg *config.Config) jwt.Keyfunc {
	return func(token *jwt.Token) (interface{}, error) {
		switch token.Method.(type) {
		case *jwt.SigningMethodEd25519:
			if cfg.EdDSAPublicKey == nil {
				return nil, fmt.Errorf("EdDSA key not configured")
			}
			return cfg.EdDSAPublicKey, nil
		case *jwt.SigningMethodRSA:
			if cfg.RSAPublicKey == nil {
				return nil, fmt.Errorf("RSA key not configured")
			}
			return cfg.RSAPublicKey, nil
		default:
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
	}
}

// ClaimsFromContext extracts validated JWT claims from the Fiber context.
func ClaimsFromContext(c fiber.Ctx) (jwt.MapClaims, bool) {
	claims, ok := c.Locals(claimsKey).(jwt.MapClaims)
	return claims, ok
}

// SubjectFromContext returns the validated subject claim, if any.
func SubjectFromContext(c fiber.Ctx) (string, bool) {
	sub, ok := c.Locals(subjectKey).(string)
	return sub, ok
}
