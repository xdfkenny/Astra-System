package repository

import (
	"context"
	"testing"

	"github.com/astra-systems/astra-service/services/admin-graphql/internal/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostgresRepositoryListCategories(t *testing.T) {
	db, cleanup := testutil.NewPostgresContainer(t)
	defer cleanup()

	repo := NewPostgresRepository(db)
	ctx := context.Background()
	storeID := uuid.New().String()
	_, err := db.ExecContext(ctx, `INSERT INTO stores (store_id, name) VALUES ($1, 'Store')`, storeID)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO categories (store_id, name, display_order, is_active) VALUES ($1, 'Beverages', 1, TRUE)`, storeID)
	require.NoError(t, err)

	cats, err := repo.ListCategories(ctx, storeID, false)
	require.NoError(t, err)
	assert.Len(t, cats, 1)
	assert.Equal(t, "Beverages", cats[0].Name)
}

func TestPostgresRepositoryListItems(t *testing.T) {
	db, cleanup := testutil.NewPostgresContainer(t)
	defer cleanup()

	repo := NewPostgresRepository(db)
	ctx := context.Background()
	storeID := uuid.New().String()
	_, err := db.ExecContext(ctx, `INSERT INTO stores (store_id, name) VALUES ($1, 'Store')`, storeID)
	require.NoError(t, err)
	var categoryID string
	err = db.QueryRowContext(ctx, `INSERT INTO categories (store_id, name, display_order, is_active) VALUES ($1, 'Food', 1, TRUE) RETURNING category_id`, storeID).Scan(&categoryID)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO items (store_id, category_id, name, price_cents, is_active) VALUES ($1, $2, 'Burger', 999, TRUE)`, storeID, categoryID)
	require.NoError(t, err)

	items, err := repo.ListItems(ctx, storeID, false)
	require.NoError(t, err)
	assert.Len(t, items, 1)
	assert.Equal(t, "Burger", items[0].Name)
}

func TestPostgresRepositoryListOrders(t *testing.T) {
	db, cleanup := testutil.NewPostgresContainer(t)
	defer cleanup()

	repo := NewPostgresRepository(db)
	ctx := context.Background()
	storeID := uuid.New().String()
	kioskID := uuid.New().String()
	cartID := uuid.New().String()
	_, err := db.ExecContext(ctx, `INSERT INTO stores (store_id, name) VALUES ($1, 'Store')`, storeID)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO kiosks (kiosk_id, store_id, hardware_id, display_name, signing_key_hash) VALUES ($1, $2, 'hw-1', 'Kiosk', 'hash')`, kioskID, storeID)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO carts (cart_id, store_id, kiosk_id, session_id, items_json, total_cents) VALUES ($1, $2, $3, $4, '[]', 100)`, cartID, storeID, kioskID, cartID)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO orders (order_id, store_id, kiosk_id, cart_id, order_number, status, total_cents, items_json) VALUES ($1, $2, $3, $4, 'ORD-001', 'pending', 100, '[]')`, uuid.New().String(), storeID, kioskID, cartID)
	require.NoError(t, err)

	orders, total, err := repo.ListOrders(ctx, OrderFilter{StoreID: storeID})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, orders, 1)
}

func TestPostgresRepositoryListInventory(t *testing.T) {
	db, cleanup := testutil.NewPostgresContainer(t)
	defer cleanup()

	repo := NewPostgresRepository(db)
	ctx := context.Background()
	storeID := uuid.New().String()
	_, err := db.ExecContext(ctx, `INSERT INTO stores (store_id, name) VALUES ($1, 'Store')`, storeID)
	require.NoError(t, err)
	var itemID string
	err = db.QueryRowContext(ctx, `INSERT INTO categories (store_id, name, display_order, is_active) VALUES ($1, 'Cat', 1, TRUE) RETURNING category_id`, storeID).Scan(&itemID)
	require.NoError(t, err)
	err = db.QueryRowContext(ctx, `INSERT INTO items (store_id, category_id, name, price_cents, is_active) VALUES ($1, $2, 'Item', 100, TRUE) RETURNING item_id`, storeID, itemID).Scan(&itemID)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO inventory (store_id, item_id, quantity_available) VALUES ($1, $2, 42)`, storeID, itemID)
	require.NoError(t, err)

	inv, err := repo.ListInventory(ctx, storeID)
	require.NoError(t, err)
	assert.Len(t, inv, 1)
	assert.Equal(t, int32(42), inv[0].QuantityAvailable)
}

func TestPostgresRepositoryListUsers(t *testing.T) {
	db, cleanup := testutil.NewPostgresContainer(t)
	defer cleanup()

	repo := NewPostgresRepository(db)
	ctx := context.Background()
	tenantID := uuid.New().String()
	_, err := db.ExecContext(ctx, `INSERT INTO tenants (tenant_id, slug, name, billing_email) VALUES ($1, 'test', 'Test', 'test@example.com')`, tenantID)
	require.NoError(t, err)
	var roleID string
	err = db.QueryRowContext(ctx, `INSERT INTO roles (tenant_id, name) VALUES ($1, 'admin') RETURNING role_id`, tenantID).Scan(&roleID)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO users (tenant_id, email, name, role_id) VALUES ($1, 'admin@example.com', 'Admin', $2)`, tenantID, roleID)
	require.NoError(t, err)

	users, err := repo.ListUsers(ctx, tenantID)
	require.NoError(t, err)
	assert.Len(t, users, 1)
	assert.Equal(t, "admin", users[0].RoleName)
}

func TestPostgresRepositoryListRoles(t *testing.T) {
	db, cleanup := testutil.NewPostgresContainer(t)
	defer cleanup()

	repo := NewPostgresRepository(db)
	ctx := context.Background()
	tenantID := uuid.New().String()
	_, err := db.ExecContext(ctx, `INSERT INTO tenants (tenant_id, slug, name, billing_email) VALUES ($1, 'test', 'Test', 'test@example.com')`, tenantID)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO roles (tenant_id, name) VALUES ($1, 'manager')`, tenantID)
	require.NoError(t, err)

	roles, err := repo.ListRoles(ctx, tenantID)
	require.NoError(t, err)
	assert.Len(t, roles, 1)
	assert.Equal(t, "manager", roles[0].Name)
}
