# syntax=docker/dockerfile:1.7

# Astra-Service Legacy POS Adapter
# Multi-stage distroless build. Build context is the repository root.

ARG GO_VERSION=1.26.0
ARG ALPINE_VERSION=3.21

FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine${ALPINE_VERSION} AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

ENV GOWORK=off

COPY astra-service/packages/go-common/go.mod astra-service/packages/go-common/go.sum ./astra-service/packages/go-common/
COPY proto/go.mod proto/go.sum ./proto/
COPY astra-service/services/legacy-pos-adapter/go.mod astra-service/services/legacy-pos-adapter/go.sum ./astra-service/services/legacy-pos-adapter/

RUN --mount=type=cache,target=/go/pkg/mod \
    cd astra-service/services/legacy-pos-adapter && go mod download

COPY astra-service/packages/go-common/ ./astra-service/packages/go-common/
COPY proto/ ./proto/
COPY astra-service/services/legacy-pos-adapter/ ./astra-service/services/legacy-pos-adapter/

ARG TARGETOS
ARG TARGETARCH
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    cd astra-service/services/legacy-pos-adapter && \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -ldflags="-s -w -extldflags=-static" \
    -o /legacy-pos-adapter \
    ./cmd/legacy-pos-adapter

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /legacy-pos-adapter /legacy-pos-adapter

USER nonroot:nonroot

EXPOSE 8087 8088

STOPSIGNAL SIGTERM

ENTRYPOINT ["/legacy-pos-adapter"]
