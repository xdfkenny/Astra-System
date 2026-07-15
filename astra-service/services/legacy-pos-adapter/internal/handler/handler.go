// Package handler adapts NATS JetStream messages into legacy-pos-adapter use
// cases.
package handler

import (
	"context"
	"fmt"

	eventsv1 "github.com/astra-systems/astra-service/proto/gen/go/events"
	orderv1 "github.com/astra-systems/astra-service/proto/gen/go/order"
	"github.com/astra-systems/astra-service/services/legacy-pos-adapter/internal/service"
	"github.com/nats-io/nats.go/jetstream"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

// EventHandler wires JetStream consumers to the adapter service.
type EventHandler struct {
	service *service.AdapterService
}

// NewEventHandler returns a handler bound to the supplied service.
func NewEventHandler(svc *service.AdapterService) *EventHandler {
	return &EventHandler{service: svc}
}

// RegisterConsumers creates durable consumers for order-relevant domain events
// and returns the consume contexts so callers can stop them on shutdown.
func (h *EventHandler) RegisterConsumers(ctx context.Context, bus subscriber) ([]jetstream.ConsumeContext, error) {
	var contexts []jetstream.ConsumeContext

	orderConsumer, err := bus.Subscribe(ctx, "ASTRA_ORDER", "legacy-pos-adapter-order-created",
		"astra.order.created.v1", h.HandleOrderCreated)
	if err != nil {
		return nil, fmt.Errorf("handler: subscribe order created: %w", err)
	}
	contexts = append(contexts, orderConsumer)

	return contexts, nil
}

// HandleOrderCreated converts an OrderCreated event into a legacy POS submission.
func (h *EventHandler) HandleOrderCreated(ctx context.Context, msg jetstream.Msg) error {
	var envelope eventsv1.EventEnvelope
	if err := protojson.Unmarshal(msg.Data(), &envelope); err != nil {
		return fmt.Errorf("handler: unmarshal envelope: %w", err)
	}

	var order eventsv1.OrderCreated
	if err := anypb.UnmarshalTo(envelope.Payload, &order, proto.UnmarshalOptions{}); err != nil {
		return fmt.Errorf("handler: unmarshal OrderCreated payload: %w", err)
	}

	protoOrder := &orderv1.Order{
		OrderId:    order.OrderId,
		CartId:     order.CartId,
		StoreId:    order.StoreId,
		KioskId:    order.KioskId,
		TotalCents: order.TotalCents,
	}

	if _, err := h.service.HandleOrderCreated(ctx, protoOrder); err != nil {
		return fmt.Errorf("handler: process order: %w", err)
	}
	return nil
}

// subscriber is a minimal interface for NATS subscriptions, kept narrow so
// tests can provide a fake bus.
type subscriber interface {
	Subscribe(ctx context.Context, streamName, durableName, filterSubject string, handler func(ctx context.Context, msg jetstream.Msg) error) (jetstream.ConsumeContext, error)
}

// compile-time interface assertion.
var _ subscriber = (*testSubscriber)(nil)

type testSubscriber struct{}

func (testSubscriber) Subscribe(context.Context, string, string, string, func(context.Context, jetstream.Msg) error) (jetstream.ConsumeContext, error) {
	return nil, nil
}
