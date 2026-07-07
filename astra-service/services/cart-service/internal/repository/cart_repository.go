// Package repository implements Postgres persistence for the Cart aggregate
// using normalized rows (carts + cart_lines) and optimistic locking via the
// version column. Every write is combined with an outbox event in the same
// transaction.
package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/astra-service/go-common/outbox"
	"github.com/astra-systems/astra-service/services/cart-service/internal/cart"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

// CartRepository provides Postgres access for carts.
type CartRepository struct {
	db *sql.DB
}

// NewCartRepository creates a new repository backed by db.
func NewCartRepository(db *sql.DB) *CartRepository {
	return &CartRepository{db: db}
}

// CreateCart inserts a new cart and its lines. It returns an error if a cart
// with the same ID already exists.
func (r *CartRepository) CreateCart(ctx context.Context, c *cart.Cart) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("cart_repository: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := r.upsertCart(ctx, tx, c, true); err != nil {
		return err
	}
	if err := r.replaceLines(ctx, tx, c); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("cart_repository: commit create: %w", err)
	}
	return nil
}

// GetCart loads a cart and its lines by cart ID.
func (r *CartRepository) GetCart(ctx context.Context, cartID string) (*cart.Cart, error) {
	c := &cart.Cart{}
	var customerPhone sql.NullString
	var status string
	var expiresAt, createdAt, updatedAt time.Time

	err := r.db.QueryRowContext(ctx, `
		SELECT cart_id, store_id, kiosk_id, session_id, customer_phone,
		       status, finalized, version, total_cents, tax_cents,
		       discount_cents, final_total_cents, reserved_inventory,
		       expires_at, created_at, updated_at, created_at_ms, updated_at_ms
		FROM carts
		WHERE cart_id = $1`, cartID).Scan(
		&c.CartID, &c.StoreID, &c.KioskID, &c.SessionID, &customerPhone,
		&status, &c.Finalized, &c.Version, &c.TotalCents, &c.TaxCents,
		&c.DiscountCents, &c.FinalTotalCents, &c.ReservedInventory,
		&expiresAt, &createdAt, &updatedAt, &c.CreatedAtMs, &c.UpdatedAtMs,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, cart.ErrCartNotFound
		}
		return nil, fmt.Errorf("cart_repository: select cart: %w", err)
	}

	c.CustomerPhone = customerPhone.String
	c.Status = cart.CartStatus(status)
	c.ExpiresAt = expiresAt
	c.CreatedAt = createdAt
	c.UpdatedAt = updatedAt

	lines, err := r.loadLines(ctx, cartID)
	if err != nil {
		return nil, err
	}
	c.Lines = lines

	return c, nil
}

