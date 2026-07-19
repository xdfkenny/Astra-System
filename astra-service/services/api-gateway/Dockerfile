# syntax=docker/dockerfile:1.7

# Astra-Service API Gateway
# Production-grade Go/Fiber gateway with distroless base image
# Multi-stage build with BuildKit cache mounts

ARG GO_VERSION=1.25
ARG ALPINE_VERSION=3.21

# ───────────────────────────────────────────────
# Stage 1: Build
# ───────────────────────────────────────────────
FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine${ALPINE_VERSION} AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

ENV GOWORK=off

# Copy the module's own go.mod/go.sum plus the local go-common dependency
# (referenced via a replace directive). Building with GOWORK=off avoids pulling
# the entire monorepo workspace into the build context.
COPY astra-service/packages/go-common/go.mod astra-service/packages/go-common/go.sum ./astra-service/packages/go-common/
COPY astra-service/services/api-gateway/go.mod astra-service/services/api-gateway/go.sum ./astra-service/services/api-gateway/

RUN --mount=type=cache,target=/go/pkg/mod \
    cd astra-service/services/api-gateway && go mod download

COPY astra-service/packages/go-common/ ./astra-service/packages/go-common/
COPY astra-service/services/api-gateway/ ./astra-service/services/api-gateway/

ARG TARGETOS
ARG TARGETARCH
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    cd astra-service/services/api-gateway && \
    go build -ldflags="-s -w -extldflags=-static" \
    -o /tmp/api-gateway \
    ./cmd/gateway

# ───────────────────────────────────────────────
# Stage 2: Distroless runtime
# ───────────────────────────────────────────────
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /tmp/api-gateway /usr/local/bin/api-gateway

USER nonroot:nonroot

EXPOSE 8080
STOPSIGNAL SIGTERM

ENTRYPOINT ["/usr/local/bin/api-gateway"]
