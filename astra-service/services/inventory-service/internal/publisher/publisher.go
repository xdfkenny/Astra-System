// Package publisher abstracts the domain event publisher used by the
// inventory service.
package publisher

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/astra-service/go-common/eventbus"
	"github.com/google/uuid"
)

// Event describes a domain event to be published.
type Event struct {
	EventID      string
	EventType    string
	AggregateID  string
	Payload      map[string]any
	OccurredAtMs int64
}

// Publisher writes inventory domain events.
type Publisher interface {
	Publish(ctx context.Context, event Event) error
}

// NATSPublisher publishes events to a NATS JetStream subject.
type NATSPublisher struct {
	bus           *eventbus.Bus
	subjectPrefix string
}

// NewNATSPublisher returns a publisher that sends events to NATS.
func NewNATSPublisher(bus *eventbus.Bus, subjectPrefix string) *NATSPublisher {
	return &NATSPublisher{bus: bus, subjectPrefix: subjectPrefix}
}

// Publish sends the event as JSON to the configured subject.
func (n *NATSPublisher) Publish(ctx context.Context, event Event) error {
	if event.EventID == "" {
		event.EventID = uuid.New().String()
	}
	data, err := json.Marshal(event.Payload)
	if err != nil {
		return fmt.Errorf("publisher: marshal payload: %w", err)
	}
	subject := n.subjectPrefix + "." + event.EventType
	if err := n.bus.Publish(ctx, subject, data); err != nil {
		return fmt.Errorf("publisher: publish %s: %w", subject, err)
	}
	return nil
}

// MemoryPublisher records published events in memory for tests.
type MemoryPublisher struct {
	Events []Event
}

// NewMemoryPublisher returns a fresh in-memory publisher.
func NewMemoryPublisher() *MemoryPublisher {
	return &MemoryPublisher{}
}

// Publish stores the event in memory.
func (m *MemoryPublisher) Publish(ctx context.Context, event Event) error {
	if event.EventID == "" {
		event.EventID = uuid.New().String()
	}
	m.Events = append(m.Events, event)
	return nil
}
