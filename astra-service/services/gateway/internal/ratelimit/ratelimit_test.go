package ratelimit

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/astra-systems/astra-service/services/gateway/internal/config"
	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestMiddleware_AllowsUnderBurst(t *testing.T) {
	mr := miniredis.RunT(t)
	defer mr.Close()

	app := fiber.New()
	cfg := &config.Config{RateLimitRPS: 10, RateLimitBurst: 5}
	app.Use(Middleware(cfg, redis.NewClient(&redis.Options{Addr: mr.Addr()})))
	app.Get("/test", func(c fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) })

	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode, "request %d", i)
	}
}

func TestMiddleware_BlocksOverBurst(t *testing.T) {
	mr := miniredis.RunT(t)
	defer mr.Close()

	app := fiber.New()
	cfg := &config.Config{RateLimitRPS: 10, RateLimitBurst: 2}
	app.Use(Middleware(cfg, redis.NewClient(&redis.Options{Addr: mr.Addr()})))
	app.Get("/test", func(c fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) })

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode, "request %d", i)
	}

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusTooManyRequests, resp.StatusCode)
	require.Equal(t, "1", resp.Header.Get("Retry-After"))
}

func TestMiddleware_FailOpenWhenRedisMissing(t *testing.T) {
	app := fiber.New()
	cfg := &config.Config{RateLimitRPS: 10, RateLimitBurst: 1}
	app.Use(Middleware(cfg, nil))
	app.Get("/test", func(c fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
}
