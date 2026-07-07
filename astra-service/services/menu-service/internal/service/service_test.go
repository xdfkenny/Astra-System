package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	menupb "github.com/astra-systems/astra-service/proto/gen/go/menu"
	"github.com/astra-systems/astra-service/services/menu-service/internal/cache"
	"github.com/astra-systems/astra-service/services/menu-service/internal/model"
	"github.com/astra-systems/astra-service/services/menu-service/internal/service"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

type mockStore struct {
	categories []model.Category
	items      []model.Item
	item       *model.Item
	groups     map[uuid.UUID][]model.ModifierGroup
	err        error
}

func (m *mockStore) GetCategories(ctx context.Context, storeID uuid.UUID, includeInactive bool) ([]model.Category, error) {
	return m.categories, m.err
}

func (m *mockStore) GetItems(ctx context.Context, storeID uuid.UUID, includeInactive bool) ([]model.Item, error) {
	return m.items, m.err
}

func (m *mockStore) GetItemByID(ctx context.Context, itemID uuid.UUID) (*model.Item, error) {
	return m.item, m.err
}

func (m *mockStore) GetModifierGroupsByItemIDs(ctx context.Context, itemIDs []uuid.UUID) (map[uuid.UUID][]model.ModifierGroup, error) {
	return m.groups, m.err
}

func (m *mockStore) SearchItems(ctx context.Context, storeID uuid.UUID, query, categoryID string, includeInactive bool) ([]model.Item, error) {
	return m.items, m.err
}

func newCache(t *testing.T) (*cache.Cache, func()) {
	s := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: s.Addr()})
	return cache.New(client), func() { _ = client.Close(); s.Close() }
}

func TestMenuService_GetMenu(t *testing.T) {
	store := &mockStore{
		categories: []model.Category{{CategoryID: uuid.New(), StoreID: uuid.New(), Name: "Drinks"}},
		items:      []model.Item{{ItemID: uuid.New(), StoreID: uuid.New(), CategoryID: uuid.New(), Name: "Coffee", PriceCents: 250}},
		groups:     map[uuid.UUID][]model.ModifierGroup{},
	}
	c, cleanup := newCache(t)
	defer cleanup()

	svc := service.NewMenuService(store, c, time.Minute)
	resp, err := svc.GetMenu(context.Background(), &menupb.MenuRequest{StoreId: store.categories[0].StoreID.String()})
	require.NoError(t, err)
	require.Len(t, resp.Categories, 1)
	require.Len(t, resp.Items, 1)
	require.Equal(t, "Coffee", resp.Items[0].Name)
}

func TestMenuService_GetMenu_InvalidStoreID(t *testing.T) {
	c, cleanup := newCache(t)
	defer cleanup()
	svc := service.NewMenuService(&mockStore{}, c, time.Minute)
	_, err := svc.GetMenu(context.Background(), &menupb.MenuRequest{StoreId: "not-a-uuid"})
	require.Error(t, err)
}

func TestMenuService_GetCategories_Caches(t *testing.T) {
	store := &mockStore{
		categories: []model.Category{{CategoryID: uuid.New(), StoreID: uuid.New(), Name: "Sides"}},
	}
	c, cleanup := newCache(t)
	defer cleanup()

	svc := service.NewMenuService(store, c, time.Minute)
	storeID := store.categories[0].StoreID.String()

	resp, err := svc.GetCategories(context.Background(), &menupb.MenuRequest{StoreId: storeID})
	require.NoError(t, err)
	require.Len(t, resp.Categories, 1)

	store.categories = nil // ensure cache is used
	resp, err = svc.GetCategories(context.Background(), &menupb.MenuRequest{StoreId: storeID})
	require.NoError(t, err)
	require.Len(t, resp.Categories, 1)
}

func TestMenuService_GetItem_NotFound(t *testing.T) {
	c, cleanup := newCache(t)
	defer cleanup()
	svc := service.NewMenuService(&mockStore{}, c, time.Minute)
	_, err := svc.GetItem(context.Background(), &menupb.GetItemRequest{ItemId: uuid.New().String()})
	require.Error(t, err)
}

func TestMenuService_SearchItems(t *testing.T) {
	store := &mockStore{
		items:  []model.Item{{ItemID: uuid.New(), StoreID: uuid.New(), CategoryID: uuid.New(), Name: "Tea", PriceCents: 150}},
		groups: map[uuid.UUID][]model.ModifierGroup{},
	}
	c, cleanup := newCache(t)
	defer cleanup()

	svc := service.NewMenuService(store, c, time.Minute)
	resp, err := svc.SearchItems(context.Background(), &menupb.SearchItemsRequest{StoreId: store.items[0].StoreID.String(), Query: "tea"})
	require.NoError(t, err)
	require.Len(t, resp.Items, 1)
}

func TestMenuService_GetItem(t *testing.T) {
	itemID := uuid.New()
	store := &mockStore{
		item:   &model.Item{ItemID: itemID, StoreID: uuid.New(), CategoryID: uuid.New(), Name: "Cookie", PriceCents: 199},
		groups: map[uuid.UUID][]model.ModifierGroup{},
	}
	c, cleanup := newCache(t)
	defer cleanup()

	svc := service.NewMenuService(store, c, time.Minute)
	resp, err := svc.GetItem(context.Background(), &menupb.GetItemRequest{ItemId: itemID.String()})
	require.NoError(t, err)
	require.Equal(t, "Cookie", resp.Name)
}
