# syntax=docker/dockerfile:1.7

# Astra-Service P2P Sync Daemon
# Production-grade Rust daemon for kiosk mesh synchronization.
# Multi-stage build with distroless base.

ARG RUST_VERSION=1.86
ARG ALPINE_VERSION=3.21

# ───────────────────────────────────────────────
# Stage 1: Build
# ───────────────────────────────────────────────
FROM --platform=$BUILDPLATFORM rust:${RUST_VERSION}-alpine${ALPINE_VERSION} AS builder

# protoc is required to compile the gRPC/protobuf definitions.
RUN apk add --no-cache musl-dev pkgconfig openssl-dev ca-certificates protobuf-dev

WORKDIR /build
COPY astra-service/sync-daemon/Cargo.toml astra-service/sync-daemon/Cargo.lock* ./
COPY astra-service/sync-daemon/build.rs ./
COPY astra-service/sync-daemon/proto ./proto
COPY astra-service/sync-daemon/src ./src

ENV RUSTFLAGS="-C target-feature=+crt-static"
RUN --mount=type=cache,target=/usr/local/cargo/registry \
    --mount=type=cache,target=/build/target \
    cargo build --release && \
    cp target/release/astra-syncd /tmp/astra-syncd

# ───────────────────────────────────────────────
# Stage 2: Distroless runtime
# ───────────────────────────────────────────────
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /tmp/astra-syncd /usr/local/bin/astra-syncd

USER nonroot:nonroot
EXPOSE 4499
STOPSIGNAL SIGTERM
ENTRYPOINT ["/usr/local/bin/astra-syncd"]
CMD ["--config", "/etc/astra-syncd/config.toml"]
