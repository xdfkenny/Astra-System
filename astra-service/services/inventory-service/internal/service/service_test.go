package service

import (
	"context"
	"testing"
	"time"

	"github.com/astra-systems/astra-service/services/inventory-service/internal/cache"
	"github.com/astra-systems/astra-service/services/inventory-service/internal/ledger"
	"github.com/astra-systems/astra-service/services/inventory-service/internal/publisher"
	"github.com/astra-systems/astra-service/services/inventory-service/internal/repository"
	inventoryv1 "github.com/astra-systems/astra-service/proto/gen/go/inventory"
	"github.com/google/uuid"
)

func newTestService() (*Inventory, *repository.MemoryRepository, *publisher.MemoryPublisher) {
	repo := repository.NewMemoryRepository()
	c := cache.NewMemoryCache()
	pub := publisher.NewMemoryPublisher()
	svc := NewInventory(repo, c, pub, 5*time.Minute, 30*time.Second)
	return svc, repo, pub
}

func seedStock(repo *repository.MemoryRepository, storeID, itemID string, available, reserved int32) {
	repo.SeedInventory(ledger.Stock{
		StoreID:           storeID,
		ItemID:            itemID,
		InventoryID:       uuid.New().String(),
		QuantityAvailable: available,
		QuantityReserved:  reserved,
	})
	repo.SeedTransaction(ledger.Transaction{
		StoreID:         storeID,
		ItemID:          itemID,
		TransactionType: inventoryv1.InventoryTransactionType_INVENTORY_TRANSACTION_TYPE_RESTOCK,
		QuantityDelta:   int(available + reserved),
		RunningBalance:  int(available + reserved),
	})
}

