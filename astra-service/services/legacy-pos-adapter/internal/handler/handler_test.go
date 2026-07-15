package handler

import (
	"context"
	"testing"
	"time"

	eventsv1 "github.com/astra-systems/astra-service/proto/gen/go/events"
	orderv1 "github.com/astra-systems/astra-service/proto/gen/go/order"
	"github.com/astra-systems/astra-service/services/legacy-pos-adapter/internal/repository"
	"github.com/astra-systems/astra-service/services/legacy-pos-adapter/internal/service"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/anypb"
)

type fakeMsg struct {
	data []byte
}

func (f fakeMsg) Data() []byte                              { return f.data }
func (f fakeMsg) Subject() string                           { return "" }
func (f fakeMsg) Headers() nats.Header                      { return nil }
func (f fakeMsg) Reply() string                             { return "" }
func (f fakeMsg) Ack() error                                { return nil }
func (f fakeMsg) DoubleAck(context.Context) error           { return nil }
func (f fakeMsg) Nak() error                                { return nil }
func (f fakeMsg) NakWithDelay(_ time.Duration) error        { return nil }
func (f fakeMsg) InProgress() error                         { return nil }
func (f fakeMsg) Term() error                               { return nil }
func (f fakeMsg) TermWithReason(_ string) error             { return nil }
func (f fakeMsg) Metadata() (*jetstream.MsgMetadata, error) { return nil, nil }
func (f fakeMsg) Duplicate() bool                           { return false }

func TestHandleOrderCreated(t *testing.T) {
	orderCreated := &eventsv1.OrderCreated{
		OrderId:    uuid.New().String(),
		CartId:     "cart-1",
		StoreId:    "store-1",
		KioskId:    "kiosk-1",
		TotalCents: 1234,
		Currency:   "USD",
	}
	anyPayload, err := anypb.New(orderCreated)
	if err != nil {
		t.Fatalf("anypb new: %v", err)
	}
	envelope := &eventsv1.EventEnvelope{
		EventId:     uuid.New().String(),
		AggregateId: orderCreated.OrderId,
		Payload:     anyPayload,
	}
	data, err := protojson.Marshal(envelope)
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}

	repo := repository.NewMemoryRepository()
	svc := service.New(repo, nil, false)
	h := NewEventHandler(svc)

	if err := h.HandleOrderCreated(context.Background(), fakeMsg{data: data}); err != nil {
		t.Fatalf("handle order created: %v", err)
	}

	list, err := repo.ListSubmissionsByOrder(context.Background(), orderCreated.OrderId)
	if err != nil {
		t.Fatalf("list submissions: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 submission, got %d", len(list))
	}
	if list[0].OrderID != orderCreated.OrderId {
		t.Fatalf("expected order id %s, got %s", orderCreated.OrderId, list[0].OrderID)
	}
}

func TestHandleOrderCreated_InvalidPayload(t *testing.T) {
	repo := repository.NewMemoryRepository()
	svc := service.New(repo, nil, false)
	h := NewEventHandler(svc)

	if err := h.HandleOrderCreated(context.Background(), fakeMsg{data: []byte("not-json")}); err == nil {
		t.Fatal("expected error for invalid payload")
	}
}

// compile-time assertion that the handler can be wired with a disabled service.
var _ = NewEventHandler(service.New(repository.NewMemoryRepository(), nil, false))
var _ = (*orderv1.Order)(nil)
