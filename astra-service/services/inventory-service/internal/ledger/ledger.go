// Package ledger implements insert-only inventory arithmetic.
// Stock levels are derived from inventory_transactions rows; available
// quantity is physical stock minus active soft reservations.
package ledger

import (
	"fmt"

	inventoryv1 "github.com/astra-systems/astra-service/proto/gen/go/inventory"
)

// Transaction represents one row in the inventory_transactions ledger.
type Transaction struct {
	TransactionID   string
	StoreID         string
	ItemID          string
	TransactionType inventoryv1.InventoryTransactionType
	QuantityDelta   int
	RunningBalance  int
	ReferenceID     *string
	ReferenceType   *string
	Notes           *string
}

// Stock holds derived levels for a single store/item pair.
type Stock struct {
	StoreID           string
	ItemID            string
	InventoryID       string
	QuantityAvailable int32
	QuantityReserved  int32
	QuantityOnOrder   int32
	ReorderPoint      int32
	ReorderQuantity   int32
	Location          string
}

// SumTransactions returns the physical on-hand quantity from a list of
// ledger rows ordered from oldest to newest. The last row's running balance
// is the current physical stock if the rows are contiguous.
func SumTransactions(transactions []Transaction) int32 {
	if len(transactions) == 0 {
		return 0
	}
	last := transactions[len(transactions)-1]
	if last.RunningBalance < 0 {
		return 0
	}
	return int32(last.RunningBalance)
}

// Available computes quantity_available as physical stock minus active
// reservations. The result is never negative.
func Available(physical int32, reserved int32) int32 {
	available := physical - reserved
	if available < 0 {
		return 0
	}
	return available
}

// ApplyDelta checks that a ledger delta can be applied without driving the
// running balance negative and returns the new balance.
func ApplyDelta(current int32, delta int32, typ inventoryv1.InventoryTransactionType) (int32, error) {
	next := int(current) + int(delta)
	if next < 0 {
		return 0, fmt.Errorf("ledger: %s delta %d would drive balance to %d", typ.String(), delta, next)
	}
	return int32(next), nil
}
