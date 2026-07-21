# Legacy POS Adapter

## Overview

The Legacy POS Adapter implements the **Strangler Fig pattern** to enable gradual migration from legacy Point-of-Sale systems to Astra-System without a big-bang cutover.

## Architecture

```
Legacy POS Terminal
    │ (existing protocol)
    ▼
Legacy POS Adapter (Go)
    │ REST / message bridge
    ▼
Astra-System Backend
    │
    ▼
Eventually: Legacy POS decommissioned
```

## How It Works

1. **Proxy mode:** Legacy POS requests are proxied through the adapter to Astra-System
2. **Translation:** Legacy data formats are translated to Astra-System protobuf/JSON schemas
3. **Message bridge:** Events are published to NATS for Astra-System consumption
4. **Gradual migration:** Features migrate one by one from legacy to Astra-System
5. **Cutover:** When all features are migrated, the adapter is removed

## Key Features

- REST adapter for legacy POS HTTP APIs
- Message bridge for event synchronization
- Data format translation
- Fallback to legacy system when Astra-System is unavailable
- Monitoring and metrics for migration tracking
