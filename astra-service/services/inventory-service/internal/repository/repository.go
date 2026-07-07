// Package repository defines the inventory persistence interface and its
// PostgreSQL ledger implementation.
package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/astra-service/go-common/outbox"
	"github.com/astra-systems/astra-service/services/inventory-service/internal/ledger"
	"github.com/astra-systems/astra-service/services/inventory-service/internal/publisher"
	inventoryv1 "github.com/astra-systems/astra-service/proto/gen/go/inventory"
	"github.com/google/uuid"
)

// Repository is the persistence contract satisfied by the production Postgres
// implementation and by the in-memory fake used in tests.
type Repository interface {
	// GetStock returns the derived stock level for a store/item pair.
	GetStock(ctx context.Context, storeID, itemID string) (ledger.Stock, error)

	// Reserve attempts to soft-hold quantity units for cartID. It returns the
	// reservation ID and the new stock level.
	Reserve(ctx context.Context, storeID, itemID, cartID, kioskID string, quantity int32) (string, ledger.Stock, error)

	// Release removes an active reservation for cartID/itemID and returns the
	// new stock level. If no reservation exists it returns the current level
	// without error.
	Release(ctx context.Context, storeID, itemID, cartID string) (ledger.Stock, error)

	// AdjustStock inserts a ledger delta and returns the new stock level.
	AdjustStock(ctx context.Context, storeID, itemID string, delta int32, typ inventoryv1.InventoryTransactionType, referenceID, referenceType, notes string) (ledger.Stock, error)

	// ExpireReservations releases reservations whose expires_at_ms is older
	// than nowMs and writes compensating ledger entries. It returns the number
	// of reservations expired.
	ExpireReservations(ctx context.Context, nowMs int64) (int64, error)

	// Close cleans up any resources held by the repository.
	Close() error
}

