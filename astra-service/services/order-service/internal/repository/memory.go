package repository

import (
	"context"
	"sync"
	"time"
)

// MemoryRepository is an in-memory implementation of Repository for unit and
// integration-style tests that do not require PostgreSQL.
type MemoryRepository struct {
	mu            sync.RWMutex
	orders        map[string]*Order
	ordersByCart  map[string]*Order
	idempotency   map[string]*Order // key = scope + "|" + key
	outbox        []OutboxEvent
	orderSequence int64
}

// NewMemoryRepository returns a fresh in-memory repository.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		orders:       make(map[string]*Order),
		ordersByCart: make(map[string]*Order),
		idempotency:  make(map[string]*Order),
	}
}

// CreateOrder stores the order and appends the outbox event.
func (r *MemoryRepository) CreateOrder(ctx context.Context, order *Order, event OutboxEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.orders[order.OrderID]; exists {
		return ErrOrderConflict
	}
	if _, exists := r.ordersByCart[order.CartID]; exists {
		return ErrOrderConflict
	}

	r.orders[order.OrderID] = cloneOrder(order)
	r.ordersByCart[order.CartID] = r.orders[order.OrderID]
	if order.IdempotencyKey != "" {
		r.idempotency[order.CartID+"|"+order.IdempotencyKey] = r.orders[order.OrderID]
	}
	r.outbox = append(r.outbox, event)
	return nil
}

// GetOrder returns a copy of the order by ID.
func (r *MemoryRepository) GetOrder(ctx context.Context, orderID string) (*Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	order, ok := r.orders[orderID]
	if !ok {
		return nil, ErrOrderNotFound
	}
	return cloneOrder(order), nil
}

// GetOrderByCartID returns the order associated with a cart.
func (r *MemoryRepository) GetOrderByCartID(ctx context.Context, cartID string) (*Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	order, ok := r.ordersByCart[cartID]
	if !ok {
		return nil, ErrOrderNotFound
	}
	return cloneOrder(order), nil
}

// GetOrderByIdempotencyKey returns a previously stored order for a key.
func (r *MemoryRepository) GetOrderByIdempotencyKey(ctx context.Context, cartID, idempotencyKey string) (*Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	order, ok := r.idempotency[cartID+"|"+idempotencyKey]
	if !ok {
		return nil, ErrOrderNotFound
	}
	return cloneOrder(order), nil
}

// ListOrders returns a filtered, paginated list.
func (r *MemoryRepository) ListOrders(ctx context.Context, filter ListFilter) ([]*Order, int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 {
		filter.PageSize = 20
	}

	var matched []*Order
	for _, order := range r.orders {
		if filter.StoreID != "" && order.StoreID != filter.StoreID {
			continue
		}
		if filter.KioskID != "" && order.KioskID != filter.KioskID {
			continue
		}
		if filter.Status != "" && order.Status != filter.Status {
			continue
		}
		matched = append(matched, cloneOrder(order))
	}

	total := int64(len(matched))
	start := int((filter.Page - 1) * filter.PageSize)
	if start > len(matched) {
		start = len(matched)
	}
	end := start + int(filter.PageSize)
	if end > len(matched) {
		end = len(matched)
	}
	return matched[start:end], total, nil
}

// UpdateOrderStatus updates the order status and records an outbox event.
func (r *MemoryRepository) UpdateOrderStatus(ctx context.Context, orderID, status string, event OutboxEvent) (*Order, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	order, ok := r.orders[orderID]
	if !ok {
		return nil, ErrOrderNotFound
	}
	if order.Status == status {
		return cloneOrder(order), nil
	}
	order.Status = status
	now := time.Now().UTC()
	order.UpdatedAt = now
	if status == "cancelled" {
		order.CancelledAt = &now
	}
	r.outbox = append(r.outbox, event)
	return cloneOrder(order), nil
}

// MarkPaid transitions an order to paid.
func (r *MemoryRepository) MarkPaid(ctx context.Context, orderID string, paidAt time.Time, event OutboxEvent) (*Order, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	order, ok := r.orders[orderID]
	if !ok {
		return nil, ErrOrderNotFound
	}
	if order.Status == "paid" || order.Status == "fulfilled" {
		return cloneOrder(order), nil
	}
	if order.Status != "pending" {
		return nil, ErrInvalidStatus
	}
	order.Status = "paid"
	order.PaidAt = &paidAt
	order.UpdatedAt = time.Now().UTC()
	r.outbox = append(r.outbox, event)
	return cloneOrder(order), nil
}

// MarkFulfilled transitions an order to fulfilled.
func (r *MemoryRepository) MarkFulfilled(ctx context.Context, orderID, fulfilledBy string, fulfilledAt time.Time, event OutboxEvent) (*Order, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	order, ok := r.orders[orderID]
	if !ok {
		return nil, ErrOrderNotFound
	}
	if order.Status == "fulfilled" {
		return cloneOrder(order), nil
	}
	if order.Status != "paid" {
		return nil, ErrInvalidStatus
	}
	order.Status = "fulfilled"
	order.FulfilledAt = &fulfilledAt
	if fulfilledBy != "" {
		if order.Metadata == nil {
			order.Metadata = make(map[string]string)
		}
		order.Metadata["fulfilled_by"] = fulfilledBy
	}
	order.UpdatedAt = time.Now().UTC()
	r.outbox = append(r.outbox, event)
	return cloneOrder(order), nil
}

// Outbox returns a copy of recorded outbox events for test assertions.
func (r *MemoryRepository) Outbox() []OutboxEvent {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]OutboxEvent, len(r.outbox))
	copy(out, r.outbox)
	return out
}

func cloneOrder(src *Order) *Order {
	items := make([]OrderItem, len(src.Items))
	copy(items, src.Items)
	for i := range items {
		mods := make([]string, len(items[i].ModifierOptionIDs))
		copy(mods, items[i].ModifierOptionIDs)
		items[i].ModifierOptionIDs = mods
	}

	tax := make(map[string]string, len(src.TaxBreakdown))
	for k, v := range src.TaxBreakdown {
		tax[k] = v
	}

	meta := make(map[string]string, len(src.Metadata))
	for k, v := range src.Metadata {
		meta[k] = v
	}

	order := *src
	order.Items = items
	order.TaxBreakdown = tax
	order.Metadata = meta
	return &order
}
