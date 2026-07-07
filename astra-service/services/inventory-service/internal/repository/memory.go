package repository

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/astra-systems/astra-service/services/inventory-service/internal/ledger"
	"github.com/astra-systems/astra-service/services/inventory-service/internal/publisher"
	inventoryv1 "github.com/astra-systems/astra-service/proto/gen/go/inventory"
	"github.com/google/uuid"
)

// reservation models an active soft hold in the memory repository.
type reservation struct {
	reservationID string
	storeID       string
	itemID        string
	cartID        string
	kioskID       string
	quantity      int32
	expiresAtMs   int64
}

// MemoryRepository is an in-memory implementation of Repository for tests.
type MemoryRepository struct {
	mu           sync.RWMutex
	inventory    map[string]ledger.Stock
	reservations map[string]*reservation
	transactions []ledger.Transaction
	events       []publisher.Event
	nowFn        func() time.Time
}

// NewMemoryRepository returns a fresh in-memory repository.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		inventory:    make(map[string]ledger.Stock),
		reservations: make(map[string]*reservation),
		nowFn:        time.Now,
	}
}

// SeedInventory inserts or updates the base inventory record for a store/item.
func (m *MemoryRepository) SeedInventory(stock ledger.Stock) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := stock.StoreID + ":" + stock.ItemID
	m.inventory[key] = stock
}

// SeedTransaction appends a ledger row to set the physical balance.
func (m *MemoryRepository) SeedTransaction(tx ledger.Transaction) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.transactions = append(m.transactions, tx)
	key := tx.StoreID + ":" + tx.ItemID
	stock := m.inventory[key]
	stock.StoreID = tx.StoreID
	stock.ItemID = tx.ItemID
	stock.QuantityAvailable = ledger.Available(int32(tx.RunningBalance), stock.QuantityReserved)
	m.inventory[key] = stock
}

// GetStock returns the derived stock level.
func (m *MemoryRepository) GetStock(ctx context.Context, storeID, itemID string) (ledger.Stock, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	key := storeID + ":" + itemID
	stock, ok := m.inventory[key]
	if !ok {
		return ledger.Stock{}, fmt.Errorf("repository: inventory not found for store %s item %s", storeID, itemID)
	}
	return stock, nil
}

