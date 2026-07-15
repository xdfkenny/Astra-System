// Package service implements the InventoryService gRPC handlers and the
// background reservation expiry worker.
package service

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"

	"github.com/astra-service/go-common/observability"
	inventoryv1 "github.com/astra-systems/astra-service/proto/gen/go/inventory"
	"github.com/astra-systems/astra-service/services/inventory-service/internal/cache"
	"github.com/astra-systems/astra-service/services/inventory-service/internal/ledger"
	"github.com/astra-systems/astra-service/services/inventory-service/internal/publisher"
	"github.com/astra-systems/astra-service/services/inventory-service/internal/repository"
)

// Inventory implements inventoryv1.InventoryServiceServer.
type Inventory struct {
	inventoryv1.UnimplementedInventoryServiceServer

	repo     repository.Repository
	cache    cache.Cache
	pub      publisher.Publisher
	ttl      time.Duration
	sweep    time.Duration
	stopCh   chan struct{}
	stopOnce sync.Once
	wg       sync.WaitGroup
}

// NewInventory returns a gRPC service backed by repo, cache, and pub.
func NewInventory(repo repository.Repository, c cache.Cache, pub publisher.Publisher, ttl, sweep time.Duration) *Inventory {
	return &Inventory{
		repo:   repo,
		cache:  c,
		pub:    pub,
		ttl:    ttl,
		sweep:  sweep,
		stopCh: make(chan struct{}),
	}
}

// Start runs the background reservation expiry worker.
func (s *Inventory) Start(ctx context.Context) {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		ticker := time.NewTicker(s.sweep)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-s.stopCh:
				return
			case <-ticker.C:
				if _, err := s.repo.ExpireReservations(ctx, time.Now().UnixMilli()); err != nil {
					observability.Error(ctx, "inventory-service: reservation sweep error", err)
				}
			}
		}
	}()
}

// Stop gracefully shuts down background workers.
func (s *Inventory) Stop() {
	s.stopOnce.Do(func() {
		close(s.stopCh)
	})
	s.wg.Wait()
}

// GetStock returns the current stock level for a store/item pair.
func (s *Inventory) GetStock(ctx context.Context, req *inventoryv1.GetStockRequest) (*inventoryv1.StockLevel, error) {
	if req.StoreId == "" || req.ItemId == "" {
		return nil, fmt.Errorf("service: store_id and item_id are required")
	}

	if cached, ok, err := s.cache.GetStock(ctx, req.StoreId, req.ItemId); err == nil && ok {
		return toProtoStock(cached), nil
	}

	stock, err := s.repo.GetStock(ctx, req.StoreId, req.ItemId)
	if err != nil {
		return nil, fmt.Errorf("service: get stock: %w", err)
	}

	if err := s.cache.SetStock(ctx, stock, s.ttl); err != nil {
		observability.Error(ctx, "inventory-service: cache set error", err)
	}

	return toProtoStock(stock), nil
}

// ReserveStock attempts to soft-hold quantity units for a cart.
func (s *Inventory) ReserveStock(ctx context.Context, req *inventoryv1.ReservationRequest) (*inventoryv1.StockLevel, error) {
	if req.StoreId == "" || req.ItemId == "" || req.CartId == "" || req.KioskId == "" {
		return nil, fmt.Errorf("service: store_id, item_id, cart_id, and kiosk_id are required")
	}
	if req.Quantity <= 0 {
		return nil, fmt.Errorf("service: quantity must be positive")
	}

	reservationID, stock, err := s.repo.Reserve(ctx, req.StoreId, req.ItemId, req.CartId, req.KioskId, req.Quantity)
	if err != nil {
		return nil, fmt.Errorf("service: reserve stock: %w", err)
	}

	if err := s.cache.Invalidate(ctx, req.StoreId, req.ItemId); err != nil {
		observability.Error(ctx, "inventory-service: cache invalidate error", err)
	}

	expiresAtMs := time.Now().Add(s.ttl).UnixMilli()
	if req.ExpiresAtMs > 0 {
		expiresAtMs = req.ExpiresAtMs
	}
	if err := s.pub.Publish(ctx, publisher.Event{
		EventType:    "InventoryReserved",
		AggregateID:  req.CartId,
		OccurredAtMs: time.Now().UnixMilli(),
		Payload: map[string]any{
			"reservation_id": reservationID,
			"store_id":       req.StoreId,
			"item_id":        req.ItemId,
			"cart_id":        req.CartId,
			"kiosk_id":       req.KioskId,
			"quantity":       req.Quantity,
			"expires_at_ms":  expiresAtMs,
		},
	}); err != nil {
		observability.Error(ctx, "inventory-service: publish reserved event error", err)
	}

	return toProtoStock(stock), nil
}