// SaveCart persists the cart and its lines using optimistic locking. When
// event is non-nil, the outbox entry is written in the same transaction. If
// the cart version has changed since it was read, ErrVersionConflict is
// returned.
func (r *CartRepository) SaveCart(ctx context.Context, c *cart.Cart, event *outbox.Entry) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("cart_repository: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Optimistic update: only succeed if the persisted version matches the
	// version the caller read. If not, another writer modified the cart first.
	expectedVersion := c.Version - 1
	res, err := tx.ExecContext(ctx, `
		UPDATE carts
		SET store_id = $2,
		    kiosk_id = $3,
		    session_id = $4,
		    customer_phone = $5,
		    status = $6,
		    finalized = $7,
		    version = $8,
		    total_cents = $9,
		    tax_cents = $10,
		    discount_cents = $11,
		    final_total_cents = $12,
		    reserved_inventory = $13,
		    expires_at = $14,
		    updated_at = $15,
		    updated_at_ms = $16
		WHERE cart_id = $1 AND version = $17`,
		c.CartID, c.StoreID, c.KioskID, c.SessionID, sqlNullString(c.CustomerPhone),
		string(c.Status), c.Finalized, c.Version, c.TotalCents, c.TaxCents,
		c.DiscountCents, c.FinalTotalCents, c.ReservedInventory,
		c.ExpiresAt, c.UpdatedAt, c.UpdatedAtMs, expectedVersion,
	)
	if err != nil {
		return fmt.Errorf("cart_repository: update cart: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("cart_repository: rows affected: %w", err)
	}
	if rows == 0 {
		return cart.ErrVersionConflict
	}

	if err := r.replaceLines(ctx, tx, c); err != nil {
		return err
	}
	if event != nil {
		if err := outbox.InsertWithinTx(ctx, tx, *event); err != nil {
			return fmt.Errorf("cart_repository: insert outbox: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("cart_repository: commit save: %w", err)
	}
	return nil
}

// DeleteCart removes a cart and its lines.
func (r *CartRepository) DeleteCart(ctx context.Context, cartID string) error {
	if _, err := r.db.ExecContext(ctx, `DELETE FROM carts WHERE cart_id = $1`, cartID); err != nil {
		return fmt.Errorf("cart_repository: delete cart: %w", err)
	}
	return nil
}

func (r *CartRepository) upsertCart(ctx context.Context, tx *sql.Tx, c *cart.Cart, insertOnly bool) error {
	if insertOnly {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO carts
			  (cart_id, store_id, kiosk_id, session_id, customer_phone,
			   status, finalized, version, total_cents, tax_cents,
			   discount_cents, final_total_cents, reserved_inventory,
			   expires_at, created_at, updated_at, created_at_ms, updated_at_ms)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)`,
			c.CartID, c.StoreID, c.KioskID, c.SessionID, sqlNullString(c.CustomerPhone),
			string(c.Status), c.Finalized, c.Version, c.TotalCents, c.TaxCents,
			c.DiscountCents, c.FinalTotalCents, c.ReservedInventory,
			c.ExpiresAt, c.CreatedAt, c.UpdatedAt, c.CreatedAtMs, c.UpdatedAtMs,
		)
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				return cart.ErrVersionConflict
			}
			return fmt.Errorf("cart_repository: insert cart: %w", err)
		}
		return nil
	}

	_, err := tx.ExecContext(ctx, `
		INSERT INTO carts
		  (cart_id, store_id, kiosk_id, session_id, customer_phone,
		   status, finalized, version, total_cents, tax_cents,
		   discount_cents, final_total_cents, reserved_inventory,
		   expires_at, created_at, updated_at, created_at_ms, updated_at_ms)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
		ON CONFLICT (cart_id) DO UPDATE SET
		  store_id = EXCLUDED.store_id,
		  version = EXCLUDED.version,
		  finalized = EXCLUDED.finalized,
		  status = EXCLUDED.status,
		  total_cents = EXCLUDED.total_cents,
		  tax_cents = EXCLUDED.tax_cents,
		  discount_cents = EXCLUDED.discount_cents,
		  final_total_cents = EXCLUDED.final_total_cents,
		  reserved_inventory = EXCLUDED.reserved_inventory,
		  expires_at = EXCLUDED.expires_at,
		  updated_at = EXCLUDED.updated_at,
		  updated_at_ms = EXCLUDED.updated_at_ms`,
		c.CartID, c.StoreID, c.KioskID, c.SessionID, sqlNullString(c.CustomerPhone),
		string(c.Status), c.Finalized, c.Version, c.TotalCents, c.TaxCents,
		c.DiscountCents, c.FinalTotalCents, c.ReservedInventory,
		c.ExpiresAt, c.CreatedAt, c.UpdatedAt, c.CreatedAtMs, c.UpdatedAtMs,
	)
	if err != nil {
		return fmt.Errorf("cart_repository: upsert cart: %w", err)
	}
	return nil
}

func (r *CartRepository) replaceLines(ctx context.Context, tx *sql.Tx, c *cart.Cart) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM cart_lines WHERE cart_id = $1`, c.CartID); err != nil {
		return fmt.Errorf("cart_repository: delete lines: %w", err)
	}
	for _, line := range c.Lines {
		modifiersJSON, err := json.Marshal(line.Modifiers)
		if err != nil {
			return fmt.Errorf("cart_repository: marshal modifiers: %w", err)
		}
		lineID := line.LineID
		if lineID == "" {
			lineID = uuid.New().String()
		}
		_, err = tx.ExecContext(ctx, `
			INSERT INTO cart_lines
			  (line_id, cart_id, menu_item_id, name_snapshot, unit_price_cents_snapshot,
			   quantity, modifiers, added_at_ms)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			lineID, c.CartID, line.MenuItemID, line.NameSnapshot,
			line.UnitPriceCentsSnapshot, line.Quantity, modifiersJSON, line.AddedAtMs,
		)
		if err != nil {
			return fmt.Errorf("cart_repository: insert line: %w", err)
		}
	}
	return nil
}

func (r *CartRepository) loadLines(ctx context.Context, cartID string) ([]cart.Line, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT line_id, cart_id, menu_item_id, name_snapshot,
		       unit_price_cents_snapshot, quantity, modifiers, added_at_ms
		FROM cart_lines
		WHERE cart_id = $1
		ORDER BY added_at_ms ASC`, cartID)
	if err != nil {
		return nil, fmt.Errorf("cart_repository: select lines: %w", err)
	}
	defer rows.Close()

	var lines []cart.Line
	for rows.Next() {
		var line cart.Line
		var modifiersJSON []byte
		if err := rows.Scan(
			&line.LineID, &line.CartID, &line.MenuItemID, &line.NameSnapshot,
			&line.UnitPriceCentsSnapshot, &line.Quantity, &modifiersJSON, &line.AddedAtMs,
		); err != nil {
			return nil, fmt.Errorf("cart_repository: scan line: %w", err)
		}
		if len(modifiersJSON) > 0 {
			if err := json.Unmarshal(modifiersJSON, &line.Modifiers); err != nil {
				return nil, fmt.Errorf("cart_repository: unmarshal modifiers: %w", err)
			}
		}
		lines = append(lines, line)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("cart_repository: iterate lines: %w", err)
	}
	return lines, nil
}

func sqlNullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}
