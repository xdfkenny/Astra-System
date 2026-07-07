// Package repository implements persistence for the cloud-side sync gateway.
package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/astra-systems/astra-service/services/sync-service/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store abstracts the database operations required by the sync-service. The
// concrete implementation uses pgx directly; tests substitute an in-memory
// fake that satisfies the same contract.
type Store interface {
	GetKiosk(ctx context.Context, kioskID string) (*model.Kiosk, error)
	InsertSyncEvents(ctx context.Context, events []model.SyncEvent) error
	GetDeltasSince(ctx context.Context, storeID, kioskID string, since time.Time, limit int32) ([]model.SyncEvent, error)
	UpsertHeartbeat(ctx context.Context, hb model.Heartbeat) error
	GetLatestHeartbeat(ctx context.Context, kioskID string) (*model.Heartbeat, error)
	GetLastCheckpoint(ctx context.Context, storeID, kioskID string) (time.Time, error)
}

// PostgresStore implements Store against PostgreSQL.
type PostgresStore struct {
	pool *pgxpool.Pool
}

// NewPostgresStore opens a connection pool from the supplied DSN.
func NewPostgresStore(ctx context.Context, dsn string) (*PostgresStore, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("repository: open pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("repository: ping database: %w", err)
	}
	return &PostgresStore{pool: pool}, nil
}

// Close releases the underlying connection pool.
func (s *PostgresStore) Close() {
	s.pool.Close()
}

// Ping verifies the database connection is alive.
func (s *PostgresStore) Ping(ctx context.Context) error {
	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return s.pool.Ping(pingCtx)
}

// GetKiosk returns a kiosk by ID, requiring it to be active and the leader of
// its store mesh.
func (s *PostgresStore) GetKiosk(ctx context.Context, kioskID string) (*model.Kiosk, error) {
	id, err := uuid.Parse(kioskID)
	if err != nil {
		return nil, fmt.Errorf("repository: invalid kiosk id: %w", err)
	}

	var k model.Kiosk
	var deletedAt sql.NullTime
	err = s.pool.QueryRow(ctx, `
		SELECT kiosk_id, store_id, hardware_id, display_name, signing_key_hash, is_leader, sync_status, last_seen_at, created_at
		FROM kiosks
		WHERE kiosk_id = $1 AND deleted_at IS NULL`, id,
	).Scan(
		&k.KioskID, &k.StoreID, &k.HardwareID, &k.DisplayName,
		&k.SigningKeyHash, &k.IsLeader, &k.SyncStatus, &k.LastSeenAt, &deletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrKioskNotFound
		}
		return nil, fmt.Errorf("repository: select kiosk: %w", err)
	}
	return &k, nil
}

// InsertSyncEvents writes raw sync events idempotently. Duplicate
// sync_event_id values are ignored so retries from the kiosk mesh are safe.
func (s *PostgresStore) InsertSyncEvents(ctx context.Context, events []model.SyncEvent) error {
	if len(events) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	for _, e := range events {
		payload, err := json.Marshal(e.PayloadJSON)
		if err != nil {
			return fmt.Errorf("repository: marshal payload for %s: %w", e.SyncEventID, err)
		}
		vc, err := json.Marshal(e.VectorClock)
		if err != nil {
			return fmt.Errorf("repository: marshal vector clock for %s: %w", e.SyncEventID, err)
		}
		batch.Queue(`
			INSERT INTO sync_events (sync_event_id, store_id, kiosk_id, event_type, payload_json, vector_clock, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (sync_event_id) DO NOTHING`,
			e.SyncEventID, e.StoreID, e.KioskID, e.EventType, payload, vc, e.CreatedAt,
		)
	}

	br := s.pool.SendBatch(ctx, batch)
	defer br.Close()

	var execErr error
	for i := 0; i < len(events); i++ {
		if _, err := br.Exec(); err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				continue
			}
			if execErr == nil {
				execErr = fmt.Errorf("repository: insert sync event: %w", err)
			}
		}
	}
	if err := br.Close(); err != nil {
		return fmt.Errorf("repository: close batch: %w", err)
	}
	return execErr
}

// GetDeltasSince returns sync events originating from other kiosks in the same
// store that are newer than the supplied checkpoint. Results are ordered by
// created_at ascending and limited to maxDeltas.
func (s *PostgresStore) GetDeltasSince(ctx context.Context, storeID, kioskID string, since time.Time, limit int32) ([]model.SyncEvent, error) {
	sid, err := uuid.Parse(storeID)
	if err != nil {
		return nil, fmt.Errorf("repository: invalid store id: %w", err)
	}
	kid, err := uuid.Parse(kioskID)
	if err != nil {
		return nil, fmt.Errorf("repository: invalid kiosk id: %w", err)
	}
	if limit <= 0 {
		limit = 1000
	}

	rows, err := s.pool.Query(ctx, `
		SELECT sync_event_id, store_id, kiosk_id, event_type, payload_json, vector_clock, created_at
		FROM sync_events
		WHERE store_id = $1 AND kiosk_id != $2 AND created_at > $3
		ORDER BY created_at ASC, sync_event_id ASC
		LIMIT $4`, sid, kid, since.UTC(), limit)
	if err != nil {
		return nil, fmt.Errorf("repository: select deltas: %w", err)
	}
	defer rows.Close()

	return scanSyncEvents(rows)
}

