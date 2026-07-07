// Package repository implements persistence for the Order aggregate using
// PostgreSQL and the transactional outbox pattern. Every state-mutating
// operation writes the domain change and an outbox event in the same
// transaction so the database and event stream can never diverge.
package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/astra-service/go-common/outbox"
)

// Common domain errors returned by the repository.
var (
	ErrOrderNotFound  = errors.New("repository: order not found")
	ErrOrderConflict  = errors.New("repository: order already exists")
	ErrInvalidStatus  = errors.New("repository: invalid status transition")
)

// OrderItem is a denormalized snapshot of a cart line at order time.
type OrderItem struct {
	OrderItemID        string
	OrderID            string
	ItemID             string
	NameSnapshot       string
	PriceCentsSnapshot int64
	Quantity           int32
	ModifierOptionIDs  []string
	LineTotalCents     int64
	CreatedAt          time.Time
}

// Order is the aggregate root persisted by this repository.
type Order struct {
	OrderID        string
	StoreID        string
	KioskID        string
	CartID         string
	OrderNumber    string
	Status         string
	SubtotalCents  int64
	TaxCents       int64
	DiscountCents  int64
	TotalCents     int64
	Currency       string
	Items          []OrderItem
	TaxBreakdown   map[string]string
	Metadata       map[string]string
	IdempotencyKey string
	PaidAt         *time.Time
	FulfilledAt    *time.Time
	CancelledAt    *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// OutboxEvent carries the event that must be written atomically with the
// domain change.
type OutboxEvent struct {
	EventID      string
	EventType    string
	Payload      []byte
	OccurredAtMs int64
}

// Repository is the persistence contract for orders. Implementations must
// guarantee that domain writes and outbox events are committed atomically.
type Repository interface {
	CreateOrder(ctx context.Context, order *Order, event OutboxEvent) error
	GetOrder(ctx context.Context, orderID string) (*Order, error)
	GetOrderByCartID(ctx context.Context, cartID string) (*Order, error)
	GetOrderByIdempotencyKey(ctx context.Context, cartID, idempotencyKey string) (*Order, error)
	ListOrders(ctx context.Context, filter ListFilter) ([]*Order, int64, error)
	UpdateOrderStatus(ctx context.Context, orderID, status string, event OutboxEvent) (*Order, error)
	MarkPaid(ctx context.Context, orderID string, paidAt time.Time, event OutboxEvent) (*Order, error)
	MarkFulfilled(ctx context.Context, orderID, fulfilledBy string, fulfilledAt time.Time, event OutboxEvent) (*Order, error)
}

// ListFilter controls pagination and filtering for ListOrders.
type ListFilter struct {
	StoreID  string
	KioskID  string
	Status   string
	Page     int32
	PageSize int32
}

// PostgresRepository is the production implementation backed by a PostgreSQL
// pool/connection. It uses raw SQL so every query and transaction boundary is
// explicit.
type PostgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository returns a repository backed by the supplied *sql.DB.
func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// CreateOrder inserts a new order, its line items, and an outbox event in one
// transaction. The cart_id unique index prevents duplicate orders from a single
// cart, and the idempotency_keys unique index prevents duplicate processing
// keyed by caller-supplied idempotency keys.
func (r *PostgresRepository) CreateOrder(ctx context.Context, order *Order, event OutboxEvent) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("repository: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // safe no-op after Commit

	itemsJSON, err := json.Marshal(order.Items)
	if err != nil {
		return fmt.Errorf("repository: marshal items: %w", err)
	}

	taxJSON, err := json.Marshal(order.TaxBreakdown)
	if err != nil {
		return fmt.Errorf("repository: marshal tax breakdown: %w", err)
	}

	metaJSON, err := json.Marshal(order.Metadata)
	if err != nil {
		return fmt.Errorf("repository: marshal metadata: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO orders (
			order_id, store_id, kiosk_id, cart_id, order_number, status,
			subtotal_cents, tax_cents, discount_cents, total_cents, currency,
			items_json, tax_breakdown_json, metadata, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)`,
		order.OrderID, order.StoreID, order.KioskID, order.CartID, order.OrderNumber,
		order.Status, order.SubtotalCents, order.TaxCents, order.DiscountCents,
		order.TotalCents, order.Currency, itemsJSON, taxJSON, metaJSON,
		order.CreatedAt, order.UpdatedAt,
	); err != nil {
		return fmt.Errorf("repository: insert order: %w", err)
	}

	for i := range order.Items {
		item := &order.Items[i]
		modifierJSON, err := json.Marshal(item.ModifierOptionIDs)
		if err != nil {
			return fmt.Errorf("repository: marshal modifier ids: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO order_items (
				order_item_id, order_id, item_id, name_snapshot, price_cents_snapshot,
				quantity, modifier_option_ids, line_total_cents, created_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
			item.OrderItemID, order.OrderID, item.ItemID, item.NameSnapshot,
			item.PriceCentsSnapshot, item.Quantity, modifierJSON,
			item.LineTotalCents, item.CreatedAt,
		); err != nil {
			return fmt.Errorf("repository: insert order item: %w", err)
		}
	}

	if order.IdempotencyKey != "" {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO idempotency_keys (key, scope, order_id, created_at)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (scope, key) DO NOTHING`,
			order.IdempotencyKey, order.CartID, order.OrderID, order.CreatedAt,
		); err != nil {
			return fmt.Errorf("repository: insert idempotency key: %w", err)
		}
	}

	if err := outbox.InsertWithinTx(ctx, tx, outbox.Entry{
		EventID:       event.EventID,
		AggregateType: "order",
		AggregateID:   order.OrderID,
		EventType:     event.EventType,
		Payload:       event.Payload,
		OccurredAtMs:  event.OccurredAtMs,
	}); err != nil {
		return fmt.Errorf("repository: insert outbox: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("repository: commit: %w", err)
	}
	return nil
}

// GetOrder loads an order by primary key, including its line items.
func (r *PostgresRepository) GetOrder(ctx context.Context, orderID string) (*Order, error) {
	order, err := r.queryOrder(ctx, `SELECT order_id, store_id, kiosk_id, cart_id, order_number, status,
		subtotal_cents, tax_cents, discount_cents, total_cents, currency,
		items_json, tax_breakdown_json, metadata, paid_at, fulfilled_at, cancelled_at, created_at, updated_at
		FROM orders WHERE order_id = $1`, orderID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrOrderNotFound
		}
		return nil, err
	}
	return order, nil
}

