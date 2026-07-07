// Package eventbus wraps NATS publication for ingested sync batches.
package eventbus

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/astra-service/go-common/eventbus"
)

// Publisher publishes domain events to NATS JetStream.
type Publisher interface {
	PublishBatchIngested(ctx context.Context, storeID, kioskID string, deltaCount int) error
}

// NATSPublisher implements Publisher using the shared eventbus Bus.
type NATSPublisher struct {
	bus *eventbus.Bus
}

// NewNATSPublisher returns a Publisher backed by the supplied Bus.
func NewNATSPublisher(bus *eventbus.Bus) *NATSPublisher {
	return &NATSPublisher{bus: bus}
}

// PublishBatchIngested emits a durable notification that a batch was accepted
// so downstream processors (analytics, inventory, payment settlement) can
// subscribe to the sync stream.
func (p *NATSPublisher) PublishBatchIngested(ctx context.Context, storeID, kioskID string, deltaCount int) error {
	payload := map[string]any{
		"store_id":    storeID,
		"kiosk_id":    kioskID,
		"delta_count": deltaCount,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("eventbus: marshal batch ingested: %w", err)
	}
	if err := p.bus.Publish(ctx, "astra.sync.batch_ingested", data); err != nil {
		return fmt.Errorf("eventbus: publish batch ingested: %w", err)
	}
	return nil
}

// Bus exposes the underlying eventbus handle for health checks and shutdown.
func (p *NATSPublisher) Bus() *eventbus.Bus {
	return p.bus
}
