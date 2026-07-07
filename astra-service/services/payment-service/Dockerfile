# syntax=docker/dockerfile:1.7

# Astra-Service Payment Service
ARG GO_VERSION=1.23
ARG ALPINE_VERSION=3.20

FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine${ALPINE_VERSION} AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build
COPY astra-service/go.work astra-service/go.work.sum* ./astra-service/
COPY astra-service/packages/go-common/go.mod astra-service/packages/go-common/go.sum ./astra-service/packages/go-common/
COPY astra-service/services/payment-service/go.mod astra-service/services/payment-service/go.sum ./astra-service/services/payment-service/

RUN --mount=type=cache,target=/go/pkg/mod \
    cd astra-service && go mod download -x

COPY astra-service/packages/go-common/ ./astra-service/packages/go-common/
COPY astra-service/services/payment-service/ ./astra-service/services/payment-service/

ARG TARGETOS
ARG TARGETARCH
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -ldflags="-s -w -extldflags=-static" \
    -o /tmp/payment-service \
    ./astra-service/services/payment-service/cmd

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /tmp/payment-service /usr/local/bin/payment-service
USER nonroot:nonroot
EXPOSE 8084
STOPSIGNAL SIGTERM
ENTRYPOINT ["/usr/local/bin/payment-service"]
