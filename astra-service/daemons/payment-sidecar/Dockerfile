# syntax=docker/dockerfile:1.7

# Astra-Service Payment Sidecar
# Local Verifone bridge. Runs as a loopback-only sidecar on the kiosk.
# Multi-stage build with distroless base.

ARG RUST_VERSION=1.86
ARG ALPINE_VERSION=3.21

# ───────────────────────────────────────────────
# Stage 1: Build
# ───────────────────────────────────────────────
FROM --platform=$BUILDPLATFORM rust:${RUST_VERSION}-alpine${ALPINE_VERSION} AS builder

RUN apk add --no-cache musl-dev pkgconfig openssl-dev ca-certificates

WORKDIR /build
COPY astra-service/daemons/payment-sidecar/Cargo.toml astra-service/daemons/payment-sidecar/Cargo.lock* ./
COPY astra-service/daemons/payment-sidecar/src ./src

# Static linking produces a single binary with no dynamic musl dependency,
# which is required for distroless/static.
ENV RUSTFLAGS="-C target-feature=+crt-static"
RUN --mount=type=cache,target=/usr/local/cargo/registry \
    --mount=type=cache,target=/build/target \
    cargo build --release && \
    cp target/release/astra-payment-sidecar /tmp/astra-payment-sidecar

# ───────────────────────────────────────────────
# Stage 2: Distroless runtime
# ───────────────────────────────────────────────
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /tmp/astra-payment-sidecar /usr/local/bin/astra-payment-sidecar

USER nonroot:nonroot
EXPOSE 8963
STOPSIGNAL SIGTERM
ENTRYPOINT ["/usr/local/bin/astra-payment-sidecar"]
CMD ["--host", "127.0.0.1", "--port", "8963"]
