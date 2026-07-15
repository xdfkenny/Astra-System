// Package cache provides a Redis-backed session cache for active carts. The
// key format is "cart:{lane_id}:{session_id}" with a configurable TTL.
package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/astra-systems/astra-service/services/cart-service/internal/cart"
	"github.com/redis/go-redis/v9"
)

// CartCache stores serialized cart snapshots indexed by lane and session.
type CartCache struct {
	client     *redis.Client
	defaultTTL time.Duration
}

// NewCartCache creates a cache using the supplied Redis client.
func NewCartCache(client *redis.Client, ttl time.Duration) *CartCache {
	return &CartCache{
		client:     client,
		defaultTTL: ttl,
	}
}

// Key returns the canonical Redis key for a lane/session pair.
func Key(laneID, sessionID string) string {
	return fmt.Sprintf("cart:%s:%s", laneID, sessionID)
}

// Get retrieves a cart from the cache.
func (c *CartCache) Get(ctx context.Context, laneID, sessionID string) (*cart.Cart, error) {
	data, err := c.client.Get(ctx, Key(laneID, sessionID)).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("cache: get: %w", err)
	}

	var cached cart.Cart
	if err := json.Unmarshal([]byte(data), &cached); err != nil {
		return nil, fmt.Errorf("cache: unmarshal: %w", err)
	}
	return &cached, nil
}

// Set stores a cart in the cache with the configured TTL.
func (c *CartCache) Set(ctx context.Context, laneID, sessionID string, crt *cart.Cart) error {
	if crt == nil {
		return c.client.Del(ctx, Key(laneID, sessionID)).Err()
	}
	data, err := json.Marshal(crt)
	if err != nil {
		return fmt.Errorf("cache: marshal: %w", err)
	}
	if err := c.client.Set(ctx, Key(laneID, sessionID), data, c.defaultTTL).Err(); err != nil {
		return fmt.Errorf("cache: set: %w", err)
	}
	return nil
}

// Delete removes a cached cart.
func (c *CartCache) Delete(ctx context.Context, laneID, sessionID string) error {
	if err := c.client.Del(ctx, Key(laneID, sessionID)).Err(); err != nil {
		return fmt.Errorf("cache: delete: %w", err)
	}
	return nil
}

// Ping verifies connectivity to Redis.
func (c *CartCache) Ping(ctx context.Context) error {
	if err := c.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("cache: ping: %w", err)
	}
	return nil
}

// Close closes the underlying Redis client.
func (c *CartCache) Close() error {
	return c.client.Close()
}

// Ensure CartCache implements the TTL contract exposed by tests.
var _ interface {
	Get(context.Context, string, string) (*cart.Cart, error)
} = (*CartCache)(nil)

// SessionTTL returns the configured default TTL.
func (c *CartCache) SessionTTL() time.Duration {
	return c.defaultTTL
}
