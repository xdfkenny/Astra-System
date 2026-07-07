// Package server assembles the Fiber v3 application and its middleware chain.
// The chain order is:
//
//	RequestID → structured JSON Logger → Recover → strict whitelist CORS →
//	Redis-backed token-bucket RateLimit → JWT EdDSA/RS256 Auth →
//	Prometheus Metrics → Handler.
package server

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/astra-systems/astra-service/services/gateway/internal/auth"
	"github.com/astra-systems/astra-service/services/gateway/internal/config"
	"github.com/astra-systems/astra-service/services/gateway/internal/metrics"
	"github.com/astra-systems/astra-service/services/gateway/internal/ratelimit"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	"github.com/redis/go-redis/v9"
)

var logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

// New creates a Fiber application with the full gateway middleware chain.
func New(cfg *config.Config, redisClient *redis.Client) *fiber.App {
	level, err := slogLevel(cfg.LogLevel)
	if err != nil {
		level = slog.LevelInfo
	}
	logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))

	app := fiber.New(fiber.Config{
		AppName:      "astra-gateway",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
		BodyLimit:    2 * 1024 * 1024,
		ErrorHandler: func(c fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if fiberErr, ok := err.(*fiber.Error); ok {
				code = fiberErr.Code
			}
			return c.Status(code).JSON(fiber.Map{"error": err.Error()})
		},
	})

	// 1. Request ID.
	app.Use(requestid.New())
	// 2. Structured JSON logger.
	app.Use(structuredLogger())
	// 3. Recover from panics.
	app.Use(recover.New())
	// 4. Strict CORS whitelist.
	app.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.AllowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "Authorization", "Idempotency-Key", "X-Request-ID", "X-Astra-Kiosk-Id"},
		ExposeHeaders:    []string{"X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           600,
	}))
	// 5. Redis-backed token-bucket rate limiter.
	app.Use(ratelimit.Middleware(cfg, redisClient))
	// 6. JWT EdDSA auth with RS256 fallback.
	app.Use(auth.Middleware(cfg))
	// 7. Prometheus metrics.
	app.Use(metrics.Middleware())

	return app
}

func structuredLogger() fiber.Handler {
	return func(c fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		duration := time.Since(start)

		status := c.Response().StatusCode()
		if err != nil {
			if fiberErr, ok := err.(*fiber.Error); ok {
				status = fiberErr.Code
			} else if status == fiber.StatusOK {
				status = fiber.StatusInternalServerError
			}
		}

		requestID := c.Get("X-Request-ID")
		if requestID == "" {
			requestID = "-"
		}

		logger.LogAttrs(c.Context(), slog.LevelInfo, "http_request",
			slog.String("request_id", requestID),
			slog.String("method", c.Method()),
			slog.String("path", c.Path()),
			slog.Int("status", status),
			slog.Duration("duration", duration),
			slog.String("ip", c.IP()),
			slog.String("error", errString(err)),
		)
		return err
	}
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func slogLevel(level string) (slog.Level, error) {
	switch level {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, fmt.Errorf("unknown log level %q", level)
	}
}