// GetOrderByCartID loads the order created from a given cart.
func (r *PostgresRepository) GetOrderByCartID(ctx context.Context, cartID string) (*Order, error) {
	order, err := r.queryOrder(ctx, `SELECT order_id, store_id, kiosk_id, cart_id, order_number, status,
		subtotal_cents, tax_cents, discount_cents, total_cents, currency,
		items_json, tax_breakdown_json, metadata, paid_at, fulfilled_at, cancelled_at, created_at, updated_at
		FROM orders WHERE cart_id = $1`, cartID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrOrderNotFound
		}
		return nil, err
	}
	return order, nil
}

// GetOrderByIdempotencyKey returns a previously created order for a given
// caller-supplied idempotency key scoped to a cart.
func (r *PostgresRepository) GetOrderByIdempotencyKey(ctx context.Context, cartID, idempotencyKey string) (*Order, error) {
	var orderID string
	if err := r.db.QueryRowContext(ctx, `
		SELECT order_id FROM idempotency_keys WHERE scope = $1 AND key = $2`,
		cartID, idempotencyKey,
	).Scan(&orderID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrOrderNotFound
		}
		return nil, fmt.Errorf("repository: lookup idempotency key: %w", err)
	}
	return r.GetOrder(ctx, orderID)
}

// ListOrders returns a paginated list of orders with total count.
func (r *PostgresRepository) ListOrders(ctx context.Context, filter ListFilter) ([]*Order, int64, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 {
		filter.PageSize = 20
	}
	if filter.PageSize > 100 {
		filter.PageSize = 100
	}
	offset := (filter.Page - 1) * filter.PageSize

	args := []any{filter.PageSize, offset}
	where := "WHERE 1=1"
	if filter.StoreID != "" {
		where += fmt.Sprintf(" AND store_id = $%d", len(args)+1)
		args = append(args, filter.StoreID)
	}
	if filter.KioskID != "" {
		where += fmt.Sprintf(" AND kiosk_id = $%d", len(args)+1)
		args = append(args, filter.KioskID)
	}
	if filter.Status != "" {
		where += fmt.Sprintf(" AND status = $%d", len(args)+1)
		args = append(args, filter.Status)
	}

	countQuery := "SELECT COUNT(*) FROM orders " + where
	var total int64
	if err := r.db.QueryRowContext(ctx, countQuery, args[2:]...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("repository: count orders: %w", err)
	}

	query := `SELECT order_id, store_id, kiosk_id, cart_id, order_number, status,
		subtotal_cents, tax_cents, discount_cents, total_cents, currency,
		items_json, tax_breakdown_json, metadata, paid_at, fulfilled_at, cancelled_at, created_at, updated_at
		FROM orders ` + where + ` ORDER BY created_at DESC LIMIT $1 OFFSET $2`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("repository: query orders: %w", err)
	}
	defer rows.Close()

	var orders []*Order
	for rows.Next() {
		order, err := scanOrder(rows)
		if err != nil {
			return nil, 0, err
		}
		orders = append(orders, order)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("repository: iterate orders: %w", err)
	}
	return orders, total, nil
}

