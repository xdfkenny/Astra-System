package inventory

import (
	"context"
	"net"
	"testing"

	inventoryv1 "github.com/astra-systems/astra-service/proto/gen/go/inventory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

type fakeInventoryServer struct {
	inventoryv1.UnimplementedInventoryServiceServer
	lastReserve *inventoryv1.ReservationRequest
	lastRelease *inventoryv1.ReleaseStockRequest
	stock       *inventoryv1.StockLevel
}

func (s *fakeInventoryServer) ReserveStock(ctx context.Context, req *inventoryv1.ReservationRequest) (*inventoryv1.StockLevel, error) {
	s.lastReserve = req
	if s.stock == nil {
		return &inventoryv1.StockLevel{
			StoreId:           req.StoreId,
			ItemId:            req.ItemId,
			QuantityAvailable: 100,
			QuantityReserved:  req.Quantity,
		}, nil
	}
	return s.stock, nil
}

func (s *fakeInventoryServer) ReleaseStock(ctx context.Context, req *inventoryv1.ReleaseStockRequest) (*inventoryv1.StockLevel, error) {
	s.lastRelease = req
	return &inventoryv1.StockLevel{
		StoreId:           req.StoreId,
		ItemId:            req.ItemId,
		QuantityAvailable: 100,
	}, nil
}

func newFakeInventoryServer(t *testing.T) (*fakeInventoryServer, *grpc.ClientConn) {
	t.Helper()
	lis := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	fake := &fakeInventoryServer{}
	inventoryv1.RegisterInventoryServiceServer(server, fake)

	go func() {
		if err := server.Serve(lis); err != nil {
			t.Logf("fake inventory server error: %v", err)
		}
	}()
	t.Cleanup(func() { server.Stop() })

	ctx := context.Background()
	conn, err := grpc.DialContext(ctx, "bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	return fake, conn
}

func TestClient_Reserve(t *testing.T) {
	fake, conn := newFakeInventoryServer(t)
	client := &Client{conn: inventoryv1.NewInventoryServiceClient(conn), cc: conn}

	ctx := context.Background()
	stock, err := client.Reserve(ctx, "store-1", "kiosk-1", "item-1", "cart-1", 3, 1234567890)
	require.NoError(t, err)

	assert.Equal(t, int32(3), fake.lastReserve.Quantity)
	assert.Equal(t, "store-1", fake.lastReserve.StoreId)
	assert.Equal(t, "item-1", fake.lastReserve.ItemId)
	assert.Equal(t, "cart-1", fake.lastReserve.CartId)
	assert.Equal(t, int32(3), stock.QuantityReserved)
}

func TestClient_Release(t *testing.T) {
	fake, conn := newFakeInventoryServer(t)
	client := &Client{conn: inventoryv1.NewInventoryServiceClient(conn), cc: conn}

	ctx := context.Background()
	_, err := client.Release(ctx, "store-1", "item-1", "cart-1", 2, "customer removed item")
	require.NoError(t, err)

	assert.Equal(t, int32(2), fake.lastRelease.Quantity)
	assert.Equal(t, "customer removed item", fake.lastRelease.Reason)
}
