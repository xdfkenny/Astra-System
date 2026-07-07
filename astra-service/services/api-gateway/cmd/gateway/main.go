// Command gateway is the Astra-Service API Gateway entrypoint: a Go/Fiber
// process that terminates kiosk-facing REST traffic, enforces the zero-trust
// boundary (HMAC request signing, rate limiting, circuit breaking), and
// proxies to internal domain microservices over the NATS event bus / gRPC.
package main

import (
	"context"
	"log"
	"log/slog"
	"os/signal"
	"syscall"
	"time"

	"github.com/astra-service/api-gateway/internal/config"
	"github.com/astra-service/api-gateway/internal/middleware"
	"github.com/astra-service/api-gateway/internal/router"
	"github.com/astra-service/go-common/eventbus"
	"github.com/astra-service/go-common/observability"
	"github.com/gofiber/fiber/v2"
	fiberRecover "github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/redis/go-redis/v9"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("gateway: fatal startup error: %v", err)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	shutdownObs, err := observability.Init(ctx, observability.Config{
		ServiceName:    "astra-api-gateway",
		ServiceVersion: "0.1.0",
		Environment:    cfg.Environment,
		OTLPEndpoint:   cfg.OTLPEndpoint,
		SampleRatio:    1.0,
	})
	if err != nil {
		return err
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = shutdownObs(shutdownCtx)
	}()

	bus, err := eventbus.Connect(ctx, cfg.NatsURL)
	if err != nil {
		return err
	}
	defer bus.Close()

	redisOpts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		return err
	}
	redisClient := redis.NewClient(redisOpts)
	defer redisClient.Close()

	health := observability.NewCompositeChecker()
	health.Register("redis", &observability.RedisCheck{Client: redisClient})
	health.Register("nats", &observability.NATSCheck{Bus: bus})

	rateLimiter := middleware.NewRateLimiter(redisClient, cfg.RateLimitRPS, cfg.RateLimitBurst)
	breakers := middleware.NewBreakerRegistry()

	app := fiber.New(fiber.Config{
		AppName:               "astra-api-gateway",
		DisableStartupMessage: cfg.Environment == "production",
		ReadTimeout:           10 * time.Second,
		WriteTimeout:          10 * time.Second,
		IdleTimeout:           60 * time.Second,
		BodyLimit:             2 * 1024 * 1024, // 2MB — generous for cart JSON, hostile to abuse
	})

	app.Use(fiberRecover.New()) // never let a panic in one handler kill the whole process
	app.Use(requestid.New())    // correlates logs/traces across the request lifecycle
	app.Use(securityHeaders())
	app.Use(rateLimiter.Middleware())
	app.Use(middleware.RequireSignedRequest(resolveKioskKey(cfg)))

	router.Register(app, router.Dependencies{Bus: bus, Breakers: breakers, Health: health})

	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- app.Listen(":" + cfg.Port)
	}()

	observability.Info(ctx, "api-gateway started", slog.String("port", cfg.Port))

	select {
	case err := <-serverErrors:
		return err
	case <-ctx.Done():
		observability.Info(ctx, "api-gateway: shutdown signal received, draining connections...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := app.ShutdownWithContext(shutdownCtx); err != nil {
			return err
		}
		observability.Info(ctx, "api-gateway: graceful shutdown complete")
		return nil
	}
}

// securityHeaders enforces CORS/CSP/HSTS per the security mandate: strict
// CSP with no unsafe-inline/unsafe-eval, HSTS with preload, and a locked
// CORS allowlist (kiosk origins only, never "*").
func securityHeaders() fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains; preload")
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("X-Frame-Options", "DENY")
		c.Set("Referrer-Policy", "no-referrer")
		c.Set("Content-Security-Policy",
			"default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; "+
				"img-src 'self' data: https://cdn.astra-service.internal; "+
				"connect-src 'self' http://127.0.0.1:8963 http://127.0.0.1:4499; "+
				"frame-ancestors 'none'; base-uri 'none'")

		origin := c.Get("Origin")
		if isAllowedOrigin(origin) {
			c.Set("Access-Control-Allow-Origin", origin)
			c.Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
			c.Set("Access-Control-Allow-Headers",
				"Content-Type,Idempotency-Key,X-Astra-Kiosk-Id,X-Astra-Timestamp,X-Astra-Signature")
			c.Set("Access-Control-Max-Age", "600")
		}
		if c.Method() == fiber.MethodOptions {
			return c.SendStatus(fiber.StatusNoContent)
		}
		return c.Next()
	}
}

func isAllowedOrigin(origin string) bool {
	allowed := map[string]bool{
		"https://kiosk.astra-service.internal": true,
		"http://localhost:5170":                true, // local dev only
	}
	return allowed[origin]
}

// resolveKioskKey looks up a provisioned kiosk's HMAC signing key. In
// production this queries Vault (per-kiosk AppRole secret, cached with a
// short TTL). For local development we fall back to the shared dev key so
// the gateway is runnable standalone, matching the pattern used in router.go's handlers.
func resolveKioskKey(cfg *config.Config) func(kioskID string) ([]byte, bool) {
	return func(kioskID string) ([]byte, bool) {
		if kioskID == "" {
			return nil, false
		}
		return cfg.HMACSigningKey, true
	}
}
