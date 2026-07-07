// Package outbox builds domain event payloads for the transactional outbox.
// The actual persistence and relay are provided by go-common/outbox.
package outbox

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/astra-service/go-common/outbox"
	cartv1 "github.com/astra-systems/astra-service/proto/gen/go/cart"
	eventsv1 "github.com/astra-systems/astra-service/proto/gen/go/events"
	"github.com/astra-systems/astra-service/services/cart-service/internal/cart"
	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

const (
	EventTypeItemAddedToCart = "ItemAddedToCart"
	EventTypeCartFinalized   = "CartFinalized"
	AggregateTypeCart        = "cart"
)

// NewItemAddedToCart builds an outbox entry for an item added to a cart.
func NewItemAddedToCart(c *cart.Cart, line cart.Line) (outbox.Entry, error) {
	eventID := uuid.New().String()
	payload := &eventsv1.ItemAddedToCart{
		CartId:                 c.CartID,
		ItemId:                 line.MenuItemID,
		NameSnapshot:           line.NameSnapshot,
		Quantity:               int32(line.Quantity),
		UnitPriceCentsSnapshot: int64(line.UnitPriceCentsSnapshot),
	}
	data, err := marshalPayload(EventTypeItemAddedToCart, payload)
	if err != nil {
		return outbox.Entry{}, fmt.Errorf("outbox: marshal ItemAddedToCart: %w", err)
	}
	return outbox.Entry{
		EventID:       eventID,
		AggregateType: AggregateTypeCart,
		AggregateID:   c.CartID,
		EventType:     EventTypeItemAddedToCart,
		Payload:       data,
		OccurredAtMs:  c.UpdatedAtMs,
	}, nil
}

// NewCartFinalized builds an outbox entry for a finalized cart.
func NewCartFinalized(c *cart.Cart, orderID, currency string) (outbox.Entry, error) {
	eventID := uuid.New().String()
	payload := &eventsv1.CartFinalized{
		CartId:          c.CartID,
		OrderId:         orderID,
		FinalTotalCents: int64(c.FinalTotalCents),
		Currency:        currency,
	}
	data, err := marshalPayload(EventTypeCartFinalized, payload)
	if err != nil {
		return outbox.Entry{}, fmt.Errorf("outbox: marshal CartFinalized: %w", err)
	}
	return outbox.Entry{
		EventID:       eventID,
		AggregateType: AggregateTypeCart,
		AggregateID:   c.CartID,
		EventType:     EventTypeCartFinalized,
		Payload:       data,
		OccurredAtMs:  c.UpdatedAtMs,
	}, nil
}

// SubjectResolver maps cart outbox event types to NATS subjects.
func SubjectResolver(eventType string) string {
	switch eventType {
	case EventTypeItemAddedToCart:
		return "astra.cart.item_added.v1"
	case EventTypeCartFinalized:
		return "astra.cart.finalized.v1"
	default:
		return "astra.cart.unknown.v1"
	}
}

func marshalPayload(typeURL string, msg proto.Message) ([]byte, error) {
	anyMsg, err := anypb.New(msg)
	if err != nil {
		return nil, fmt.Errorf("anypb new: %w", err)
	}

	envelope := &eventsv1.EventEnvelope{
		EventId:        uuid.New().String(),
		AggregateId:    "",
		AggregateType:  AggregateTypeCart,
		SequenceNumber: 1,
		Timestamp:      time.Now().UTC().Format(time.RFC3339),
		Payload:        anyMsg,
		Metadata:       map[string]string{"event_type": typeURL},
	}
	return json.Marshal(envelope)
}

// cartLineProto is kept to satisfy any potential static imports.
var _ = cartv1.CartStatus_CART_STATUS_ACTIVE
