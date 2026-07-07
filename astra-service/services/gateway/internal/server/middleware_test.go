package server

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/astra-systems/astra-service/services/gateway/internal/config"
	"github.com/astra-systems/astra-service/services/gateway/internal/routes"
	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestMiddlewareChain_HealthSkipsAuth(t *testing.T) {
	app := newTestApp(t)

	for _, path := range []string{"/health", "/live", "/ready", "/metrics"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode, path)
	}
}

func TestMiddlewareChain_ProtectedRequiresAuth(t *testing.T) {
	app := newTestApp(t)

	req := httptest.NewRequest(http.MethodGet, "/v1/menu", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestMiddlewareChain_AllowedOrigin(t *testing.T) {
	app := newTestApp(t)

	req := httptest.NewRequest(http.MethodOptions, "/v1/menu", nil)
	req.Header.Set("Origin", "http://localhost:5170")
	req.Header.Set("Access-Control-Request-Method", "GET")
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
	require.Equal(t, "http://localhost:5170", resp.Header.Get("Access-Control-Allow-Origin"))
}

func TestMiddlewareChain_DisallowedOrigin(t *testing.T) {
	app := newTestApp(t)

	req := httptest.NewRequest(http.MethodOptions, "/v1/menu", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	req.Header.Set("Access-Control-Request-Method", "GET")
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Empty(t, resp.Header.Get("Access-Control-Allow-Origin"))
}

func TestMiddlewareChain_JWTPasses(t *testing.T) {
	app, cfg, signer := newTestAppWithSigner(t)
	token := signTestToken(t, signer, cfg.JWTIssuer, cfg.JWTAudience)

	req := httptest.NewRequest(http.MethodGet, "/v1/menu", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req)
	require.NoError(t, err)
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	require.Equal(t, http.StatusBadGateway, resp.StatusCode, "expected bad gateway because no real gRPC backend: %s", string(body))
}

func newTestApp(t *testing.T) *fiber.App {
	t.Helper()
	app, _, _ := newTestAppWithSigner(t)
	return app
}

func newTestAppWithSigner(t *testing.T) (*fiber.App, *config.Config, ed25519.PrivateKey) {
	t.Helper()
	mr := miniredis.RunT(t)
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	cfg := &config.Config{
		Port:           "8080",
		Environment:    "test",
		LogLevel:       "error",
		AllowedOrigins: []string{"http://localhost:5170"},
		RateLimitRPS:   100,
		RateLimitBurst: 100,
		JWTIssuer:      "astra-service",
		JWTAudience:    "astra-gateway",
		EdDSAPublicKey: pub,
		Services:       map[string]config.Downstream{},
	}

	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	app := New(cfg, redisClient)
	routes.Register(app, cfg, &noopChecker{}, nil)
	return app, cfg, priv
}

func signTestToken(t *testing.T, signer ed25519.PrivateKey, issuer, audience string) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, jwt.MapClaims{
		"iss": issuer,
		"aud": audience,
		"sub": "kiosk-1",
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	signed, err := token.SignedString(signer)
	require.NoError(t, err)
	return signed
}

type noopChecker struct{}

func (n *noopChecker) Check(_ context.Context) error { return nil }
