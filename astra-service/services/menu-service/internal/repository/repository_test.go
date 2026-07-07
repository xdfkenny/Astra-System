package repository_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/astra-systems/astra-service/services/menu-service/internal/model"
	"github.com/astra-systems/astra-service/services/menu-service/internal/repository"
	"github.com/astra-systems/astra-service/services/menu-service/internal/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func setupRepo(t *testing.T) (*repository.Repository, *sql.DB, func()) {
	t.Helper()
	db, cleanup := testutil.NewPostgresContainer(t)
	repo, err := repository.NewRepository(db)
	require.NoError(t, err)
	return repo, db, func() {
		_ = repo.Close()
		cleanup()
	}
}

func seedStore(t *testing.T, db *sql.DB) uuid.UUID {
	t.Helper()
	storeID := uuid.New()
	_, err := db.Exec(`INSERT INTO stores (store_id, name, timezone, currency, tax_rate) VALUES ($1, $2, $3, $4, $5)`,
		storeID, "Test Store", "UTC", "USD", 0.0)
	require.NoError(t, err)
	return storeID
}

func TestRepository_GetCategories(t *testing.T) {
	repo, db, cleanup := setupRepo(t)
	defer cleanup()
	ctx := context.Background()
	storeID := seedStore(t, db)

	cat := &model.Category{StoreID: storeID, Name: "Beverages", DisplayOrder: 1, IsActive: true}
	require.NoError(t, repo.CreateCategory(ctx, cat))

	categories, err := repo.GetCategories(ctx, storeID, false)
	require.NoError(t, err)
	require.Len(t, categories, 1)
	require.Equal(t, "Beverages", categories[0].Name)
}

func TestRepository_GetItems(t *testing.T) {
	repo, db, cleanup := setupRepo(t)
	defer cleanup()
	ctx := context.Background()
	storeID := seedStore(t, db)

	cat := &model.Category{StoreID: storeID, Name: "Food", DisplayOrder: 1, IsActive: true}
	require.NoError(t, repo.CreateCategory(ctx, cat))

	item := &model.Item{StoreID: storeID, CategoryID: cat.CategoryID, Name: "Burger", PriceCents: 999, IsActive: true}
	require.NoError(t, repo.CreateItem(ctx, item))

	items, err := repo.GetItems(ctx, storeID, false)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "Burger", items[0].Name)
}

func TestRepository_GetItemByID(t *testing.T) {
	repo, db, cleanup := setupRepo(t)
	defer cleanup()
	ctx := context.Background()
	storeID := seedStore(t, db)

	cat := &model.Category{StoreID: storeID, Name: "Food", DisplayOrder: 1, IsActive: true}
	require.NoError(t, repo.CreateCategory(ctx, cat))

	item := &model.Item{StoreID: storeID, CategoryID: cat.CategoryID, Name: "Fries", PriceCents: 399, IsActive: true}
	require.NoError(t, repo.CreateItem(ctx, item))

	fetched, err := repo.GetItemByID(ctx, item.ItemID)
	require.NoError(t, err)
	require.NotNil(t, fetched)
	require.Equal(t, "Fries", fetched.Name)

	missing, err := repo.GetItemByID(ctx, uuid.New())
	require.NoError(t, err)
	require.Nil(t, missing)
}

func TestRepository_SearchItems(t *testing.T) {
	repo, db, cleanup := setupRepo(t)
	defer cleanup()
	ctx := context.Background()
	storeID := seedStore(t, db)

	cat := &model.Category{StoreID: storeID, Name: "Drinks", DisplayOrder: 1, IsActive: true}
	require.NoError(t, repo.CreateCategory(ctx, cat))

	require.NoError(t, repo.CreateItem(ctx, &model.Item{StoreID: storeID, CategoryID: cat.CategoryID, Name: "Cola", PriceCents: 199, IsActive: true}))
	require.NoError(t, repo.CreateItem(ctx, &model.Item{StoreID: storeID, CategoryID: cat.CategoryID, Name: "Water", PriceCents: 99, IsActive: true}))

	results, err := repo.SearchItems(ctx, storeID, "col", "", false)
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, "Cola", results[0].Name)
}

func TestRepository_UpdateItemPrice(t *testing.T) {
	repo, db, cleanup := setupRepo(t)
	defer cleanup()
	ctx := context.Background()
	storeID := seedStore(t, db)

	cat := &model.Category{StoreID: storeID, Name: "Food", DisplayOrder: 1, IsActive: true}
	require.NoError(t, repo.CreateCategory(ctx, cat))

	item := &model.Item{StoreID: storeID, CategoryID: cat.CategoryID, Name: "Shake", PriceCents: 499, IsActive: true}
	require.NoError(t, repo.CreateItem(ctx, item))

	require.NoError(t, repo.UpdateItemPrice(ctx, item.ItemID, 599))

	updated, err := repo.GetItemByID(ctx, item.ItemID)
	require.NoError(t, err)
	require.Equal(t, 599, updated.PriceCents)

	var published bool
	err = db.QueryRowContext(ctx, "SELECT published FROM outbox_events WHERE aggregate_id = $1", item.ItemID).Scan(&published)
	require.NoError(t, err)
	require.False(t, published)
}

func TestRepository_CategoryLifecycle(t *testing.T) {
	repo, db, cleanup := setupRepo(t)
	defer cleanup()
	ctx := context.Background()
	storeID := seedStore(t, db)

	cat := &model.Category{StoreID: storeID, Name: "Desserts", DisplayOrder: 1, IsActive: true}
	require.NoError(t, repo.CreateCategory(ctx, cat))

	cat.Name = "Sweets"
	require.NoError(t, repo.UpdateCategory(ctx, cat))

	categories, err := repo.GetCategories(ctx, storeID, false)
	require.NoError(t, err)
	require.Len(t, categories, 1)
	require.Equal(t, "Sweets", categories[0].Name)

	require.NoError(t, repo.DeleteCategory(ctx, cat.CategoryID, storeID))
	categories, err = repo.GetCategories(ctx, storeID, false)
	require.NoError(t, err)
	require.Empty(t, categories)
}
