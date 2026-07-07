// Package events builds canonical EventEnvelope protobuf messages for payment
// domain events and resolves them to NATS subjects.
package events

import (
	"fmt"
	"time"

	commonv1 "github.com/astra-systems/astra-service/proto/gen/go/common"
	eventsv1 "github.com/astra-systems/astra-service/proto/gen/go/events"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
)

const (
	// EventTypePaymentInitiated is emitted when a payment intent is accepted.
	EventTypePaymentInitiated = "PaymentInitiated"
	// EventTypePaymentConfirmed is emitted when a payment reaches a terminal or
	// intermediate confirmed state.
	EventTypePaymentConfirmed = "PaymentConfirmed"
	// EventTypePaymentFailed is emitted when a payment fails or is declined.
	EventTypePaymentFailed = "PaymentFailed"
)

const (
	// SubjectPaymentInitiated is the NATS JetStream subject for initiated payments.
	SubjectPaymentInitiated = "astra.payment.initiated"
	// SubjectPaymentConfirmed is the NATS JetStream subject for confirmed payments.
	SubjectPaymentConfirmed = "astra.payment.confirmed"
	// SubjectPaymentFailed is the NATS JetStream subject for failed payments.
	SubjectPaymentFailed = "astra.payment.failed"
)

// Envelope builds an EventEnvelope wrapping the supplied protobuf payload.
func Envelope(eventType, aggregateID string, seq int64, metadata map[string]string, payload any) (*eventsv1.EventEnvelope, error) {
	var a *anypb.Any
	switch p := payload.(type) {
	case *eventsv1.PaymentInitiated:
		var err error
		a, err = anypb.New(p)
		if err != nil {
			return nil, fmt.Errorf("events: pack PaymentInitiated: %w", err)
		}
	case *eventsv1.PaymentConfirmed:
		var err error
		a, err = anypb.New(p)
		if err != nil {
			return nil, fmt.Errorf("events: pack PaymentConfirmed: %w", err)
		}
	case *PaymentFailed:
		s, err := structpb.NewStruct(map[string]any{
			"payment_id":     p.PaymentID,
			"order_id":       p.OrderID,
			"kiosk_id":       p.KioskID,
			"status":         p.Status,
			"decline_reason": p.DeclineReason,
		})
		if err != nil {
			return nil, fmt.Errorf("events: build PaymentFailed struct: %w", err)
		}
		a, err = anypb.New(s)
		if err != nil {
			return nil, fmt.Errorf("events: pack PaymentFailed: %w", err)
		}
	default:
		return nil, fmt.Errorf("events: unsupported payload type %T", payload)
	}

	return &eventsv1.EventEnvelope{
		EventId:        uuid.Must(uuid.NewV7()).String(),
		AggregateId:    aggregateID,
		AggregateType:  "payment",
		SequenceNumber: seq,
		Timestamp:      time.Now().UTC().Format(time.RFC3339Nano),
		Payload:        a,
		Metadata:       metadata,
		Hlc:            &commonv1.HLC{PhysicalTimeMs: time.Now().UnixMilli(), LogicalCounter: 0, NodeId: "payment-orchestrator"},
	}, nil
}

// PaymentFailed is the domain event payload for failed payments. It is not
// defined in the shared events proto, so it is carried as a google.protobuf.Struct.
type PaymentFailed struct {
	PaymentID     string
	OrderID       string
	KioskID       string
	Status        string
	DeclineReason string
}

// Subject returns the NATS subject for a payment event type.
func Subject(eventType string) string {
	switch eventType {
	case EventTypePaymentInitiated:
		return SubjectPaymentInitiated
	case EventTypePaymentConfirmed:
		return SubjectPaymentConfirmed
	case EventTypePaymentFailed:
		return SubjectPaymentFailed
	default:
		return "astra.payment.unknown"
	}
}
