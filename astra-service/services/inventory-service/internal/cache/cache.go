// Package cache abstracts the Redis real-time inventory cache.
package cache

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/astra-systems/astra-service/services/inventory-service/internal/ledger"
	"github.com/redis/go-redis/v9"
)

// Cache stores and retrieves derived stock levels for fast reads.
type Cache interface {
	GetStock(ctx context.Context, storeID, itemID string) (ledger.Stock, bool, error)
	SetStock(ctx context.Context, stock ledger.Stock, ttl time.Duration) error
	Invalidate(ctx context.Context, storeID, itemID string) error
	Close() error
}

// RedisCache implements Cache on top of Redis.
type RedisCache struct {
	client *redis.Client
}

// NewRedisCache returns a Cache backed by client.
func NewRedisCache(client *redis.Client) *RedisCache {
	return &RedisCache{client: client}
}

func cacheKey(storeID, itemID string) string {
	return fmt.Sprintf("inventory:%s:%s", storeID, itemID)
}

// GetStock reads a cached stock level.
func (r *RedisCache) GetStock(ctx context.Context, storeID, itemID string) (ledger.Stock, bool, error) {
	key := cacheKey(storeID, itemID)
	m, err := r.client.HGetAll(ctx, key).Result()
	if err != nil {
		return ledger.Stock{}, false, fmt.Errorf("cache: hgetall: %w", err)
	}
	if len(m) == 0 {
		return ledger.Stock{}, false, nil
	}
	stock := ledger.Stock{
		StoreID:     storeID,
		ItemID:      itemID,
		InventoryID: m["inventory_id"],
		Location:    m["location"],
	}
	stock.QuantityAvailable = parseInt32(m["quantity_available"])
	stock.QuantityReserved = parseInt32(m["quantity_reserved"])
	stock.QuantityOnOrder = parseInt32(m["quantity_on_order"])
	stock.ReorderPoint = parseInt32(m["reorder_point"])
	stock.ReorderQuantity = parseInt32(m["reorder_quantity"])
	return stock, true, nil
}

// SetStock writes a stock level hash to Redis.
func (r *RedisCache) SetStock(ctx context.Context, stock ledger.Stock, ttl time.Duration) error {
	key := cacheKey(stock.StoreID, stock.ItemID)
	if err := r.client.HSet(ctx, key, map[string]string{
		"inventory_id":       stock.InventoryID,
		"quantity_available": strconv.FormatInt(int64(stock.QuantityAvailable), 10),
		"quantity_reserved":  strconv.FormatInt(int64(stock.QuantityReserved), 10),
		"quantity_on_order":  strconv.FormatInt(int64(stock.QuantityOnOrder), 10),
		"reorder_point":      strconv.FormatInt(int64(stock.ReorderPoint), 10),
		"reorder_quantity":   strconv.FormatInt(int64(stock.ReorderQuantity), 10),
		"location":           stock.Location,
	}).Err(); err != nil {
		return fmt.Errorf("cache: hset: %w", err)
	}
	if err := r.client.Expire(ctx, key, ttl).Err(); err != nil {
		return fmt.Errorf("cache: expire: %w", err)
	}
	return nil
}

// Invalidate removes a cached stock level.
func (r *RedisCache) Invalidate(ctx context.Context, storeID, itemID string) error {
	if err := r.client.Del(ctx, cacheKey(storeID, itemID)).Err(); err != nil {
		return fmt.Errorf("cache: del: %w", err)
	}
	return nil
}

// Close closes the Redis client.
func (r *RedisCache) Close() error {
	return r.client.Close()
}

func parseInt32(s string) int32 {
	v, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return 0
	}
	return int32(v)
}