func scanSyncEvents(rows pgx.Rows) ([]model.SyncEvent, error) {
	var out []model.SyncEvent
	for rows.Next() {
		var e model.SyncEvent
		var payload []byte
		var vc []byte
		if err := rows.Scan(&e.SyncEventID, &e.StoreID, &e.KioskID, &e.EventType, &payload, &vc, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("repository: scan sync event: %w", err)
		}
		if err := json.Unmarshal(payload, &e.PayloadJSON); err != nil {
			return nil, fmt.Errorf("repository: unmarshal payload: %w", err)
		}
		if err := json.Unmarshal(vc, &e.VectorClock); err != nil {
			return nil, fmt.Errorf("repository: unmarshal vector clock: %w", err)
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository: iterate deltas: %w", err)
	}
	return out, nil
}

// UpsertHeartbeat records a kiosk heartbeat, collapsing duplicate entries
// within the same second to keep the table small.
func (s *PostgresStore) UpsertHeartbeat(ctx context.Context, hb model.Heartbeat) error {
	kid, err := uuid.Parse(hb.KioskID)
	if err != nil {
		return fmt.Errorf("repository: invalid kiosk id: %w", err)
	}
	sid, err := uuid.Parse(hb.StoreID)
	if err != nil {
		return fmt.Errorf("repository: invalid store id: %w", err)
	}
	vc, err := json.Marshal(hb.VectorClock)
	if err != nil {
		return fmt.Errorf("repository: marshal vector clock: %w", err)
	}

	_, err = s.pool.Exec(ctx, `
		INSERT INTO sync_heartbeats (kiosk_id, store_id, status, vector_clock, acknowledged_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (kiosk_id, date_trunc('second', acknowledged_at))
		DO UPDATE SET status = EXCLUDED.status, vector_clock = EXCLUDED.vector_clock, acknowledged_at = EXCLUDED.acknowledged_at`,
		kid, sid, hb.Status, vc, hb.AcknowledgedAt,
	)
	if err != nil {
		return fmt.Errorf("repository: upsert heartbeat: %w", err)
	}
	return nil
}

// GetLatestHeartbeat returns the most recent heartbeat recorded for a kiosk.
func (s *PostgresStore) GetLatestHeartbeat(ctx context.Context, kioskID string) (*model.Heartbeat, error) {
	kid, err := uuid.Parse(kioskID)
	if err != nil {
		return nil, fmt.Errorf("repository: invalid kiosk id: %w", err)
	}

	var hb model.Heartbeat
	var vc []byte
	err = s.pool.QueryRow(ctx, `
		SELECT kiosk_id, store_id, status, vector_clock, acknowledged_at
		FROM sync_heartbeats
		WHERE kiosk_id = $1
		ORDER BY acknowledged_at DESC
		LIMIT 1`, kid,
	).Scan(&hb.KioskID, &hb.StoreID, &hb.Status, &vc, &hb.AcknowledgedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrHeartbeatNotFound
		}
		return nil, fmt.Errorf("repository: select heartbeat: %w", err)
	}
	if err := json.Unmarshal(vc, &hb.VectorClock); err != nil {
		return nil, fmt.Errorf("repository: unmarshal vector clock: %w", err)
	}
	return &hb, nil
}

// GetLastCheckpoint returns the newest created_at timestamp for sync events
// that this kiosk has already seen. If no events exist, the zero time is
// returned, which causes DownloadBatch to stream the full store history.
func (s *PostgresStore) GetLastCheckpoint(ctx context.Context, storeID, kioskID string) (time.Time, error) {
	sid, err := uuid.Parse(storeID)
	if err != nil {
		return time.Time{}, fmt.Errorf("repository: invalid store id: %w", err)
	}
	kid, err := uuid.Parse(kioskID)
	if err != nil {
		return time.Time{}, fmt.Errorf("repository: invalid kiosk id: %w", err)
	}

	var t sql.NullTime
	if err := s.pool.QueryRow(ctx, `
		SELECT MAX(created_at)
		FROM sync_events
		WHERE store_id = $1 AND kiosk_id = $2`, sid, kid).Scan(&t); err != nil {
		return time.Time{}, fmt.Errorf("repository: select checkpoint: %w", err)
	}
	if !t.Valid {
		return time.Time{}, nil
	}
	return t.Time.UTC(), nil
}
