package cache

import (
	"context"
	"sync"
	"time"

	"github.com/astra-systems/astra-service/services/inventory-service/internal/ledger"
)

// MemoryCache is a thread-safe in-memory Cache for tests.
type MemoryCache struct {
	mu    sync.RWMutex
	items map[string]ledger.Stock
}

// NewMemoryCache returns a fresh in-memory cache.
func NewMemoryCache() *MemoryCache {
	return &MemoryCache{items: make(map[string]ledger.Stock)}
}

// GetStock returns a cached stock level.
func (m *MemoryCache) GetStock(ctx context.Context, storeID, itemID string) (ledger.Stock, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	stock, ok := m.items[cacheKey(storeID, itemID)]
	return stock, ok, nil
}

// SetStock stores a stock level.
func (m *MemoryCache) SetStock(ctx context.Context, stock ledger.Stock, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items[cacheKey(stock.StoreID, stock.ItemID)] = stock
	return nil
}

// Invalidate removes a stock level from the cache.
func (m *MemoryCache) Invalidate(ctx context.Context, storeID, itemID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.items, cacheKey(storeID, itemID))
	return nil
}

// Close is a no-op.
func (m *MemoryCache) Close() error {
	return nil
}