// UpdateOrderStatus performs an arbitrary status transition. It emits an
// outbox event only when the status actually changes.
func (r *PostgresRepository) UpdateOrderStatus(ctx context.Context, orderID, status string, event OutboxEvent) (*Order, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("repository: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	var previous string
	if err := tx.QueryRowContext(ctx, `
		SELECT status FROM orders WHERE order_id = $1 FOR UPDATE`, orderID,
	).Scan(&previous); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrOrderNotFound
		}
		return nil, fmt.Errorf("repository: select order: %w", err)
	}

	if previous == status {
		if err := tx.Commit(); err != nil {
			return nil, fmt.Errorf("repository: commit: %w", err)
		}
		return r.GetOrder(ctx, orderID)
	}

	var cancelledAt interface{}
	updatedAt := time.Now().UTC()
	if status == "cancelled" {
		t := updatedAt
		cancelledAt = &t
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE orders SET status = $1, cancelled_at = $2, updated_at = $3
		WHERE order_id = $4`,
		status, cancelledAt, updatedAt, orderID,
	); err != nil {
		return nil, fmt.Errorf("repository: update status: %w", err)
	}

	if err := outbox.InsertWithinTx(ctx, tx, outbox.Entry{
		EventID:       event.EventID,
		AggregateType: "order",
		AggregateID:   orderID,
		EventType:     event.EventType,
		Payload:       event.Payload,
		OccurredAtMs:  event.OccurredAtMs,
	}); err != nil {
		return nil, fmt.Errorf("repository: insert outbox: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("repository: commit: %w", err)
	}
	return r.GetOrder(ctx, orderID)
}

// MarkPaid transitions a pending order to paid. It is idempotent: if the
// order is already paid the call succeeds and no duplicate event is emitted.
func (r *PostgresRepository) MarkPaid(ctx context.Context, orderID string, paidAt time.Time, event OutboxEvent) (*Order, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("repository: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	var status string
	if err := tx.QueryRowContext(ctx, `
		SELECT status FROM orders WHERE order_id = $1 FOR UPDATE`, orderID,
	).Scan(&status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrOrderNotFound
		}
		return nil, fmt.Errorf("repository: select order: %w", err)
	}

	if status == "paid" || status == "fulfilled" {
		if err := tx.Commit(); err != nil {
			return nil, fmt.Errorf("repository: commit: %w", err)
		}
		return r.GetOrder(ctx, orderID)
	}

	if status != "pending" {
		return nil, fmt.Errorf("repository: cannot mark %s order as paid: %w", status, ErrInvalidStatus)
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE orders SET status = $1, paid_at = $2, updated_at = $3
		WHERE order_id = $4`,
		"paid", paidAt, time.Now().UTC(), orderID,
	); err != nil {
		return nil, fmt.Errorf("repository: mark paid: %w", err)
	}

	if err := outbox.InsertWithinTx(ctx, tx, outbox.Entry{
		EventID:       event.EventID,
		AggregateType: "order",
		AggregateID:   orderID,
		EventType:     event.EventType,
		Payload:       event.Payload,
		OccurredAtMs:  event.OccurredAtMs,
	}); err != nil {
		return nil, fmt.Errorf("repository: insert outbox: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("repository: commit: %w", err)
	}
	return r.GetOrder(ctx, orderID)
}

