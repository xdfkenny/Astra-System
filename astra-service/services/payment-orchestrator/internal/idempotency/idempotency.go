// Package idempotency implements idempotency-key handling backed by Redis for
// request locking and Postgres for durable result storage.
package idempotency

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// Result is the persisted outcome of an idempotent request.
type Result struct {
	PaymentID string          `json:"payment_id"`
	Status    string          `json:"status"`
	Payload   json.RawMessage `json:"payload"`
}

// Store coordinates Redis and Postgres idempotency storage.
type Store struct {
	db  *sql.DB
	rdb *redis.Client
	ttl time.Duration
}

// NewStore creates an idempotency store.
func NewStore(db *sql.DB, rdb *redis.Client) *Store {
	return &Store{
		db:  db,
		rdb: rdb,
		ttl: 24 * time.Hour,
	}
}

// Lock attempts to acquire a Redis lock for the idempotency key. It returns
// true when the caller is the first to process this key, false when another
// request is in flight, and an error for Redis failures.
func (s *Store) Lock(ctx context.Context, key uuid.UUID) (bool, error) {
	if s.rdb == nil {
		return true, nil
	}
	lockKey := "idempotency:lock:" + key.String()
	ok, err := s.rdb.SetNX(ctx, lockKey, "1", 30*time.Second).Result()
	if err != nil {
		return false, fmt.Errorf("idempotency: redis lock: %w", err)
	}
	return ok, nil
}

// Unlock releases the Redis lock for an idempotency key.
func (s *Store) Unlock(ctx context.Context, key uuid.UUID) error {
	if s.rdb == nil {
		return nil
	}
	lockKey := "idempotency:lock:" + key.String()
	if err := s.rdb.Del(ctx, lockKey).Err(); err != nil {
		return fmt.Errorf("idempotency: redis unlock: %w", err)
	}
	return nil
}

// GetResult returns a previously stored result by idempotency key, if any.
func (s *Store) GetResult(ctx context.Context, key uuid.UUID) (*Result, error) {
	var result Result
	err := s.db.QueryRowContext(ctx, `
		SELECT result FROM idempotency_results
		WHERE idempotency_key = $1 AND expires_at > NOW()`, key,
	).Scan(&result)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("idempotency: get result: %w", err)
	}
	return &result, nil
}

// SaveResult stores a durable result for an idempotency key.
func (s *Store) SaveResult(ctx context.Context, tx *sql.Tx, key uuid.UUID, r *Result) error {
	payload, err := json.Marshal(r)
	if err != nil {
		return fmt.Errorf("idempotency: marshal result: %w", err)
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO idempotency_results (idempotency_key, result, expires_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (idempotency_key) DO UPDATE SET result = EXCLUDED.result, expires_at = EXCLUDED.expires_at`,
		key, payload, time.Now().UTC().Add(s.ttl),
	)
	if err != nil {
		return fmt.Errorf("idempotency: save result: %w", err)
	}
	return nil
}

// Fingerprint returns a deterministic hash of the request body used to detect
// idempotency-key reuse with different payloads.
func Fingerprint(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}
