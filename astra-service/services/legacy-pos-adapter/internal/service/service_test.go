package service

import (
	"context"
	"errors"
	"testing"

	orderv1 "github.com/astra-systems/astra-service/proto/gen/go/order"
	"github.com/astra-systems/astra-service/services/legacy-pos-adapter/internal/client"
	"github.com/astra-systems/astra-service/services/legacy-pos-adapter/internal/repository"
)

type mockSubmitter struct {
	resp *client.LegacyPOSResponse
	err  error
}

func (m *mockSubmitter) Submit(_ context.Context, _ client.LegacyPOSRequest) (*client.LegacyPOSResponse, error) {
	return m.resp, m.err
}

func (m *mockSubmitter) BaseURL() string { return "http://mock" }

func TestAdapterService_HandleOrderCreated_Disabled(t *testing.T) {
	repo := repository.NewMemoryRepository()
	svc := New(repo, nil, false)

	order := &orderv1.Order{OrderId: "order-1", CartId: "cart-1", StoreId: "store-1", KioskId: "kiosk-1"}
	sub, err := svc.HandleOrderCreated(context.Background(), order)
	if err != nil {
		t.Fatalf("handle: %v", err)
	}
	if sub.OrderID != "order-1" {
		t.Fatalf("expected order-1, got %s", sub.OrderID)
	}
	if sub.Error != "legacy pos integration disabled" {
		t.Fatalf("expected disabled error, got %s", sub.Error)
	}

	stored, err := repo.GetSubmission(context.Background(), sub.SubmissionID)
	if err != nil {
		t.Fatalf("get submission: %v", err)
	}
	if stored.SubmissionID != sub.SubmissionID {
		t.Fatalf("submission mismatch")
	}
}

func TestAdapterService_HandleOrderCreated_Success(t *testing.T) {
	repo := repository.NewMemoryRepository()
	m := &mockSubmitter{resp: &client.LegacyPOSResponse{StatusCode: 201, Accepted: true, POSOrderID: "pos-123", Body: []byte(`{"pos_order_id":"pos-123"}`)}}
	svc := New(repo, m, true)

	order := &orderv1.Order{
		OrderId:    "order-1",
		CartId:     "cart-1",
		StoreId:    "store-1",
		KioskId:    "kiosk-1",
		TotalCents: 2500,
		Items: []*orderv1.OrderItem{
			{ItemId: "item-1", NameSnapshot: "Burger", Quantity: 1, PriceCentsSnapshot: 1500, LineTotalCents: 1500},
			{ItemId: "item-2", NameSnapshot: "Fries", Quantity: 1, PriceCentsSnapshot: 1000, LineTotalCents: 1000},
		},
	}

	sub, err := svc.HandleOrderCreated(context.Background(), order)
	if err != nil {
		t.Fatalf("handle: %v", err)
	}
	if sub.StatusCode != 201 {
		t.Fatalf("expected 201, got %d", sub.StatusCode)
	}
	if sub.Error != "" {
		t.Fatalf("expected no error, got %s", sub.Error)
	}

	list, err := repo.ListSubmissionsByOrder(context.Background(), "order-1")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 submission, got %d", len(list))
	}
}

func TestAdapterService_HandleOrderCreated_ClientError(t *testing.T) {
	repo := repository.NewMemoryRepository()
	m := &mockSubmitter{err: errors.New("connection refused")}
	svc := New(repo, m, true)

	order := &orderv1.Order{OrderId: "order-1"}
	sub, err := svc.HandleOrderCreated(context.Background(), order)
	if err != nil {
		t.Fatalf("handle: %v", err)
	}
	if sub.Error != "connection refused" {
		t.Fatalf("expected connection refused, got %s", sub.Error)
	}
}

func TestAdapterService_HandleOrderCreated_Rejected(t *testing.T) {
	repo := repository.NewMemoryRepository()
	m := &mockSubmitter{resp: &client.LegacyPOSResponse{StatusCode: 409, Accepted: false, Error: "duplicate", Body: []byte(`{"error":"duplicate"}`)}}
	svc := New(repo, m, true)

	order := &orderv1.Order{OrderId: "order-1"}
	sub, err := svc.HandleOrderCreated(context.Background(), order)
	if err != nil {
		t.Fatalf("handle: %v", err)
	}
	if sub.Error != "duplicate" {
		t.Fatalf("expected duplicate error, got %s", sub.Error)
	}
}

func TestAdapterService_HandleOrderCreated_MissingOrderID(t *testing.T) {
	svc := New(repository.NewMemoryRepository(), nil, false)
	_, err := svc.HandleOrderCreated(context.Background(), &orderv1.Order{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAdapterService_HandleOrderCreated_NilOrder(t *testing.T) {
	svc := New(repository.NewMemoryRepository(), nil, false)
	_, err := svc.HandleOrderCreated(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error")
	}
}