// PostgresRepository implements Repository on top of a ledger-style
// inventory_transactions table plus an inventory_reservations table for
// active soft holds.
type PostgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository returns a repository backed by db.
func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// Migrate creates the tables required by this repository if they do not exist.
func (r *PostgresRepository) Migrate(ctx context.Context) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS inventory (
			inventory_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			store_id UUID NOT NULL,
			item_id UUID NOT NULL,
			quantity_on_order INT NOT NULL DEFAULT 0,
			reorder_point INT NOT NULL DEFAULT 0,
			reorder_quantity INT NOT NULL DEFAULT 0,
			location TEXT,
			last_counted_at TIMESTAMPTZ,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			UNIQUE (store_id, item_id)
		)`,
		`CREATE TABLE IF NOT EXISTS inventory_transactions (
			transaction_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			store_id UUID NOT NULL,
			item_id UUID NOT NULL,
			transaction_type TEXT NOT NULL,
			quantity_delta INT NOT NULL,
			running_balance INT NOT NULL,
			reference_id UUID,
			reference_type TEXT,
			kiosk_id UUID,
			employee_id UUID,
			notes TEXT,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_inventory_transactions_store_item ON inventory_transactions(store_id, item_id, created_at)`,
		`CREATE TABLE IF NOT EXISTS inventory_reservations (
			reservation_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			store_id UUID NOT NULL,
			kiosk_id UUID NOT NULL,
			item_id UUID NOT NULL,
			cart_id UUID NOT NULL,
			quantity INT NOT NULL,
			expires_at_ms BIGINT NOT NULL,
			created_at_ms BIGINT NOT NULL,
			UNIQUE (store_id, item_id, cart_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_inventory_reservations_expiry ON inventory_reservations(expires_at_ms)`,
	}
	for _, s := range stmts {
		if _, err := r.db.ExecContext(ctx, s); err != nil {
			return fmt.Errorf("repository: migrate: %w", err)
		}
	}
	return nil
}

// GetStock derives the stock level from the ledger and active reservations.
func (r *PostgresRepository) GetStock(ctx context.Context, storeID, itemID string) (ledger.Stock, error) {
	storeUUID, itemUUID, err := parseStoreItem(storeID, itemID)
	if err != nil {
		return ledger.Stock{}, err
	}

	var invID uuid.UUID
	var onOrder, reorderPoint, reorderQty int32
	var location sql.NullString
	err = r.db.QueryRowContext(ctx, `
		SELECT inventory_id, quantity_on_order, reorder_point, reorder_quantity, location
		FROM inventory WHERE store_id = $1 AND item_id = $2`, storeUUID, itemUUID).Scan(
		&invID, &onOrder, &reorderPoint, &reorderQty, &location)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ledger.Stock{}, fmt.Errorf("repository: inventory not found for store %s item %s", storeID, itemID)
		}
		return ledger.Stock{}, fmt.Errorf("repository: select inventory: %w", err)
	}

	var runningBalance sql.NullInt32
	err = r.db.QueryRowContext(ctx, `
		SELECT running_balance FROM inventory_transactions
		WHERE store_id = $1 AND item_id = $2
		ORDER BY created_at DESC, transaction_id DESC LIMIT 1`, storeUUID, itemUUID).Scan(
		&runningBalance)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return ledger.Stock{}, fmt.Errorf("repository: select latest transaction: %w", err)
	}
	physical := runningBalance.Int32
	if !runningBalance.Valid {
		physical = 0
	}

	var reserved sql.NullInt32
	err = r.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(quantity), 0) FROM inventory_reservations
		WHERE store_id = $1 AND item_id = $2`, storeUUID, itemUUID).Scan(&reserved)
	if err != nil {
		return ledger.Stock{}, fmt.Errorf("repository: select reservations: %w", err)
	}

	loc := ""
	if location.Valid {
		loc = location.String
	}

	return ledger.Stock{
		StoreID:           storeID,
		ItemID:            itemID,
		InventoryID:       invID.String(),
		QuantityAvailable: ledger.Available(physical, reserved.Int32),
		QuantityReserved:  reserved.Int32,
		QuantityOnOrder:   onOrder,
		ReorderPoint:      reorderPoint,
		ReorderQuantity:   reorderQty,
		Location:          loc,
	}, nil
}

// Reserve atomically inserts a reservation row and a RESERVED ledger entry.
func (r *PostgresRepository) Reserve(ctx context.Context, storeID, itemID, cartID, kioskID string, quantity int32) (string, ledger.Stock, error) {
	if quantity <= 0 {
		return "", ledger.Stock{}, fmt.Errorf("repository: reserve quantity must be positive")
	}
	storeUUID, itemUUID, err := parseStoreItem(storeID, itemID)
	if err != nil {
		return "", ledger.Stock{}, err
	}
	cartUUID, err := uuid.Parse(cartID)
	if err != nil {
		return "", ledger.Stock{}, fmt.Errorf("repository: invalid cart id: %w", err)
	}
	kioskUUID, err := uuid.Parse(kioskID)
	if err != nil {
		return "", ledger.Stock{}, fmt.Errorf("repository: invalid kiosk id: %w", err)
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return "", ledger.Stock{}, fmt.Errorf("repository: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	stock, err := r.getStockWithinTx(ctx, tx, storeUUID, itemUUID)
	if err != nil {
		return "", ledger.Stock{}, err
	}
	if stock.QuantityAvailable < quantity {
		return "", ledger.Stock{}, fmt.Errorf("repository: insufficient stock (available %d, requested %d)", stock.QuantityAvailable, quantity)
	}

	reservationID := uuid.New()
	expiresAtMs := time.Now().Add(5 * time.Minute).UnixMilli()
	createdAtMs := time.Now().UnixMilli()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO inventory_reservations
			(reservation_id, store_id, kiosk_id, item_id, cart_id, quantity, expires_at_ms, created_at_ms)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (store_id, item_id, cart_id) DO UPDATE SET
			quantity = inventory_reservations.quantity + EXCLUDED.quantity,
			expires_at_ms = EXCLUDED.expires_at_ms`,
		reservationID, storeUUID, kioskUUID, itemUUID, cartUUID, quantity, expiresAtMs, createdAtMs)
	if err != nil {
		return "", ledger.Stock{}, fmt.Errorf("repository: insert reservation: %w", err)
	}

	if _, err := r.insertTransactionWithinTx(ctx, tx, storeUUID, itemUUID, inventoryv1.InventoryTransactionType_INVENTORY_TRANSACTION_TYPE_RESERVED, -int(quantity), &cartUUID, nil, nil); err != nil {
		return "", ledger.Stock{}, err
	}

	if err := r.insertOutboxEventWithinTx(ctx, tx, publisher.Event{
		EventType:    "InventoryReserved",
		AggregateID:  cartID,
		OccurredAtMs: createdAtMs,
		Payload: map[string]any{
			"reservation_id": reservationID.String(),
			"store_id":       storeID,
			"item_id":        itemID,
			"cart_id":        cartID,
			"kiosk_id":       kioskID,
			"quantity":       quantity,
			"expires_at_ms":  expiresAtMs,
		},
	}); err != nil {
		return "", ledger.Stock{}, err
	}

	if err := tx.Commit(); err != nil {
		return "", ledger.Stock{}, fmt.Errorf("repository: commit reserve: %w", err)
	}

	newStock, err := r.GetStock(ctx, storeID, itemID)
	if err != nil {
		return "", ledger.Stock{}, err
	}
	return reservationID.String(), newStock, nil
}

// Release removes a reservation for cartID/itemID and inserts a RELEASED entry.
func (r *PostgresRepository) Release(ctx context.Context, storeID, itemID, cartID string) (ledger.Stock, error) {
	storeUUID, itemUUID, err := parseStoreItem(storeID, itemID)
	if err != nil {
		return ledger.Stock{}, err
	}
	cartUUID, err := uuid.Parse(cartID)
	if err != nil {
		return ledger.Stock{}, fmt.Errorf("repository: invalid cart id: %w", err)
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return ledger.Stock{}, fmt.Errorf("repository: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	var quantity int
	var reservationID uuid.UUID
	err = tx.QueryRowContext(ctx, `
		DELETE FROM inventory_reservations
		WHERE store_id = $1 AND item_id = $2 AND cart_id = $3
		RETURNING reservation_id, quantity`, storeUUID, itemUUID, cartUUID).Scan(&reservationID, &quantity)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			if rbErr := tx.Rollback(); rbErr != nil {
				return ledger.Stock{}, fmt.Errorf("repository: rollback after missing reservation: %w", rbErr)
			}
			return r.GetStock(ctx, storeID, itemID)
		}
		return ledger.Stock{}, fmt.Errorf("repository: delete reservation: %w", err)
	}

	if _, err := r.insertTransactionWithinTx(ctx, tx, storeUUID, itemUUID, inventoryv1.InventoryTransactionType_INVENTORY_TRANSACTION_TYPE_RELEASED, quantity, &cartUUID, nil, nil); err != nil {
		return ledger.Stock{}, err
	}

	if err := r.insertOutboxEventWithinTx(ctx, tx, publisher.Event{
		EventType:    "InventoryReleased",
		AggregateID:  cartID,
		OccurredAtMs: time.Now().UnixMilli(),
		Payload: map[string]any{
			"reservation_id": reservationID.String(),
			"store_id":       storeID,
			"item_id":        itemID,
			"cart_id":        cartID,
			"quantity":       int32(quantity),
			"reason":         "explicit release",
		},
	}); err != nil {
		return ledger.Stock{}, err
	}

	if err := tx.Commit(); err != nil {
		return ledger.Stock{}, fmt.Errorf("repository: commit release: %w", err)
	}
	return r.GetStock(ctx, storeID, itemID)
}

// AdjustStock inserts a ledger delta and returns the new stock level.
func (r *PostgresRepository) AdjustStock(ctx context.Context, storeID, itemID string, delta int32, typ inventoryv1.InventoryTransactionType, referenceID, referenceType, notes string) (ledger.Stock, error) {
	storeUUID, itemUUID, err := parseStoreItem(storeID, itemID)
	if err != nil {
		return ledger.Stock{}, err
	}

	var refUUID *uuid.UUID
	if referenceID != "" {
		parsed, err := uuid.Parse(referenceID)
		if err != nil {
			return ledger.Stock{}, fmt.Errorf("repository: invalid reference id: %w", err)
		}
		refUUID = &parsed
	}

	var refType, notesPtr *string
	if referenceType != "" {
		refType = &referenceType
	}
	if notes != "" {
		notesPtr = &notes
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return ledger.Stock{}, fmt.Errorf("repository: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if _, err := r.insertTransactionWithinTx(ctx, tx, storeUUID, itemUUID, typ, int(delta), refUUID, refType, notesPtr); err != nil {
		return ledger.Stock{}, err
	}

	occurredAtMs := time.Now().UnixMilli()
	if err := r.insertOutboxEventWithinTx(ctx, tx, publisher.Event{
		EventType:    "InventoryAdjusted",
		AggregateID:  itemID,
		OccurredAtMs: occurredAtMs,
		Payload: map[string]any{
			"store_id":         storeID,
			"item_id":          itemID,
			"quantity_delta":   delta,
			"transaction_type": typ.String(),
			"reference_id":     referenceID,
			"reference_type":   referenceType,
			"notes":            notes,
		},
	}); err != nil {
		return ledger.Stock{}, err
	}

	if err := tx.Commit(); err != nil {
		return ledger.Stock{}, fmt.Errorf("repository: commit adjust: %w", err)
	}
	return r.GetStock(ctx, storeID, itemID)
}

// ExpireReservations removes expired reservations and writes RELEASED entries.
func (r *PostgresRepository) ExpireReservations(ctx context.Context, nowMs int64) (int64, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("repository: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	rows, err := tx.QueryContext(ctx, `
		DELETE FROM inventory_reservations
		WHERE expires_at_ms < $1
		RETURNING store_id, item_id, cart_id, reservation_id, quantity`, nowMs)
	if err != nil {
		return 0, fmt.Errorf("repository: delete expired reservations: %w", err)
	}
	defer rows.Close()

	var count int64
	for rows.Next() {
		var storeUUID, itemUUID, cartUUID, reservationID uuid.UUID
		var quantity int
		if err := rows.Scan(&storeUUID, &itemUUID, &cartUUID, &reservationID, &quantity); err != nil {
			return 0, fmt.Errorf("repository: scan expired reservation: %w", err)
		}
		notes := "ttl expiry"
		if _, err := r.insertTransactionWithinTx(ctx, tx, storeUUID, itemUUID, inventoryv1.InventoryTransactionType_INVENTORY_TRANSACTION_TYPE_RELEASED, quantity, &cartUUID, nil, &notes); err != nil {
			return 0, err
		}
		if err := r.insertOutboxEventWithinTx(ctx, tx, publisher.Event{
			EventType:    "InventoryReleased",
			AggregateID:  cartUUID.String(),
			OccurredAtMs: nowMs,
			Payload: map[string]any{
				"reservation_id": reservationID.String(),
				"store_id":       storeUUID.String(),
				"item_id":        itemUUID.String(),
				"cart_id":        cartUUID.String(),
				"quantity":       int32(quantity),
				"reason":         "ttl expiry",
			},
		}); err != nil {
			return 0, err
		}
		count++
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("repository: iterate expired rows: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("repository: commit expiry: %w", err)
	}
	return count, nil
}

// Close is a no-op for the Postgres repository.
func (r *PostgresRepository) Close() error {
	return nil
}

func (r *PostgresRepository) getStockWithinTx(ctx context.Context, tx *sql.Tx, storeUUID, itemUUID uuid.UUID) (ledger.Stock, error) {
	var invID uuid.UUID
	var onOrder, reorderPoint, reorderQty int32
	var location sql.NullString
	err := tx.QueryRowContext(ctx, `
		SELECT inventory_id, quantity_on_order, reorder_point, reorder_quantity, location
		FROM inventory WHERE store_id = $1 AND item_id = $2`, storeUUID, itemUUID).Scan(
		&invID, &onOrder, &reorderPoint, &reorderQty, &location)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ledger.Stock{}, fmt.Errorf("repository: inventory not found")
		}
		return ledger.Stock{}, fmt.Errorf("repository: select inventory: %w", err)
	}

	var runningBalance sql.NullInt32
	err = tx.QueryRowContext(ctx, `
		SELECT running_balance FROM inventory_transactions
		WHERE store_id = $1 AND item_id = $2
		ORDER BY created_at DESC, transaction_id DESC LIMIT 1`, storeUUID, itemUUID).Scan(&runningBalance)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return ledger.Stock{}, fmt.Errorf("repository: select latest transaction: %w", err)
	}
	physical := runningBalance.Int32
	if !runningBalance.Valid {
		physical = 0
	}

	var reserved sql.NullInt32
	err = tx.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(quantity), 0) FROM inventory_reservations
		WHERE store_id = $1 AND item_id = $2`, storeUUID, itemUUID).Scan(&reserved)
	if err != nil {
		return ledger.Stock{}, fmt.Errorf("repository: select reservations: %w", err)
	}

	loc := ""
	if location.Valid {
		loc = location.String
	}

	return ledger.Stock{
		InventoryID:       invID.String(),
		QuantityAvailable: ledger.Available(physical, reserved.Int32),
		QuantityReserved:  reserved.Int32,
		QuantityOnOrder:   onOrder,
		ReorderPoint:      reorderPoint,
		ReorderQuantity:   reorderQty,
		Location:          loc,
	}, nil
}

func (r *PostgresRepository) insertTransactionWithinTx(ctx context.Context, tx *sql.Tx, storeUUID, itemUUID uuid.UUID, typ inventoryv1.InventoryTransactionType, delta int, referenceID *uuid.UUID, referenceType, notes *string) (uuid.UUID, error) {
	var previous sql.NullInt32
	err := tx.QueryRowContext(ctx, `
		SELECT running_balance FROM inventory_transactions
		WHERE store_id = $1 AND item_id = $2
		ORDER BY created_at DESC, transaction_id DESC LIMIT 1`, storeUUID, itemUUID).Scan(&previous)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return uuid.Nil, fmt.Errorf("repository: select previous balance: %w", err)
	}
	start := int32(0)
	if previous.Valid {
		start = previous.Int32
	}

	newBalance, err := ledger.ApplyDelta(start, int32(delta), typ)
	if err != nil {
		return uuid.Nil, fmt.Errorf("repository: %w", err)
	}

	transactionID := uuid.New()
	_, err = tx.ExecContext(ctx, `
		INSERT INTO inventory_transactions
			(transaction_id, store_id, item_id, transaction_type, quantity_delta, running_balance, reference_id, reference_type, notes, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, now())`,
		transactionID, storeUUID, itemUUID, typ.String(), delta, newBalance, referenceID, referenceType, notes)
	if err != nil {
		return uuid.Nil, fmt.Errorf("repository: insert transaction: %w", err)
	}
	return transactionID, nil
}

func (r *PostgresRepository) insertOutboxEventWithinTx(ctx context.Context, tx *sql.Tx, event publisher.Event) error {
	payload, err := json.Marshal(event.Payload)
	if err != nil {
		return fmt.Errorf("repository: marshal outbox payload: %w", err)
	}
	if event.EventID == "" {
		event.EventID = uuid.New().String()
	}
	return outbox.InsertWithinTx(ctx, tx, outbox.Entry{
		EventID:       event.EventID,
		AggregateType: "inventory",
		AggregateID:   event.AggregateID,
		EventType:     event.EventType,
		Payload:       payload,
		OccurredAtMs:  event.OccurredAtMs,
	})
}

func parseStoreItem(storeID, itemID string) (uuid.UUID, uuid.UUID, error) {
	storeUUID, err := uuid.Parse(storeID)
	if err != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("repository: invalid store id: %w", err)
	}
	itemUUID, err := uuid.Parse(itemID)
	if err != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("repository: invalid item id: %w", err)
	}
	return storeUUID, itemUUID, nil
}
