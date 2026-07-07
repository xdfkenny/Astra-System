// Package clients holds gRPC client connections to downstream Astra services.
package clients

import (
	"fmt"

	cartpb "github.com/astra-systems/astra-service/proto/gen/go/cart"
	inventorypb "github.com/astra-systems/astra-service/proto/gen/go/inventory"
	menupb "github.com/astra-systems/astra-service/proto/gen/go/menu"
	orderpb "github.com/astra-systems/astra-service/proto/gen/go/order"
	paymentpb "github.com/astra-systems/astra-service/proto/gen/go/payment"
	syncpb "github.com/astra-systems/astra-service/proto/gen/go/sync"
	"github.com/astra-systems/astra-service/services/gateway/internal/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Registry holds generated gRPC clients for all downstream services.
type Registry struct {
	Menu      menupb.MenuServiceClient
	Cart      cartpb.CartServiceClient
	Order     orderpb.OrderServiceClient
	Inventory inventorypb.InventoryServiceClient
	Payment   paymentpb.PaymentOrchestratorClient
	Sync      syncpb.SyncServiceClient
	conns     []*grpc.ClientConn
}

// NewRegistry dials every configured downstream gRPC service.
func NewRegistry(cfg *config.Config) (*Registry, error) {
	menuConn, err := dial("menu", cfg.Services["menu"].GRPCAddr)
	if err != nil {
		return nil, err
	}
	cartConn, err := dial("cart", cfg.Services["cart"].GRPCAddr)
	if err != nil {
		_ = menuConn.Close()
		return nil, err
	}
	orderConn, err := dial("order", cfg.Services["order"].GRPCAddr)
	if err != nil {
		_ = menuConn.Close()
		_ = cartConn.Close()
		return nil, err
	}
	inventoryConn, err := dial("inventory", cfg.Services["inventory"].GRPCAddr)
	if err != nil {
		_ = menuConn.Close()
		_ = cartConn.Close()
		_ = orderConn.Close()
		return nil, err
	}
	paymentConn, err := dial("payment", cfg.Services["payment"].GRPCAddr)
	if err != nil {
		_ = menuConn.Close()
		_ = cartConn.Close()
		_ = orderConn.Close()
		_ = inventoryConn.Close()
		return nil, err
	}
	syncConn, err := dial("sync", cfg.Services["sync"].GRPCAddr)
	if err != nil {
		_ = menuConn.Close()
		_ = cartConn.Close()
		_ = orderConn.Close()
		_ = inventoryConn.Close()
		_ = paymentConn.Close()
		return nil, err
	}

	return &Registry{
		Menu:      menupb.NewMenuServiceClient(menuConn),
		Cart:      cartpb.NewCartServiceClient(cartConn),
		Order:     orderpb.NewOrderServiceClient(orderConn),
		Inventory: inventorypb.NewInventoryServiceClient(inventoryConn),
		Payment:   paymentpb.NewPaymentOrchestratorClient(paymentConn),
		Sync:      syncpb.NewSyncServiceClient(syncConn),
		conns:     []*grpc.ClientConn{menuConn, cartConn, orderConn, inventoryConn, paymentConn, syncConn},
	}, nil
}

func dial(name, addr string) (*grpc.ClientConn, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("dial %s service at %s: %w", name, addr, err)
	}
	return conn, nil
}

// Close closes all downstream gRPC connections.
func (r *Registry) Close() error {
	var firstErr error
	for _, conn := range r.conns {
		if err := conn.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
