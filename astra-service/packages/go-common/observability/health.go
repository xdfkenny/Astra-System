package observability

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/astra-service/go-common/eventbus"
	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"
)

// Checkable is the interface implemented by every dependency health probe.
type Checkable interface {
	Check(ctx context.Context) error
}

// CheckFunc adapts a plain function to the Checkable interface.
type CheckFunc func(ctx context.Context) error

// Check implements Checkable.
func (f CheckFunc) Check(ctx context.Context) error { return f(ctx) }

// CompositeChecker runs registered health checks in parallel and returns the
// first failure. It is safe for concurrent use.
type CompositeChecker struct {
	mu     sync.RWMutex
	checks map[string]Checkable
}

// NewCompositeChecker creates an empty composite checker.
func NewCompositeChecker() *CompositeChecker {
	return &CompositeChecker{checks: make(map[string]Checkable)}
}

// Register adds or replaces a named dependency check.
func (c *CompositeChecker) Register(name string, check Checkable) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.checks[name] = check
}

// Check runs all registered checks in parallel with a 5-second timeout.
func (c *CompositeChecker) Check(ctx context.Context) error {
	c.mu.RLock()
	checks := make(map[string]Checkable, len(c.checks))
	for name, check := range c.checks {
		checks[name] = check
	}
	c.mu.RUnlock()

	if len(checks) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	type namedErr struct {
		name string
		err  error
	}

	errCh := make(chan namedErr, len(checks))
	var wg sync.WaitGroup
	for name, check := range checks {
		wg.Add(1)
		go func(name string, check Checkable) {
			defer wg.Done()
			if err := check.Check(ctx); err != nil {
				errCh <- namedErr{name: name, err: err}
			}
		}(name, check)
	}
	go func() {
		wg.Wait()
		close(errCh)
	}()

	for ne := range errCh {
		return fmt.Errorf("%s: %w", ne.name, ne.err)
	}
	return nil
}

// DBCheck verifies Postgres connectivity by pinging the supplied *sql.DB.
type DBCheck struct{ DB *sql.DB }

// Check implements Checkable.
func (d *DBCheck) Check(ctx context.Context) error {
	if d.DB == nil {
		return fmt.Errorf("database not configured")
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return d.DB.PingContext(ctx)
}

// RedisCheck verifies Redis connectivity.
type RedisCheck struct{ Client *redis.Client }

// Check implements Checkable.
func (r *RedisCheck) Check(ctx context.Context) error {
	if r.Client == nil {
		return fmt.Errorf("redis not configured")
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return r.Client.Ping(ctx).Err()
}

// NATSCheck verifies the NATS connection is active.
type NATSCheck struct{ Bus *eventbus.Bus }

// Check implements Checkable.
func (n *NATSCheck) Check(ctx context.Context) error {
	if n.Bus == nil {
		return fmt.Errorf("nats not configured")
	}
	if n.Bus.Status() != nats.CONNECTED {
		return fmt.Errorf("nats status: %v", n.Bus.Status())
	}
	return nil
}

// HealthHandler responds 200 {"status":"ok"} for load-balancer health pings.
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// LiveHandler responds 200 {"status":"alive"} for Kubernetes liveness probes.
func LiveHandler(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{"status": "alive"})
}

// ReadyHandler returns an http.HandlerFunc that runs the supplied checker for
// Kubernetes readiness probes.
func ReadyHandler(checker Checkable) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := checker.Check(r.Context()); err != nil {
			respondJSON(w, http.StatusServiceUnavailable, map[string]any{
				"status": "not_ready",
				"detail": err.Error(),
			})
			return
		}
		respondJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	}
}

func respondJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(body)
}
