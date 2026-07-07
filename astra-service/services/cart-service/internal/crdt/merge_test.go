package crdt

import (
	"testing"
	"time"

	"github.com/astra-systems/astra-service/services/cart-service/internal/cart"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergeCarts_SumsIdenticalLines(t *testing.T) {
	now := time.Now()
	target := cart.NewCart("cart-1", "store-1", "kiosk-1", "lane-1", "session-1", "", now)
	ghost := cart.NewCart("cart-2", "store-1", "kiosk-1", "lane-1", "session-1", "", now)

	require.NoError(t, target.AddLine(cart.Line{
		LineID:                 "line-1",
		MenuItemID:             "item-a",
		NameSnapshot:           "Burger",
		UnitPriceCentsSnapshot: 500,
		Quantity:               1,
		AddedAtMs:              now.UnixMilli(),
	}, now))

	require.NoError(t, ghost.AddLine(cart.Line{
		LineID:                 "line-2",
		MenuItemID:             "item-a",
		NameSnapshot:           "Burger",
		UnitPriceCentsSnapshot: 500,
		Quantity:               2,
		AddedAtMs:              now.Add(time.Minute).UnixMilli(),
	}, now.Add(time.Minute)))

	res, err := MergeCarts(target, ghost, now.Add(time.Minute).UnixMilli(), now.Add(2*time.Minute))
	require.NoError(t, err)

	assert.Equal(t, 1, len(res.Cart.Lines))
	assert.Equal(t, 3, res.Cart.Lines[0].Quantity)
	assert.Equal(t, 1500, res.Cart.TotalCents)
	assert.Equal(t, target.Version+1, res.Cart.Version)
}

func TestMergeCarts_KeepsDistinctModifiersSeparate(t *testing.T) {
	now := time.Now()
	target := cart.NewCart("cart-1", "store-1", "kiosk-1", "lane-1", "session-1", "", now)
	ghost := cart.NewCart("cart-2", "store-1", "kiosk-1", "lane-1", "session-1", "", now)

	require.NoError(t, target.AddLine(cart.Line{
		LineID:     "line-1",
		MenuItemID: "item-a",
		Quantity:   1,
		Modifiers: []cart.Modifier{
			{ModifierOptionID: "opt-1", PriceDeltaCentsSnapshot: 100},
		},
		AddedAtMs: now.UnixMilli(),
	}, now))

	require.NoError(t, ghost.AddLine(cart.Line{
		LineID:     "line-2",
		MenuItemID: "item-a",
		Quantity:   1,
		Modifiers: []cart.Modifier{
			{ModifierOptionID: "opt-2", PriceDeltaCentsSnapshot: 200},
		},
		AddedAtMs: now.Add(time.Minute).UnixMilli(),
	}, now.Add(time.Minute)))

	res, err := MergeCarts(target, ghost, now.Add(time.Minute).UnixMilli(), now.Add(2*time.Minute))
	require.NoError(t, err)

	assert.Equal(t, 2, len(res.Cart.Lines))
	assert.Equal(t, 2, res.LinesAdded)
}

func TestMergeCarts_NilSourceReturnsTarget(t *testing.T) {
	now := time.Now()
	target := cart.NewCart("cart-1", "store-1", "kiosk-1", "lane-1", "session-1", "", now)
	res, err := MergeCarts(target, nil, 0, now)
	require.NoError(t, err)
	assert.Equal(t, target.CartID, res.Cart.CartID)
}

func TestMergeCarts_NilTargetReturnsError(t *testing.T) {
	now := time.Now()
	_, err := MergeCarts(nil, cart.NewCart("cart-2", "store-1", "kiosk-1", "lane-1", "session-1", "", now), 0, now)
	assert.ErrorIs(t, err, cart.ErrCartNotFound)
}
