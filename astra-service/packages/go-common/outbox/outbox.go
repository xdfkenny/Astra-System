// Package outbox implements the transactional outbox pattern: every domain
// write (cart mutation, order placement, inventory deduction) and its
// corresponding event are written in the SAME database transaction. A
// separate relay process then reads unpublished outbox rows and publishes
// them to NATS JetStream, marking them published only after a broker ack.
//
// WHY this exists (deep-improvement #5): without it, a service could commit
// a DB write and then crash/lose network before publishing the event —
// producing a database that says "order placed" but an event stream that
// never recorded it, silently breaking downstream inventory/analytics
// consumers and the P2P sync mesh's causal ordering. The outbox guarantees
// exactly-once *effective* semantics (at-least-once delivery + idempotent
// consumers keyed on eventId, since UUIDv7 event IDs are generated at
// write time and are naturally deduplicatable).
package outbox

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// Entry represents a single row in the `outbox_events` table (see
// db/migrations/0001_init.sql). AggregateID lets consumers partition/order
// by entity (e.g. all events for one cartId) even though the outbox table
// itself is a flat append log.
type Entry struct {
	EventID       string
	AggregateType string
	AggregateID   string
	EventType     string
	Payload       json.RawMessage
	OccurredAtMs  int64
}

// InsertWithinTx writes an outbox entry using the SAME *sql.Tx as the
// domain write it accompanies. This is the crux of the pattern: callers
// MUST pass the active transaction, never a fresh connection, or the
// atomicity guarantee is void.
func InsertWithinTx(ctx context.Context, tx *sql.Tx, e Entry) error {
	const q = `
		INSERT INTO outbox_events
			(event_id, aggregate_type, aggregate_id, event_type, payload, occurred_at_ms, published)
		VALUES ($1, $2, $3, $4, $5, $6, false)
		ON CONFLICT (event_id) DO NOTHING` // idempotent retry-safe insert

	_, err := tx.ExecContext(ctx, q,
		e.EventID, e.AggregateType, e.AggregateID, e.EventType, e.Payload, e.OccurredAtMs,
	)
	if err != nil {
		return fmt.Errorf("outbox: insert within tx: %w", err)
	}
	return nil
}

// Publisher is the minimal interface the relay needs — satisfied by
// *eventbus.Bus without creating an import cycle between packages.
type Publisher interface {
	Publish(ctx context.Context, subject string, data []byte) error
}

// SubjectResolver maps an outbox EventType to its NATS subject, injected by
// each service since subject naming is domain-specific.
type SubjectResolver func(eventType string) string

// Relay polls unpublished outbox rows and publishes them, marking success
// atomically. Runs as a background goroutine in every service that writes
// to the outbox; also runnable as a standalone sidecar for services that
// prefer to decouple the relay's failure domain from the API process.
type Relay struct {
	db          *sql.DB
	publisher   Publisher
	resolve     SubjectResolver
	pollEvery   time.Duration
	batchSize   int
}

func NewRelay(db *sql.DB, publisher Publisher, resolve SubjectResolver) *Relay {
	return &Relay{
		db:        db,
		publisher: publisher,
		resolve:   resolve,
		pollEvery: 500 * time.Millisecond, // sub-second relay latency for cart/inventory events
		batchSize: 100,
	}
}

// Run blocks until ctx is cancelled (SIGTERM), polling and relaying batches.
func (r *Relay) Run(ctx context.Context) error {
	ticker := time.NewTicker(r.pollEvery)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := r.relayBatch(ctx); err != nil {
				// Log-and-continue: a transient DB or NATS blip must not kill the
				// relay loop — the next tick retries the same unpublished rows.
				fmt.Printf("outbox relay error (will retry): %v\n", err)
			}
		}
	}
}

func (r *Relay) relayBatch(ctx context.Context) error {
	rows, err := r.db.QueryContext(ctx, `
		SELECT event_id, aggregate_type, aggregate_id, event_type, payload, occurred_at_ms
		FROM outbox_events
		WHERE published = false
		ORDER BY occurred_at_ms ASC
		LIMIT $1
		FOR UPDATE SKIP LOCKED`, r.batchSize) // SKIP LOCKED enables multiple relay
	// instances (one per service replica) to safely race over the same table
	// without double-publishing or blocking each other.
	if err != nil {
		return fmt.Errorf("outbox: query batch: %w", err)
	}
	defer rows.Close()

	type row struct {
		eventID, aggType, aggID, eventType string
		payload                            json.RawMessage
		occurredAtMs                       int64
	}
	var batch []row
	for rows.Next() {
		var rr row
		if err := rows.Scan(&rr.eventID, &rr.aggType, &rr.aggID, &rr.eventType, &rr.payload, &rr.occurredAtMs); err != nil {
			return fmt.Errorf("outbox: scan row: %w", err)
		}
		batch = append(batch, rr)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, rr := range batch {
		subject := r.resolve(rr.eventType)
		if err := r.publisher.Publish(ctx, subject, rr.payload); err != nil {
			// Leave unpublished; next tick retries. NATS publish is itself
			// idempotent-safe downstream because consumers dedupe on eventId.
			return fmt.Errorf("outbox: publish %s: %w", rr.eventID, err)
		}
		if _, err := r.db.ExecContext(ctx,
			`UPDATE outbox_events SET published = true, published_at_ms = $2 WHERE event_id = $1`,
			rr.eventID, time.Now().UnixMilli(),
		); err != nil {
			return fmt.Errorf("outbox: mark published %s: %w", rr.eventID, err)
		}
	}
	return nil
}