// ReleaseStock releases a previous reservation for a cart item.
func (s *Inventory) ReleaseStock(ctx context.Context, req *inventoryv1.ReleaseStockRequest) (*inventoryv1.StockLevel, error) {
	if req.StoreId == "" || req.ItemId == "" || req.CartId == "" {
		return nil, fmt.Errorf("service: store_id, item_id, and cart_id are required")
	}

	stock, err := s.repo.Release(ctx, req.StoreId, req.ItemId, req.CartId)
	if err != nil {
		return nil, fmt.Errorf("service: release stock: %w", err)
	}

	if err := s.cache.Invalidate(ctx, req.StoreId, req.ItemId); err != nil {
		observability.Error(ctx, "inventory-service: cache invalidate error", err)
	}

	if err := s.pub.Publish(ctx, publisher.Event{
		EventType:    "InventoryReleased",
		AggregateID:  req.CartId,
		OccurredAtMs: time.Now().UnixMilli(),
		Payload: map[string]any{
			"store_id": req.StoreId,
			"item_id":  req.ItemId,
			"cart_id":  req.CartId,
			"quantity": req.Quantity,
			"reason":   req.Reason,
		},
	}); err != nil {
		observability.Error(ctx, "inventory-service: publish released event error", err)
	}

	return toProtoStock(stock), nil
}

// AdjustStock inserts a ledger delta and returns the updated stock level.
func (s *Inventory) AdjustStock(ctx context.Context, req *inventoryv1.AdjustStockRequest) (*inventoryv1.StockLevel, error) {
	if req.StoreId == "" || req.ItemId == "" {
		return nil, fmt.Errorf("service: store_id and item_id are required")
	}
	if req.TransactionType == inventoryv1.InventoryTransactionType_INVENTORY_TRANSACTION_TYPE_UNSPECIFIED {
		return nil, fmt.Errorf("service: transaction_type is required")
	}

	stock, err := s.repo.AdjustStock(ctx, req.StoreId, req.ItemId, req.QuantityDelta, req.TransactionType, req.ReferenceId, req.ReferenceType, req.Notes)
	if err != nil {
		return nil, fmt.Errorf("service: adjust stock: %w", err)
	}

	if err := s.cache.Invalidate(ctx, req.StoreId, req.ItemId); err != nil {
		observability.Error(ctx, "inventory-service: cache invalidate error", err)
	}

	if err := s.pub.Publish(ctx, publisher.Event{
		EventType:    "InventoryAdjusted",
		AggregateID:  req.ItemId,
		OccurredAtMs: time.Now().UnixMilli(),
		Payload: map[string]any{
			"store_id":         req.StoreId,
			"item_id":          req.ItemId,
			"quantity_delta":   req.QuantityDelta,
			"transaction_type": req.TransactionType.String(),
			"reference_id":     req.ReferenceId,
			"reference_type":   req.ReferenceType,
			"notes":            req.Notes,
		},
	}); err != nil {
		observability.Error(ctx, "inventory-service: publish adjusted event error", err)
	}

	return toProtoStock(stock), nil
}

// StreamStockUpdates streams stock updates until the client disconnects.
func (s *Inventory) StreamStockUpdates(req *inventoryv1.GetStockRequest, stream inventoryv1.InventoryService_StreamStockUpdatesServer) error {
	if req.StoreId == "" || req.ItemId == "" {
		return fmt.Errorf("service: store_id and item_id are required")
	}

	ctx := stream.Context()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			stock, err := s.GetStock(ctx, req)
			if err != nil {
				return err
			}
			update := &inventoryv1.StockUpdate{
				StoreId:           stock.StoreId,
				ItemId:            stock.ItemId,
				QuantityAvailable: stock.QuantityAvailable,
				QuantityReserved:  stock.QuantityReserved,
				UpdatedAt:         time.Now().UTC().Format(time.RFC3339),
			}
			if err := stream.Send(update); err != nil {
				if err == io.EOF {
					return nil
				}
				return fmt.Errorf("service: send stream update: %w", err)
			}
		}
	}
}

func toProtoStock(stock ledger.Stock) *inventoryv1.StockLevel {
	return &inventoryv1.StockLevel{
		InventoryId:       stock.InventoryID,
		StoreId:           stock.StoreID,
		ItemId:            stock.ItemID,
		QuantityAvailable: stock.QuantityAvailable,
		QuantityReserved:  stock.QuantityReserved,
		QuantityOnOrder:   stock.QuantityOnOrder,
		ReorderPoint:      stock.ReorderPoint,
		ReorderQuantity:   stock.ReorderQuantity,
		Location:          stock.Location,
		UpdatedAt:         time.Now().UTC().Format(time.RFC3339),
	}
}

// Ensure Inventory implements the generated interface at compile time.
var _ inventoryv1.InventoryServiceServer = (*Inventory)(nil)

// slogAny is a small helper so the service does not depend on slog specifics.
func slogAny(key string, value any) slog.Attr {
	return slog.Any(key, value)
}
