package repository

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/astra-service/go-common/outbox"
	"github.com/astra-systems/astra-service/services/cart-service/internal/cart"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepository_CreateCart(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewCartRepository(db)
	c := cart.NewCart("cart-1", "store-1", "kiosk-1", "lane-1", "session-1", "+15551234567", time.Now())

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO carts").
		WithArgs(
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(),
		).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("DELETE FROM cart_lines").
		WithArgs(c.CartID).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()

	err = repo.CreateCart(context.Background(), c)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetCart(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewCartRepository(db)
	now := time.Now()
	cartID := "cart-1"

	mock.ExpectQuery("SELECT .* FROM carts WHERE cart_id = \\$1").
		WithArgs(cartID).
		WillReturnRows(sqlmock.NewRows([]string{
			"cart_id", "store_id", "kiosk_id", "session_id", "customer_phone",
			"status", "finalized", "version", "total_cents", "tax_cents",
			"discount_cents", "final_total_cents", "reserved_inventory",
			"expires_at", "created_at", "updated_at", "created_at_ms", "updated_at_ms",
		}).AddRow(
			cartID, "store-1", "kiosk-1", "session-1", nil,
			"active", false, 1, 1000, 0,
			0, 1000, false,
			now, now, now, now.UnixMilli(), now.UnixMilli(),
		))

	modifiers, _ := json.Marshal([]cart.Modifier{})
	mock.ExpectQuery("SELECT .* FROM cart_lines WHERE cart_id = \\$1").
		WithArgs(cartID).
		WillReturnRows(sqlmock.NewRows([]string{
			"line_id", "cart_id", "menu_item_id", "name_snapshot",
			"unit_price_cents_snapshot", "quantity", "modifiers", "added_at_ms",
		}).AddRow(
			"line-1", cartID, "item-1", "Burger",
			500, 2, modifiers, now.UnixMilli(),
		))

	c, err := repo.GetCart(context.Background(), cartID)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
	assert.Equal(t, cartID, c.CartID)
	assert.Equal(t, 1, len(c.Lines))
	assert.Equal(t, 1000, c.TotalCents)
}

func TestRepository_SaveCart_OptimisticLockConflict(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewCartRepository(db)
	c := cart.NewCart("cart-1", "store-1", "kiosk-1", "lane-1", "session-1", "", time.Now())
	c.Version = 2 // persisted version is expected to be 1

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE carts").
		WithArgs(
			c.CartID, c.StoreID, c.KioskID, c.SessionID, sqlmock.AnyArg(),
			string(c.Status), c.Finalized, c.Version, c.TotalCents, c.TaxCents,
			c.DiscountCents, c.FinalTotalCents, c.ReservedInventory,
			c.ExpiresAt, c.UpdatedAt, c.UpdatedAtMs, c.Version-1,
		).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectRollback()

	event := outbox.Entry{EventID: "evt-1", AggregateType: "cart", AggregateID: c.CartID, EventType: "ItemAddedToCart", Payload: []byte("{}"), OccurredAtMs: time.Now().UnixMilli()}
	err = repo.SaveCart(context.Background(), c, &event)
	assert.ErrorIs(t, err, cart.ErrVersionConflict)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_SaveCart_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewCartRepository(db)
	c := cart.NewCart("cart-1", "store-1", "kiosk-1", "lane-1", "session-1", "", time.Now())
	require.NoError(t, c.AddLine(cart.Line{LineID: "line-1", MenuItemID: "item-1", NameSnapshot: "Burger", UnitPriceCentsSnapshot: 500, Quantity: 2, AddedAtMs: time.Now().UnixMilli()}, time.Now()))
	c.Version = 1 // expected persisted version 0

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE carts").
		WithArgs(
			c.CartID, c.StoreID, c.KioskID, c.SessionID, sqlmock.AnyArg(),
			string(c.Status), c.Finalized, c.Version, c.TotalCents, c.TaxCents,
			c.DiscountCents, c.FinalTotalCents, c.ReservedInventory,
			c.ExpiresAt, c.UpdatedAt, c.UpdatedAtMs, c.Version-1,
		).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("DELETE FROM cart_lines").
		WithArgs(c.CartID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO cart_lines").
		WithArgs(sqlmock.AnyArg(), c.CartID, "item-1", "Burger", 500, 2, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO outbox_events").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	event := outbox.Entry{EventID: "evt-1", AggregateType: "cart", AggregateID: c.CartID, EventType: "ItemAddedToCart", Payload: []byte("{}"), OccurredAtMs: time.Now().UnixMilli()}
	err = repo.SaveCart(context.Background(), c, &event)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}
