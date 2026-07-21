# Communication Protocols

## Overview

Astra-System uses multiple protocols for different communication needs, chosen for their specific characteristics.

## Protocol Stack

```
┌──────────────────────────────────────────────────────────┐
│                    Application Layer                      │
│  HTTP/REST │ gRPC │ GraphQL │ WebRTC │ SSE │ WebSocket   │
├──────────────────────────────────────────────────────────┤
│                   Service Mesh Layer                      │
│  gRPC (inter-service) │ NATS (async events)              │
├──────────────────────────────────────────────────────────┤
│                    Transport Layer                        │
│  TCP/TLS │ QUIC │ HTTP/2 │ mDNS (UDP)                    │
├──────────────────────────────────────────────────────────┤
│                    Security Layer                         │
│  mTLS (TLS 1.3) │ Noise Protocol │ HMAC-SHA256           │
├──────────────────────────────────────────────────────────┤
│                    Network Layer                          │
│  IPv4/IPv6 │ TCP │ UDP                                   │
└──────────────────────────────────────────────────────────┘
```

## Protocol Summary

| Protocol | Use Case | Transport | Port(s) | Security |
|----------|----------|-----------|---------|----------|
| HTTP/REST | External API, browser clients | TCP/TLS 1.3 | 443 (ext), 8080 (int) | HTTPS, JWT, HMAC |
| gRPC | Inter-service communication | HTTP/2 over mTLS | Dynamic | mTLS (TLS 1.3) |
| NATS JetStream | Async event streaming | TCP/TLS | 4222 | TLS, NATS auth |
| WebRTC | Ghost cart P2P transfer | QUIC/DTLS | Ephemeral | DTLS-SRTP |
| SSE (Server-Sent Events) | Real-time updates (menu, inventory) | HTTP/TLS | 8080 | JWT |
| WebSocket | Bidirectional real-time | HTTP Upgrade | 8080 | WSS |
| QUIC | P2P mesh transport | UDP | 4499 | Noise Protocol |
| mDNS | Peer discovery | UDP multicast | 5353 | None (link-local) |
| GraphQL | Admin dashboard API | HTTP/TLS | 8092 | JWT |

## Detailed Protocol Descriptions

### HTTP/REST

**Used for:** External client → API Gateway, browser → kiosk backend

**Characteristics:**
- JSON request/response bodies
- Idempotency via `Idempotency-Key` header
- Consistent error format
- JWT Bearer token or HMAC-signed headers

### gRPC

**Used for:** Inter-service communication, sync daemon → cloud service

**Characteristics:**
- Protocol Buffers for serialization
- HTTP/2 transport with multiplexing
- Bi-directional streaming support
- mTLS for authentication and encryption
- gRPC-Gateway for REST transcoding

**Services:** Cart, Order, Payment, Sync, Inventory, Menu, Auth, Lane

### NATS JetStream

**Used for:** Async event bus, transactional outbox relay

**Characteristics:**
- At-least-once delivery guarantees
- Message persistence and replay
- Stream-based consumers
- Exactly-once semantics via outbox pattern

**Topics:** `astra.cart.*`, `astra.order.*`, `astra.inventory.*`, `astra.payment.*`, `astra.sync.*`

### WebRTC

**Used for:** Ghost cart transfer between kiosks

**Characteristics:**
- Peer-to-peer data channel
- QR code signaling for SDP exchange
- NFC fallback for NDEF message exchange
- Encrypted via DTLS-SRTP

**Flow:**
```
Kiosk A (offer) → QR encodes SDP offer → Kiosk B scans → SDP answer
  → ICE candidates exchanged → Data channel established
  → Cart snapshot transferred → Valtio merge → CRDT reconciliation
```

### SSE (Server-Sent Events)

**Used for:** Real-time menu and inventory updates

**Endpoints:**
- `GET /v1/menu/stream` - Menu updates
- `GET /stream` - Inventory stock updates

### QUIC

**Used for:** P2P mesh transport in sync daemon

**Characteristics:**
- Multiplexed streams (no head-of-line blocking)
- 0-RTT reconnection
- Connection migration (survives network changes)
- Built-in encryption (Noise Protocol)

### mDNS

**Used for:** Zero-config peer discovery

**Service type:** `_astra-sync._udp.local`

**Characteristics:**
- Link-local multicast (no router needed)
- Automatic peer discovery on same subnet
- Periodic advertisement and query
