// Package inventory provides a gRPC client to the inventory-service for
// reserving and releasing stock against a cart.
package inventory

import (
	"context"
	"fmt"
	"time"

	inventoryv1 "github.com/astra-systems/astra-service/proto/gen/go/inventory"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Client wraps the generated InventoryService gRPC client.
type Client struct {
	conn inventoryv1.InventoryServiceClient
	cc   *grpc.ClientConn
}

// NewClient dials the inventory-service at the supplied address.
func NewClient(ctx context.Context, addr string) (*Client, error) {
	cc, err := grpc.DialContext(ctx, addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("inventory_client: dial %s: %w", addr, err)
	}
	return &Client{
		conn: inventoryv1.NewInventoryServiceClient(cc),
		cc:   cc,
	}, nil
}

// Reserve calls InventoryService.ReserveStock and returns the resulting stock
// level. The reservation expires at expiresAtMs.
func (c *Client) Reserve(ctx context.Context, storeID, kioskID, itemID, cartID string, quantity int, expiresAtMs int64) (*inventoryv1.StockLevel, error) {
	resp, err := c.conn.ReserveStock(ctx, &inventoryv1.ReservationRequest{
		StoreId:     storeID,
		KioskId:     kioskID,
		ItemId:      itemID,
		CartId:      cartID,
		Quantity:    int32(quantity),
		ExpiresAtMs: expiresAtMs,
	})
	if err != nil {
		return nil, fmt.Errorf("inventory_client: reserve stock: %w", err)
	}
	return resp, nil
}

// Release calls InventoryService.ReleaseStock.
func (c *Client) Release(ctx context.Context, storeID, itemID, cartID string, quantity int, reason string) (*inventoryv1.StockLevel, error) {
	resp, err := c.conn.ReleaseStock(ctx, &inventoryv1.ReleaseStockRequest{
		StoreId:  storeID,
		ItemId:   itemID,
		CartId:   cartID,
		Quantity: int32(quantity),
		Reason:   reason,
	})
	if err != nil {
		return nil, fmt.Errorf("inventory_client: release stock: %w", err)
	}
	return resp, nil
}

// Close closes the underlying gRPC connection.
func (c *Client) Close() error {
	return c.cc.Close()
}

// DefaultReservationTTL is the time-to-live for a soft inventory reservation
// held against an active cart.
const DefaultReservationTTL = 5 * time.Minute

// DefaultExpiresAtMs returns the wall-clock milliseconds at which a new
// reservation should expire.
func DefaultExpiresAtMs() int64 {
	return time.Now().Add(DefaultReservationTTL).UnixMilli()
}

// ReservationClient is the subset of Client used by the cart service. It
// allows tests to inject a mock inventory service.
type ReservationClient interface {
	Reserve(ctx context.Context, storeID, kioskID, itemID, cartID string, quantity int, expiresAtMs int64) (*inventoryv1.StockLevel, error)
	Release(ctx context.Context, storeID, itemID, cartID string, quantity int, reason string) (*inventoryv1.StockLevel, error)
}

// Ensure Client implements ReservationClient.
var _ ReservationClient = (*Client)(nil)
