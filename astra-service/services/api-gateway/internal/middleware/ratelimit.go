// Package middleware holds all Fiber middleware for the API gateway:
// rate limiting, request signature verification, circuit breaking, and
// structured request logging with automatic PII redaction.
package middleware

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
)

// RateLimiter implements a distributed sliding-window token bucket backed
// by Redis, shared across all gateway replicas behind the load balancer.
// WHY Redis (not in-memory): a single misbehaving kiosk or malicious client
// hitting different gateway pods must be capped globally, not per-pod
// (which would allow RPS*replicaCount effective throughput).
type RateLimiter struct {
	client *redis.Client
	rps    int
	burst  int
}

func NewRateLimiter(client *redis.Client, rps, burst int) *RateLimiter {
	return &RateLimiter{client: client, rps: rps, burst: burst}
}

// tokenBucketScript atomically refills and consumes a token in one round
// trip using a Lua script — this avoids a check-then-act race between
// concurrent requests from the same client hitting different gateway pods.
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

// Middleware returns the Fiber handler. Rate-limit key is the kiosk ID
// (from the X-Astra-Kiosk-Id header set by every legitimate kiosk client)
// falling back to remote IP for unauthenticated/admin traffic.
func (r *RateLimiter) Middleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		key := c.Get("X-Astra-Kiosk-Id")
		if key == "" {
			key = c.IP()
		}
		redisKey := fmt.Sprintf("ratelimit:{%s}", key)

		ctx, cancel := context.WithTimeout(c.Context(), 200*time.Millisecond)
		defer cancel()

		result, err := r.client.Eval(ctx, tokenBucketScript, []string{redisKey},
			r.rps, r.burst, float64(time.Now().UnixMilli())/1000.0,
		).Int()
		if err != nil {
			// Fail OPEN on Redis unavailability: a rate limiter outage must never
			// take down checkout throughput store-wide. We log for alerting instead.
			c.Locals("ratelimit_degraded", true)
			return c.Next()
		}

		if result == 0 {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":       "rate_limited",
				"retry_after": "1",
			})
		}
		return c.Next()
	}
}

// RetryAfterHeader is a small helper some handlers use to advertise backoff.
func RetryAfterHeader(seconds int) string {
	return strconv.Itoa(seconds)
}
