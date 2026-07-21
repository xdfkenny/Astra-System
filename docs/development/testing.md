# Testing Guide

## Test Layers

| Layer | Tool | Scope | Location |
|-------|------|-------|----------|
| Unit (TS) | Vitest + happy-dom | Components, hooks, utilities | `apps/*/src/`, `packages/*/src/` |
| Unit (Go) | `go test -race` | Services, libraries | All `services/*/`, `packages/go-common/` |
| Unit (Rust) | `cargo test` / `cargo nextest` | Sync daemon, sidecars | `sync-daemon/`, `daemons/` |
| Unit (Python) | pytest | ML models | `services/ml-lane-intel/tests/` |
| Integration | Docker Compose + test harness | Service interactions | CI |
| E2E | Playwright | Kiosk UI workflows | `apps/kiosk/e2e/`, `apps/kiosk-admin/e2e/` |
| Chaos | Rust chaos tool | Resilience verification | `tools/chaos/` |
| Database | Drizzle tests | Schema validation | `database/schemas/` |

## Running Tests

```bash
# All TypeScript tests
pnpm test

# TypeScript with coverage
pnpm test -- --coverage

# Go tests
cd astra-service && go test -race ./...

# Rust tests
cd astra-service/sync-daemon && cargo test

# Specific Go service
cd astra-service/services/gateway && go test -race ./...

# Specific Rust crate
cd astra-service/sync-daemon && cargo test -p astra-syncd-crdt

# E2E tests (requires full stack running)
pnpm test:e2e

# Python tests
cd astra-service/services/ml-lane-intel
uv run pytest
```

## Writing Tests

### TypeScript (Vitest)

```typescript
import { describe, it, expect } from 'vitest'
import { computeTotals } from './computeTotals'

describe('computeTotals', () => {
  it('calculates subtotal correctly', () => {
    const items = [{ quantity: 2, unitPriceCents: 499 }]
    const result = computeTotals(items)
    expect(result.subtotalCents).toBe(998)
  })
})
```

### Go

```go
func TestCreateOrder(t *testing.T) {
    svc := NewOrderService(mockRepo)
    order, err := svc.CreateOrder(context.Background(), validRequest)
    assert.NoError(t, err)
    assert.Equal(t, "pending", string(order.Status))
}
```

### Rust

```rust
#[cfg(test)]
mod tests {
    use super::*;
    
    #[test]
    fn test_crdt_merge() {
        let mut set = ORSet::new();
        set.add("item1");
        set.add("item2");
        set.remove("item1");
        assert!(set.contains("item2"));
        assert!(!set.contains("item1"));
    }
}
```

## Integration Testing

Integration tests require Docker running with PostgreSQL, Redis, and NATS:

```bash
# Start dependencies
docker compose up -d postgres redis nats

# Run integration tests
cd astra-service/services/order-service && go test -tags=integration ./...
```

## Chaos Testing

The chaos tool (`tools/chaos/`) injects faults:

```bash
cd astra-service/tools/chaos
cargo run -- -scenario network-partition -duration 30s
```

Scenarios:
- `network-partition` - Simulate network split between kiosks
- `disk-pressure` - Fill disk to test graceful degradation
- `memory-pressure` - Limit available memory
- `service-crash` - Randomly kill service processes