// MarkFulfilled transitions a paid order to fulfilled.
func (r *PostgresRepository) MarkFulfilled(ctx context.Context, orderID, fulfilledBy string, fulfilledAt time.Time, event OutboxEvent) (*Order, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("repository: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	var status string
	var metadataJSON []byte
	if err := tx.QueryRowContext(ctx, `
		SELECT status, metadata FROM orders WHERE order_id = $1 FOR UPDATE`, orderID,
	).Scan(&status, &metadataJSON); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrOrderNotFound
		}
		return nil, fmt.Errorf("repository: select order: %w", err)
	}

	if status == "fulfilled" {
		if err := tx.Commit(); err != nil {
			return nil, fmt.Errorf("repository: commit: %w", err)
		}
		return r.GetOrder(ctx, orderID)
	}

	if status != "paid" {
		return nil, fmt.Errorf("repository: cannot fulfill %s order: %w", status, ErrInvalidStatus)
	}

	metadata := map[string]string{}
	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &metadata); err != nil {
			return nil, fmt.Errorf("repository: unmarshal metadata: %w", err)
		}
	}
	if fulfilledBy != "" {
		metadata["fulfilled_by"] = fulfilledBy
	}
	metaJSON, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("repository: marshal metadata: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE orders SET status = $1, fulfilled_at = $2, metadata = $3, updated_at = $4
		WHERE order_id = $5`,
		"fulfilled", fulfilledAt, metaJSON, time.Now().UTC(), orderID,
	); err != nil {
		return nil, fmt.Errorf("repository: mark fulfilled: %w", err)
	}

	if err := outbox.InsertWithinTx(ctx, tx, outbox.Entry{
		EventID:       event.EventID,
		AggregateType: "order",
		AggregateID:   orderID,
		EventType:     event.EventType,
		Payload:       event.Payload,
		OccurredAtMs:  event.OccurredAtMs,
	}); err != nil {
		return nil, fmt.Errorf("repository: insert outbox: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("repository: commit: %w", err)
	}
	return r.GetOrder(ctx, orderID)
}

func (r *PostgresRepository) queryOrder(ctx context.Context, query string, args ...any) (*Order, error) {
	row, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("repository: query order: %w", err)
	}
	defer row.Close()
	if !row.Next() {
		if err := row.Err(); err != nil {
			return nil, fmt.Errorf("repository: query order: %w", err)
		}
		return nil, sql.ErrNoRows
	}
	return scanOrder(row)
}

func scanOrder(scanner interface {
	Scan(dest ...any) error
}) (*Order, error) {
	var order Order
	var itemsJSON, taxJSON, metaJSON []byte
	var paidAt, fulfilledAt, cancelledAt sql.NullTime

	if err := scanner.Scan(
		&order.OrderID, &order.StoreID, &order.KioskID, &order.CartID, &order.OrderNumber, &order.Status,
		&order.SubtotalCents, &order.TaxCents, &order.DiscountCents, &order.TotalCents, &order.Currency,
		&itemsJSON, &taxJSON, &metaJSON,
		&paidAt, &fulfilledAt, &cancelledAt,
		&order.CreatedAt, &order.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("repository: scan order: %w", err)
	}

	if paidAt.Valid {
		order.PaidAt = &paidAt.Time
	}
	if fulfilledAt.Valid {
		order.FulfilledAt = &fulfilledAt.Time
	}
	if cancelledAt.Valid {
		order.CancelledAt = &cancelledAt.Time
	}

	if len(itemsJSON) > 0 {
		if err := json.Unmarshal(itemsJSON, &order.Items); err != nil {
			return nil, fmt.Errorf("repository: unmarshal items: %w", err)
		}
	}
	if len(taxJSON) > 0 {
		if err := json.Unmarshal(taxJSON, &order.TaxBreakdown); err != nil {
			return nil, fmt.Errorf("repository: unmarshal tax breakdown: %w", err)
		}
	}
	if len(metaJSON) > 0 {
		if err := json.Unmarshal(metaJSON, &order.Metadata); err != nil {
			return nil, fmt.Errorf("repository: unmarshal metadata: %w", err)
		}
	}
	return &order, nil
}
