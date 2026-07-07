package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMiddlewareAllowsHealth(t *testing.T) {
	cfg := testConfig()
	handler := Middleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestMiddlewareRejectsMissingToken(t *testing.T) {
	cfg := testConfig()
	handler := Middleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/graphql", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestMiddlewareRejectsNonAdmin(t *testing.T) {
	cfg := testConfig()
	token := signToken(t, cfg, "user-1", false)
	handler := Middleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/graphql", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestMiddlewareAllowsAdmin(t *testing.T) {
	cfg := testConfig()
	token := signToken(t, cfg, "admin-1", true)
	handler := Middleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		require.True(t, ok)
		assert.Equal(t, "admin-1", claims.Subject)
		assert.True(t, claims.IsAdmin)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/graphql", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestParseInvalidToken(t *testing.T) {
	cfg := testConfig()
	_, err := Parse("not-a-token", cfg)
	require.Error(t, err)
}

func testConfig() Config {
	return Config{
		Secret:   []byte("secret-secret-secret-secret-secret"),
		Issuer:   "astra-service",
		Audience: "astra-admin",
	}
}

func signToken(t *testing.T, cfg Config, sub string, isAdmin bool) string {
	t.Helper()
	role := "user"
	if isAdmin {
		role = "admin"
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iss":       cfg.Issuer,
		"aud":       cfg.Audience,
		"sub":       sub,
		"tenant_id": "tenant-1",
		"role":      role,
		"is_admin":  isAdmin,
		"iat":       time.Now().Unix(),
		"exp":       time.Now().Add(time.Hour).Unix(),
	})
	signed, err := token.SignedString(cfg.Secret)
	require.NoError(t, err)
	return signed
}
