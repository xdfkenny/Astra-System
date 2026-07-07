package ledger

import (
	"testing"

	inventoryv1 "github.com/astra-systems/astra-service/proto/gen/go/inventory"
)

func TestSumTransactions_EmptyReturnsZero(t *testing.T) {
	if got := SumTransactions(nil); got != 0 {
		t.Fatalf("expected 0, got %d", got)
	}
}

func TestSumTransactions_UsesLastRunningBalance(t *testing.T) {
	rows := []Transaction{
		{TransactionType: inventoryv1.InventoryTransactionType_INVENTORY_TRANSACTION_TYPE_RESTOCK, QuantityDelta: 10, RunningBalance: 10},
		{TransactionType: inventoryv1.InventoryTransactionType_INVENTORY_TRANSACTION_TYPE_SALE, QuantityDelta: -3, RunningBalance: 7},
		{TransactionType: inventoryv1.InventoryTransactionType_INVENTORY_TRANSACTION_TYPE_ADJUSTMENT, QuantityDelta: 2, RunningBalance: 9},
	}
	if got := SumTransactions(rows); got != 9 {
		t.Fatalf("expected 9, got %d", got)
	}
}

func TestSumTransactions_NegativeBalanceClampedToZero(t *testing.T) {
	rows := []Transaction{
		{TransactionType: inventoryv1.InventoryTransactionType_INVENTORY_TRANSACTION_TYPE_SALE, QuantityDelta: -5, RunningBalance: -1},
	}
	if got := SumTransactions(rows); got != 0 {
		t.Fatalf("expected 0, got %d", got)
	}
}

func TestAvailable_ReservedSubtracted(t *testing.T) {
	if got := Available(10, 3); got != 7 {
		t.Fatalf("expected 7, got %d", got)
	}
}

func TestAvailable_NeverNegative(t *testing.T) {
	if got := Available(2, 5); got != 0 {
		t.Fatalf("expected 0, got %d", got)
	}
}

func TestApplyDelta_Success(t *testing.T) {
	got, err := ApplyDelta(10, -3, inventoryv1.InventoryTransactionType_INVENTORY_TRANSACTION_TYPE_SALE)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 7 {
		t.Fatalf("expected 7, got %d", got)
	}
}

func TestApplyDelta_RejectsNegativeBalance(t *testing.T) {
	_, err := ApplyDelta(2, -5, inventoryv1.InventoryTransactionType_INVENTORY_TRANSACTION_TYPE_SALE)
	if err == nil {
		t.Fatal("expected error for over-sale")
	}
}

func TestStock_Levels(t *testing.T) {
	s := Stock{
		StoreID:           "store-1",
		ItemID:            "item-1",
		InventoryID:       "inv-1",
		QuantityAvailable: 7,
		QuantityReserved:  3,
	}
	if s.QuantityAvailable != 7 || s.QuantityReserved != 3 {
		t.Fatalf("unexpected stock levels: %+v", s)
	}
}