// Reserve creates a soft reservation and subtracts from available.
func (m *MemoryRepository) Reserve(ctx context.Context, storeID, itemID, cartID, kioskID string, quantity int32) (string, ledger.Stock, error) {
	if quantity <= 0 {
		return "", ledger.Stock{}, fmt.Errorf("repository: reserve quantity must be positive")
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	key := storeID + ":" + itemID
	stock, ok := m.inventory[key]
	if !ok {
		return "", ledger.Stock{}, fmt.Errorf("repository: inventory not found")
	}
	if stock.QuantityAvailable < quantity {
		return "", ledger.Stock{}, fmt.Errorf("repository: insufficient stock")
	}

	resID := uuid.New().String()
	expiresAtMs := m.nowFn().Add(5 * time.Minute).UnixMilli()
	createdAtMs := m.nowFn().UnixMilli()
	m.reservations[resID] = &reservation{
		reservationID: resID,
		storeID:       storeID,
		itemID:        itemID,
		cartID:        cartID,
		kioskID:       kioskID,
		quantity:      quantity,
		expiresAtMs:   expiresAtMs,
	}
	stock.QuantityReserved += quantity
	stock.QuantityAvailable = ledger.Available(stock.QuantityAvailable+stock.QuantityReserved-quantity, stock.QuantityReserved)
	m.inventory[key] = stock
	m.transactions = append(m.transactions, ledger.Transaction{
		StoreID: storeID, ItemID: itemID,
		TransactionType: inventoryv1.InventoryTransactionType_INVENTORY_TRANSACTION_TYPE_RESERVED,
		QuantityDelta:   -int(quantity), RunningBalance: int(stock.QuantityAvailable + stock.QuantityReserved),
	})
	m.events = append(m.events, publisher.Event{
		EventType:    "InventoryReserved",
		AggregateID:  cartID,
		OccurredAtMs: createdAtMs,
		Payload: map[string]any{
			"reservation_id": resID,
			"store_id":       storeID,
			"item_id":        itemID,
			"cart_id":        cartID,
			"kiosk_id":       kioskID,
			"quantity":       quantity,
			"expires_at_ms":  expiresAtMs,
		},
	})
	return resID, stock, nil
}

// Release removes a reservation for cartID/itemID and restores available stock.
func (m *MemoryRepository) Release(ctx context.Context, storeID, itemID, cartID string) (ledger.Stock, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := storeID + ":" + itemID
	stock := m.inventory[key]

	for id, r := range m.reservations {
		if r.storeID == storeID && r.itemID == itemID && r.cartID == cartID {
			stock.QuantityReserved -= r.quantity
			if stock.QuantityReserved < 0 {
				stock.QuantityReserved = 0
			}
			m.transactions = append(m.transactions, ledger.Transaction{
				StoreID: storeID, ItemID: itemID,
				TransactionType: inventoryv1.InventoryTransactionType_INVENTORY_TRANSACTION_TYPE_RELEASED,
				QuantityDelta:   int(r.quantity), RunningBalance: int(stock.QuantityAvailable + stock.QuantityReserved),
			})
			m.events = append(m.events, publisher.Event{
				EventType:    "InventoryReleased",
				AggregateID:  cartID,
				OccurredAtMs: m.nowFn().UnixMilli(),
				Payload: map[string]any{
					"reservation_id": r.reservationID,
					"store_id":       storeID,
					"item_id":        itemID,
					"cart_id":        cartID,
					"quantity":       r.quantity,
					"reason":         "explicit release",
				},
			})
			stock.QuantityAvailable = ledger.Available(stock.QuantityAvailable+stock.QuantityReserved+r.quantity, stock.QuantityReserved)
			m.inventory[key] = stock
			delete(m.reservations, id)
			break
		}
	}
	return stock, nil
}

// AdjustStock appends a ledger delta.
func (m *MemoryRepository) AdjustStock(ctx context.Context, storeID, itemID string, delta int32, typ inventoryv1.InventoryTransactionType, referenceID, referenceType, notes string) (ledger.Stock, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := storeID + ":" + itemID
	stock, ok := m.inventory[key]
	if !ok {
		return ledger.Stock{}, fmt.Errorf("repository: inventory not found")
	}
	physical := stock.QuantityAvailable + stock.QuantityReserved
	newPhysical, err := ledger.ApplyDelta(physical, delta, typ)
	if err != nil {
		return ledger.Stock{}, err
	}
	m.transactions = append(m.transactions, ledger.Transaction{
		StoreID: storeID, ItemID: itemID,
		TransactionType: typ, QuantityDelta: int(delta), RunningBalance: int(newPhysical),
	})
	m.events = append(m.events, publisher.Event{
		EventType:    "InventoryAdjusted",
		AggregateID:  itemID,
		OccurredAtMs: m.nowFn().UnixMilli(),
		Payload: map[string]any{
			"store_id":         storeID,
			"item_id":          itemID,
			"quantity_delta":   delta,
			"transaction_type": typ.String(),
			"reference_id":     referenceID,
			"reference_type":   referenceType,
			"notes":            notes,
		},
	})
	stock.QuantityAvailable = ledger.Available(newPhysical, stock.QuantityReserved)
	m.inventory[key] = stock
	return stock, nil
}

// ExpireReservations removes reservations whose expires_at_ms has passed.
func (m *MemoryRepository) ExpireReservations(ctx context.Context, nowMs int64) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var count int64
	for id, r := range m.reservations {
		if r.expiresAtMs < nowMs {
			key := r.storeID + ":" + r.itemID
			stock := m.inventory[key]
			stock.QuantityReserved -= r.quantity
			if stock.QuantityReserved < 0 {
				stock.QuantityReserved = 0
			}
			m.transactions = append(m.transactions, ledger.Transaction{
				StoreID: r.storeID, ItemID: r.itemID,
				TransactionType: inventoryv1.InventoryTransactionType_INVENTORY_TRANSACTION_TYPE_RELEASED,
				QuantityDelta:   int(r.quantity), RunningBalance: int(stock.QuantityAvailable + stock.QuantityReserved),
			})
			m.events = append(m.events, publisher.Event{
				EventType:    "InventoryReleased",
				AggregateID:  r.cartID,
				OccurredAtMs: nowMs,
				Payload: map[string]any{
					"reservation_id": r.reservationID,
					"store_id":       r.storeID,
					"item_id":        r.itemID,
					"cart_id":        r.cartID,
					"quantity":       r.quantity,
					"reason":         "ttl expiry",
				},
			})
			stock.QuantityAvailable = ledger.Available(stock.QuantityAvailable+stock.QuantityReserved+r.quantity, stock.QuantityReserved)
			m.inventory[key] = stock
			delete(m.reservations, id)
			count++
		}
	}
	return count, nil
}

// Close is a no-op.
func (m *MemoryRepository) Close() error {
	return nil
}

// SetNowFn is used by tests to control time.
func (m *MemoryRepository) SetNowFn(fn func() time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nowFn = fn
}

// Events returns the events captured by the repository.
func (m *MemoryRepository) Events() []publisher.Event {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]publisher.Event, len(m.events))
	copy(out, m.events)
	return out
}
