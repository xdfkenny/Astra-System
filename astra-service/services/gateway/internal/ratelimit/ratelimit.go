// Package ratelimit implements a distributed token-bucket rate limiter backed
// by Redis. A missing or degraded Redis instance fails open so that a cache
// outage does not become a gateway outage.
package ratelimit

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/astra-systems/astra-service/services/gateway/internal/auth"
	"github.com/astra-systems/astra-service/services/gateway/internal/config"
	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
)

var logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

const tokenBucketScript = `
local key = KEYS[1]
local rps = tonumber(ARGV[1])
local burst = tonumber(ARGV[2])
local now = tonumber(ARGV[3])

local bucket = redis.call("HMGET", key, "tokens", "ts")
local tokens = tonumber(bucket[1])
local ts = tonumber(bucket[2])

if tokens == nil then
  tokens = burst
  ts = now
end

local elapsed = math.max(0, now - ts)
tokens = math.min(burst, tokens + elapsed * rps)

local allowed = 0
if tokens >= 1 then
  allowed = 1
  tokens = tokens - 1
end

redis.call("HMSET", key, "tokens", tokens, "ts", now)
redis.call("EXPIRE", key, 60)

return allowed
`

// Middleware returns a Fiber handler that enforces a per-subject token bucket.
// The bucket key prefers X-Astra-Kiosk-Id, then the authenticated subject, then
// the client IP.
func Middleware(cfg *config.Config, client *redis.Client) fiber.Handler {
	script := redis.NewScript(tokenBucketScript)
	return func(c fiber.Ctx) error {
		if client == nil {
			return c.Next()
		}

		key := subjectKey(c)
		if key == "" {
			key = c.IP()
		}
		redisKey := fmt.Sprintf("ratelimit:{%s}", key)

		ctx, cancel := context.WithTimeout(c.Context(), 200*time.Millisecond)
		defer cancel()

		now := float64(time.Now().UnixMilli()) / 1000.0
		result, err := script.Run(ctx, client, []string{redisKey}, cfg.RateLimitRPS, cfg.RateLimitBurst, now).Int()
		if err != nil {
			logger.LogAttrs(ctx, slog.LevelWarn, "rate_limiter_redis_error",
				slog.String("error", err.Error()),
				slog.String("key", key),
			)
			c.Locals("ratelimit_degraded", true)
			return c.Next()
		}

		if result == 0 {
			c.Set("Retry-After", "1")
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":       "rate_limited",
				"retry_after": "1",
			})
		}
		return c.Next()
	}
}

func subjectKey(c fiber.Ctx) string {
	if kioskID := c.Get("X-Astra-Kiosk-Id"); kioskID != "" {
		return kioskID
	}
	if sub, ok := auth.SubjectFromContext(c); ok {
		return sub
	}
	return ""
}

// FormatRetryAfter returns the next whole-second retry value as a string.
func FormatRetryAfter(seconds int) string {
	return strconv.Itoa(seconds)
}
