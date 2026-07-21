// Package eventbus wraps NATS JetStream for the event-sourced, CQRS-style
// architecture shared by every Astra-Service backend microservice. We chose
// NATS JetStream over Kafka deliberately: it runs as a single ~20MB static
// binary that we can co-locate on kiosk hardware for the edge tier (Kafka's
// JVM footprint and ZooKeeper/KRaft operational overhead are non-starters
// on ARM64 kiosk SoCs), while still giving us durable, replayable streams
// with consumer acknowledgement semantics for the cloud tier.
package eventbus

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

func streamReplicas() int {
	if v := os.Getenv("NATS_STREAM_REPLICAS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return 3
}

// StreamConfig defines the durable JetStream streams Astra-Service depends on.
var StreamConfig = []jetstream.StreamConfig{
	{
		Name:      "ASTRA_CART",
		Subjects:  []string{"astra.cart.>"},
		Storage:   jetstream.FileStorage,
		Retention: jetstream.LimitsPolicy,
		MaxAge:    30 * 24 * time.Hour,
		Replicas: streamReplicas(),
	},
	{
		Name:      "ASTRA_INVENTORY",
		Subjects:  []string{"astra.inventory.>"},
		Storage:   jetstream.FileStorage,
		Retention: jetstream.LimitsPolicy,
		MaxAge:    90 * 24 * time.Hour,
		Replicas: streamReplicas(),
	},
	{
		Name:      "ASTRA_PAYMENT",
		Subjects:  []string{"astra.payment.>"},
		Storage:   jetstream.FileStorage,
		Retention: jetstream.LimitsPolicy,
		MaxAge:    7 * 365 * 24 * time.Hour, // PCI/financial audit retention
		Replicas: streamReplicas(),
	},
	{
		Name:      "ASTRA_ORDER",
		Subjects:  []string{"astra.order.>"},
		Storage:   jetstream.FileStorage,
		Retention: jetstream.LimitsPolicy,
		MaxAge:    7 * 365 * 24 * time.Hour,
		Replicas: streamReplicas(),
	},
	{
		Name:      "ASTRA_MENU",
		Subjects:  []string{"astra.menu.>"},
		Storage:   jetstream.FileStorage,
		Retention: jetstream.LimitsPolicy,
		MaxAge:    90 * 24 * time.Hour,
		Replicas: streamReplicas(),
	},
	{
		Name:      "ASTRA_SYSTEM",
		Subjects:  []string{"astra.sync.>", "astra.kiosk.>"},
		Storage:   jetstream.FileStorage,
		Retention: jetstream.LimitsPolicy,
		MaxAge:    7 * 24 * time.Hour,
		Replicas: streamReplicas(),
	},
}

// Bus is the shared JetStream client handle used by every service.
type Bus struct {
	nc *nats.Conn
	js jetstream.JetStream
}

// Connect establishes a NATS connection with production-grade reconnection
// semantics: unlimited reconnect attempts with capped backoff, because a
// kiosk's connection to the store's NATS cluster flapping must never crash
// the service — it must degrade to local-queue-and-retry.
func Connect(ctx context.Context, url string) (*Bus, error) {
	nc, err := nats.Connect(url,
		nats.MaxReconnects(-1),
		nats.ReconnectWait(2*time.Second),
		nats.ReconnectBufSize(8*1024*1024),
		nats.Timeout(5*time.Second),
		nats.RetryOnFailedConnect(true),
	)
	if err != nil {
		return nil, fmt.Errorf("eventbus: connect: %w", err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("eventbus: jetstream init: %w", err)
	}

	bus := &Bus{nc: nc, js: js}
	if err := bus.ensureStreams(ctx); err != nil {
		nc.Close()
		return nil, err
	}
	return bus, nil
}

func (b *Bus) ensureStreams(ctx context.Context) error {
	for _, cfg := range StreamConfig {
		if _, err := b.js.CreateOrUpdateStream(ctx, cfg); err != nil {
			return fmt.Errorf("eventbus: ensure stream %s: %w", cfg.Name, err)
		}
	}
	return nil
}

// Publish sends a message and waits for JetStream acknowledgement — this is
// the "at least once" durability guarantee every event-sourced write relies
// on. Callers combine this with the transactional outbox pattern (see
// packages/go-common/outbox) so a DB commit and an event publish are never
// silently divergent.
func (b *Bus) Publish(ctx context.Context, subject string, data []byte) error {
	_, err := b.js.Publish(ctx, subject, data)
	if err != nil {
		return fmt.Errorf("eventbus: publish %s: %w", subject, err)
	}
	return nil
}

// Subscribe creates a durable, explicitly-acked consumer. Explicit ack
// (rather than auto-ack) is mandatory here: a service must only ack after
// its own local transaction (e.g. inventory deduction) has committed, or a
// crash between receipt and processing would silently drop an event.
func (b *Bus) Subscribe(
	ctx context.Context,
	streamName, durableName, filterSubject string,
	handler func(ctx context.Context, msg jetstream.Msg) error,
) (jetstream.ConsumeContext, error) {
	stream, err := b.js.Stream(ctx, streamName)
	if err != nil {
		return nil, fmt.Errorf("eventbus: get stream %s: %w", streamName, err)
	}

	consumer, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:       durableName,
		FilterSubject: filterSubject,
		AckPolicy:     jetstream.AckExplicitPolicy,
		AckWait:       30 * time.Second,
		MaxDeliver:    5, // after 5 failed attempts, message routes to a dead-letter policy upstream
		DeliverPolicy: jetstream.DeliverAllPolicy,
	})
	if err != nil {
		return nil, fmt.Errorf("eventbus: create consumer %s: %w", durableName, err)
	}

	return consumer.Consume(func(msg jetstream.Msg) {
		if err := handler(ctx, msg); err != nil {
			_ = msg.Nak() // triggers redelivery with backoff up to MaxDeliver
			return
		}
		_ = msg.Ack()
	})
}

// Status returns the current NATS connection state. It is used by readiness
// probes in services that cannot serve traffic without an active event bus.
func (b *Bus) Status() nats.Status {
	return b.nc.Status()
}

// Close drains and closes the underlying NATS connection for graceful
// SIGTERM shutdown.
func (b *Bus) Close() {
	_ = b.nc.Drain()
}