func TestGetStock_ReturnsSeededLevel(t *testing.T) {
	svc, repo, _ := newTestService()
	storeID := uuid.New().String()
	itemID := uuid.New().String()
	seedStock(repo, storeID, itemID, 10, 2)

	stock, err := svc.GetStock(context.Background(), &inventoryv1.GetStockRequest{StoreId: storeID, ItemId: itemID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stock.QuantityAvailable != 10 || stock.QuantityReserved != 2 {
		t.Fatalf("expected available=10 reserved=2, got available=%d reserved=%d", stock.QuantityAvailable, stock.QuantityReserved)
	}
}

func TestReserveStock_Success(t *testing.T) {
	svc, repo, pub := newTestService()
	storeID := uuid.New().String()
	itemID := uuid.New().String()
	cartID := uuid.New().String()
	kioskID := uuid.New().String()
	seedStock(repo, storeID, itemID, 10, 0)

	stock, err := svc.ReserveStock(context.Background(), &inventoryv1.ReservationRequest{
		StoreId: storeID, ItemId: itemID, CartId: cartID, KioskId: kioskID, Quantity: 3,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stock.QuantityAvailable != 7 || stock.QuantityReserved != 3 {
		t.Fatalf("expected available=7 reserved=3, got available=%d reserved=%d", stock.QuantityAvailable, stock.QuantityReserved)
	}

	if len(pub.Events) != 1 {
		t.Fatalf("expected 1 published event, got %d", len(pub.Events))
	}
	if pub.Events[0].EventType != "InventoryReserved" {
		t.Fatalf("expected InventoryReserved, got %s", pub.Events[0].EventType)
	}
}

func TestReserveStock_InsufficientStock(t *testing.T) {
	svc, repo, _ := newTestService()
	storeID := uuid.New().String()
	itemID := uuid.New().String()
	seedStock(repo, storeID, itemID, 2, 0)

	_, err := svc.ReserveStock(context.Background(), &inventoryv1.ReservationRequest{
		StoreId: storeID, ItemId: itemID, CartId: uuid.New().String(), KioskId: uuid.New().String(), Quantity: 5,
	})
	if err == nil {
		t.Fatal("expected error for insufficient stock")
	}
}

func TestReleaseStock_Success(t *testing.T) {
	svc, repo, pub := newTestService()
	storeID := uuid.New().String()
	itemID := uuid.New().String()
	cartID := uuid.New().String()
	kioskID := uuid.New().String()
	seedStock(repo, storeID, itemID, 10, 0)

	if _, err := svc.ReserveStock(context.Background(), &inventoryv1.ReservationRequest{
		StoreId: storeID, ItemId: itemID, CartId: cartID, KioskId: kioskID, Quantity: 4,
	}); err != nil {
		t.Fatalf("reserve failed: %v", err)
	}

	stock, err := svc.ReleaseStock(context.Background(), &inventoryv1.ReleaseStockRequest{
		StoreId: storeID, ItemId: itemID, CartId: cartID, Quantity: 4, Reason: "removed from cart",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stock.QuantityAvailable != 10 || stock.QuantityReserved != 0 {
		t.Fatalf("expected available=10 reserved=0, got available=%d reserved=%d", stock.QuantityAvailable, stock.QuantityReserved)
	}

	if len(pub.Events) != 2 {
		t.Fatalf("expected 2 published events, got %d", len(pub.Events))
	}
	if pub.Events[1].EventType != "InventoryReleased" {
		t.Fatalf("expected InventoryReleased, got %s", pub.Events[1].EventType)
	}
}

func TestAdjustStock_Success(t *testing.T) {
	svc, repo, pub := newTestService()
	storeID := uuid.New().String()
	itemID := uuid.New().String()
	seedStock(repo, storeID, itemID, 10, 0)

	stock, err := svc.AdjustStock(context.Background(), &inventoryv1.AdjustStockRequest{
		StoreId: storeID, ItemId: itemID, QuantityDelta: 5,
		TransactionType: inventoryv1.InventoryTransactionType_INVENTORY_TRANSACTION_TYPE_RESTOCK,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stock.QuantityAvailable != 15 {
		t.Fatalf("expected available=15, got %d", stock.QuantityAvailable)
	}

	if len(pub.Events) != 1 || pub.Events[0].EventType != "InventoryAdjusted" {
		t.Fatalf("expected InventoryAdjusted event, got %+v", pub.Events)
	}
}

func TestReservationExpiry_WorkerReleasesExpired(t *testing.T) {
	svc, repo, _ := newTestService()
	storeID := uuid.New().String()
	itemID := uuid.New().String()
	cartID := uuid.New().String()
	kioskID := uuid.New().String()
	seedStock(repo, storeID, itemID, 10, 0)

	repo.SetNowFn(func() time.Time { return time.UnixMilli(1000) })
	if _, err := svc.ReserveStock(context.Background(), &inventoryv1.ReservationRequest{
		StoreId: storeID, ItemId: itemID, CartId: cartID, KioskId: kioskID, Quantity: 3,
	}); err != nil {
		t.Fatalf("reserve failed: %v", err)
	}

	// Move time forward past the default 5-minute TTL.
	nowMs := time.UnixMilli(1000).Add(6 * time.Minute).UnixMilli()
	repo.SetNowFn(func() time.Time { return time.UnixMilli(nowMs) })
	count, err := repo.ExpireReservations(context.Background(), nowMs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 expired reservation, got %d", count)
	}

	stock, err := svc.GetStock(context.Background(), &inventoryv1.GetStockRequest{StoreId: storeID, ItemId: itemID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stock.QuantityAvailable != 10 || stock.QuantityReserved != 0 {
		t.Fatalf("expected available=10 reserved=0, got available=%d reserved=%d", stock.QuantityAvailable, stock.QuantityReserved)
	}
}

func TestReserveStock_MissingFields(t *testing.T) {
	svc, _, _ := newTestService()
	_, err := svc.ReserveStock(context.Background(), &inventoryv1.ReservationRequest{})
	if err == nil {
		t.Fatal("expected error for missing fields")
	}
}

func TestAdjustStock_MissingTransactionType(t *testing.T) {
	svc, repo, _ := newTestService()
	storeID := uuid.New().String()
	itemID := uuid.New().String()
	seedStock(repo, storeID, itemID, 10, 0)

	_, err := svc.AdjustStock(context.Background(), &inventoryv1.AdjustStockRequest{
		StoreId: storeID, ItemId: itemID, QuantityDelta: 1,
	})
	if err == nil {
		t.Fatal("expected error for missing transaction type")
	}
}
