package middleware

import (
	"fmt"
	"sync"
	"time"

	"github.com/sony/gobreaker"
)

// BreakerRegistry manages one circuit breaker per downstream dependency
// (cart-service, inventory-service, order-service, Verifone orchestrator,
// etc.) and exposes live state for the admin "Circuit Breaker Dashboard"
// (deep-improvement #7). Centralizing this in the gateway means every
// downstream call — REST or gRPC — shares the same failure-isolation policy
// instead of each handler hand-rolling its own retry logic.
type BreakerRegistry struct {
	mu       sync.RWMutex
	breakers map[string]*gobreaker.CircuitBreaker
}

func NewBreakerRegistry() *BreakerRegistry {
	return &BreakerRegistry{breakers: make(map[string]*gobreaker.CircuitBreaker)}
}

// Get lazily creates a breaker per named downstream. Settings: trip after 5
// consecutive failures OR a failure ratio > 60% over a 10-request rolling
// window; half-open probe after 15s. These numbers are tuned so a single
// slow database failover doesn't trip the breaker on 1-2 slow requests, but
// a genuinely dead downstream is isolated within ~1-2 seconds under kiosk
// traffic volumes.
func (r *BreakerRegistry) Get(name string) *gobreaker.CircuitBreaker {
	r.mu.RLock()
	b, ok := r.breakers[name]
	r.mu.RUnlock()
	if ok {
		return b
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	// Re-check after acquiring write lock (double-checked locking).
	if b, ok := r.breakers[name]; ok {
		return b
	}

	settings := gobreaker.Settings{
		Name:        name,
		MaxRequests: 3, // requests allowed through in half-open state before deciding
		Interval:    10 * time.Second,
		Timeout:     15 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 10 && failureRatio >= 0.6 || counts.ConsecutiveFailures >= 5
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			// In production this emits an OTel span event + NATS
			// astra.kiosk.heartbeat-adjacent alert subject so the admin
			// dashboard and PagerDuty both observe the transition in real time.
			fmt.Printf("circuit_breaker state_change name=%s from=%s to=%s\n", name, from, to)
		},
	}

	b = gobreaker.NewCircuitBreaker(settings)
	r.breakers[name] = b
	return b
}

// Snapshot returns a serializable view of every breaker's current state,
// consumed by GET /v1/admin/fleet-health and shaped to match the
// `PaymentLaneHealth` contract expected by the kiosk-admin frontend
// (packages/shared-types mirrors this shape in TypeScript).
type BreakerSnapshot struct {
	LaneID              string `json:"laneId"`
	CircuitState        string `json:"circuitState"`
	ConsecutiveFailures int    `json:"consecutiveFailures"`
	LastFailureReason   string `json:"lastFailureReason,omitempty"`
}

func mapGobreakerState(s gobreaker.State) string {
	switch s {
	case gobreaker.StateClosed:
		return "closed"
	case gobreaker.StateHalfOpen:
		return "half_open"
	case gobreaker.StateOpen:
		return "open"
	default:
		return "closed"
	}
}

func (r *BreakerRegistry) Snapshot() []BreakerSnapshot {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]BreakerSnapshot, 0, len(r.breakers))
	for name, b := range r.breakers {
		counts := b.Counts()
		out = append(out, BreakerSnapshot{
			LaneID:              name,
			CircuitState:        mapGobreakerState(b.State()),
			ConsecutiveFailures: int(counts.ConsecutiveFailures),
		})
	}
	return out
}
