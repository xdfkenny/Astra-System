package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Cache wraps a Redis client with menu-specific serialization.
type Cache struct {
	client *redis.Client
	prefix string
}

// New creates a Cache backed by Redis.
func New(client *redis.Client) *Cache {
	return &Cache{client: client, prefix: "menu:"}
}

// Ping verifies the Redis connection.
func (c *Cache) Ping(ctx context.Context) error {
	if err := c.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("cache: ping: %w", err)
	}
	return nil
}

// GetMenu retrieves the cached menu for a store.
func (c *Cache) GetMenu(ctx context.Context, storeID string, out any) (bool, error) {
	data, err := c.client.Get(ctx, c.key("menu:"+storeID)).Bytes()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("cache: get menu: %w", err)
	}
	if err := json.Unmarshal(data, out); err != nil {
		return false, fmt.Errorf("cache: unmarshal menu: %w", err)
	}
	return true, nil
}

// SetMenu stores a menu for a store with the supplied TTL.
func (c *Cache) SetMenu(ctx context.Context, storeID string, value any, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("cache: marshal menu: %w", err)
	}
	if err := c.client.Set(ctx, c.key("menu:"+storeID), data, ttl).Err(); err != nil {
		return fmt.Errorf("cache: set menu: %w", err)
	}
	return nil
}

// InvalidateMenu removes cached menu and category data for a store.
func (c *Cache) InvalidateMenu(ctx context.Context, storeID string) error {
	keys := []string{
		c.key("menu:" + storeID),
		c.key("categories:" + storeID),
	}
	if err := c.client.Del(ctx, keys...).Err(); err != nil {
		return fmt.Errorf("cache: invalidate menu: %w", err)
	}
	return nil
}

// GetCategories retrieves cached categories for a store.
func (c *Cache) GetCategories(ctx context.Context, storeID string, out any) (bool, error) {
	data, err := c.client.Get(ctx, c.key("categories:"+storeID)).Bytes()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("cache: get categories: %w", err)
	}
	if err := json.Unmarshal(data, out); err != nil {
		return false, fmt.Errorf("cache: unmarshal categories: %w", err)
	}
	return true, nil
}

// SetCategories stores categories for a store with the supplied TTL.
func (c *Cache) SetCategories(ctx context.Context, storeID string, value any, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("cache: marshal categories: %w", err)
	}
	if err := c.client.Set(ctx, c.key("categories:"+storeID), data, ttl).Err(); err != nil {
		return fmt.Errorf("cache: set categories: %w", err)
	}
	return nil
}

func (c *Cache) key(s string) string {
	return c.prefix + s
}
