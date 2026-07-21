# syntax=docker/dockerfile:1.7

ARG GO_VERSION=1.25.2
ARG ALPINE_VERSION=3.21

FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine${ALPINE_VERSION} AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build
COPY astra-service/go.work astra-service/go.work.sum* ./astra-service/
COPY astra-service/packages/go-common/go.mod astra-service/packages/go-common/go.sum ./astra-service/packages/go-common/
COPY proto/go.mod proto/go.sum* ./proto/
COPY astra-service/services/payment-orchestrator/go.mod astra-service/services/payment-orchestrator/go.sum* ./astra-service/services/payment-orchestrator/

RUN --mount=type=cache,target=/go/pkg/mod \
    cd astra-service && go mod download -x

COPY astra-service/packages/go-common/ ./astra-service/packages/go-common/
COPY proto/ ./proto/
COPY astra-service/services/payment-orchestrator/ ./astra-service/services/payment-orchestrator/

ARG TARGETOS
ARG TARGETARCH
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -ldflags="-s -w -extldflags=-static" \
    -o /tmp/payment-orchestrator \
    ./astra-service/services/payment-orchestrator/cmd/payment-orchestrator

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /tmp/payment-orchestrator /usr/local/bin/payment-orchestrator
USER nonroot:nonroot
EXPOSE 8086 50086
STOPSIGNAL SIGTERM
ENTRYPOINT ["/usr/local/bin/payment-orchestrator"]
