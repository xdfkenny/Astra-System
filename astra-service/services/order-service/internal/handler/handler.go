// Package handler adapts NATS JetStream messages into order-service use
// cases. Consumers are explicit-ack so a crash between receipt and commit never
// silently drops an event.
package handler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/astra-service/go-common/eventbus"
	"github.com/astra-systems/astra-service/services/order-service/internal/service"
	eventsv1 "github.com/astra-systems/astra-service/proto/gen/go/events"
	"github.com/nats-io/nats.go/jetstream"
)

// EventHandler wires JetStream consumers to the order service.
type EventHandler struct {
	service *service.OrderService
	bus     *eventbus.Bus
}

// NewEventHandler returns a handler bound to the supplied service and bus.
func NewEventHandler(svc *service.OrderService, bus *eventbus.Bus) *EventHandler {
	return &EventHandler{service: svc, bus: bus}
}

// RegisterConsumers creates durable consumers for the order-relevant domain
// events and returns the consume contexts so callers can stop them on shutdown.
func (h *EventHandler) RegisterConsumers(ctx context.Context) ([]jetstream.ConsumeContext, error) {
	var contexts []jetstream.ConsumeContext

	cartConsumer, err := h.bus.Subscribe(ctx, "ASTRA_CART", "order-service-cart-finalized",
		"astra.cart.finalized.v1", h.HandleCartFinalized)
	if err != nil {
		return nil, fmt.Errorf("handler: subscribe cart finalized: %w", err)
	}
	contexts = append(contexts, cartConsumer)

	paymentConsumer, err := h.bus.Subscribe(ctx, "ASTRA_PAYMENT", "order-service-payment-confirmed",
		"astra.payment.confirmed.v1", h.HandlePaymentConfirmed)
	if err != nil {
		for _, c := range contexts {
			c.Stop()
		}
		return nil, fmt.Errorf("handler: subscribe payment confirmed: %w", err)
	}
	contexts = append(contexts, paymentConsumer)

	return contexts, nil
}

// HandleCartFinalized converts a finalized cart into an order. It is
// idempotent: duplicate events for the same cart return the existing order.
func (h *EventHandler) HandleCartFinalized(ctx context.Context, msg jetstream.Msg) error {
	var evt eventsv1.CartFinalized
	if err := json.Unmarshal(msg.Data(), &evt); err != nil {
		return fmt.Errorf("handler: unmarshal CartFinalized: %w", err)
	}
	if _, err := h.service.HandleCartFinalized(ctx, &evt); err != nil {
		return fmt.Errorf("handler: handle CartFinalized: %w", err)
	}
	return nil
}

// HandlePaymentConfirmed transitions a pending order to paid when a payment
// authorization or capture is confirmed.
func (h *EventHandler) HandlePaymentConfirmed(ctx context.Context, msg jetstream.Msg) error {
	var evt eventsv1.PaymentConfirmed
	if err := json.Unmarshal(msg.Data(), &evt); err != nil {
		return fmt.Errorf("handler: unmarshal PaymentConfirmed: %w", err)
	}
	if err := h.service.HandlePaymentConfirmed(ctx, &evt); err != nil {
		return fmt.Errorf("handler: handle PaymentConfirmed: %w", err)
	}
	return nil
}
