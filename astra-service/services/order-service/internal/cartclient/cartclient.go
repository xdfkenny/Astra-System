// Package cartclient provides a thin client for fetching finalized cart
// details from the CartService. The interface keeps the order-service decoupled
// from transport details and makes unit tests trivial to fake.
package cartclient

import (
	"context"
	"fmt"

	cartv1 "github.com/astra-systems/astra-service/proto/gen/go/cart"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Client is the minimal surface needed by the order-service to convert a cart
// into an order.
type Client interface {
	GetCart(ctx context.Context, cartID string) (*cartv1.Cart, error)
}

// GRPCClient calls the CartService over gRPC.
type GRPCClient struct {
	client cartv1.CartServiceClient
	conn   *grpc.ClientConn
}

// NewGRPCClient dials the cart-service at the supplied target.
func NewGRPCClient(target string) (*GRPCClient, error) {
	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("cartclient: dial: %w", err)
	}
	return &GRPCClient{
		client: cartv1.NewCartServiceClient(conn),
		conn:   conn,
	}, nil
}

// GetCart fetches a cart by ID.
func (c *GRPCClient) GetCart(ctx context.Context, cartID string) (*cartv1.Cart, error) {
	return c.client.GetCart(ctx, &cartv1.GetCartRequest{CartId: cartID})
}

// Close tears down the underlying gRPC connection.
func (c *GRPCClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// FakeClient is an in-memory implementation used by tests.
type FakeClient struct {
	carts map[string]*cartv1.Cart
}

// NewFakeClient returns a fake client backed by the supplied carts.
func NewFakeClient(carts map[string]*cartv1.Cart) *FakeClient {
	if carts == nil {
		carts = make(map[string]*cartv1.Cart)
	}
	return &FakeClient{carts: carts}
}

// GetCart returns the cart if it has been registered.
func (f *FakeClient) GetCart(ctx context.Context, cartID string) (*cartv1.Cart, error) {
	cart, ok := f.carts[cartID]
	if !ok {
		return nil, fmt.Errorf("cartclient: cart %s not found", cartID)
	}
	return cart, nil
}

// Register adds a cart to the fake store.
func (f *FakeClient) Register(cart *cartv1.Cart) {
	f.carts[cart.CartId] = cart
}
