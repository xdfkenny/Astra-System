package cache_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/astra-systems/astra-service/services/menu-service/internal/cache"
	"github.com/astra-systems/astra-service/services/menu-service/internal/model"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func newTestCache(t *testing.T) (*cache.Cache, func()) {
	s := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: s.Addr()})
	return cache.New(client), func() { _ = client.Close(); s.Close() }
}

func TestCache_MenuRoundTrip(t *testing.T) {
	c, cleanup := newTestCache(t)
	defer cleanup()
	ctx := context.Background()

	storeID := uuid.New().String()
	menu := model.Menu{StoreID: uuid.MustParse(storeID), Categories: []model.Category{{Name: "Mains"}}}

	hit, err := c.GetMenu(ctx, storeID, &model.Menu{})
	require.NoError(t, err)
	require.False(t, hit)

	require.NoError(t, c.SetMenu(ctx, storeID, menu, time.Minute))

	var fetched model.Menu
	hit, err = c.GetMenu(ctx, storeID, &fetched)
	require.NoError(t, err)
	require.True(t, hit)
	require.Len(t, fetched.Categories, 1)
	require.Equal(t, "Mains", fetched.Categories[0].Name)
}

func TestCache_InvalidateMenu(t *testing.T) {
	c, cleanup := newTestCache(t)
	defer cleanup()
	ctx := context.Background()

	storeID := uuid.New().String()
	require.NoError(t, c.SetMenu(ctx, storeID, model.Menu{StoreID: uuid.MustParse(storeID)}, time.Minute))
	require.NoError(t, c.SetCategories(ctx, storeID, []model.Category{{Name: "All"}}, time.Minute))

	require.NoError(t, c.InvalidateMenu(ctx, storeID))

	var menu model.Menu
	hit, err := c.GetMenu(ctx, storeID, &menu)
	require.NoError(t, err)
	require.False(t, hit)

	var cats []model.Category
	hit, err = c.GetCategories(ctx, storeID, &cats)
	require.NoError(t, err)
	require.False(t, hit)
}
