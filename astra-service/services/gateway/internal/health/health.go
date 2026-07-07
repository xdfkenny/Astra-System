// Package health provides readiness and liveness checks for the gateway's
// external dependencies: PostgreSQL, Redis and NATS.
package health

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"
)

// Checker is the interface used by the readiness probe.
type Checker interface {
	Check(ctx context.Context) error
}

// CompositeChecker pings every dependency in parallel and reports the first
// failure.
type CompositeChecker struct {
	db    *sql.DB
	redis *redis.Client
	nats  *nats.Conn
}

// NewCompositeChecker builds a checker backed by real clients.
func NewCompositeChecker(db *sql.DB, redisClient *redis.Client, natsConn *nats.Conn) *CompositeChecker {
	return &CompositeChecker{db: db, redis: redisClient, nats: natsConn}
}

// Check verifies Postgres, Redis and NATS connectivity within a bounded time.
func (c *CompositeChecker) Check(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	errCh := make(chan error, 3)

	go func() {
		if c.db == nil {
			errCh <- fmt.Errorf("postgres: not configured")
			return
		}
		if err := c.db.PingContext(ctx); err != nil {
			errCh <- fmt.Errorf("postgres: %w", err)
			return
		}
		errCh <- nil
	}()

	go func() {
		if c.redis == nil {
			errCh <- fmt.Errorf("redis: not configured")
			return
		}
		if err := c.redis.Ping(ctx).Err(); err != nil {
			errCh <- fmt.Errorf("redis: %w", err)
			return
		}
		errCh <- nil
	}()

	go func() {
		if c.nats == nil {
			errCh <- fmt.Errorf("nats: not configured")
			return
		}
		if !c.nats.IsConnected() {
			errCh <- fmt.Errorf("nats: not connected")
			return
		}
		errCh <- nil
	}()

	for i := 0; i < 3; i++ {
		if err := <-errCh; err != nil {
			return err
		}
	}
	return nil
}
