# syntax=docker/dockerfile:1.7

# Astra-Service Cart Service
ARG GO_VERSION=1.26.0
ARG ALPINE_VERSION=3.21

FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine${ALPINE_VERSION} AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build
COPY astra-service/go.work astra-service/go.work.sum* ./astra-service/
COPY astra-service/packages/go-common/go.mod astra-service/packages/go-common/go.sum ./astra-service/packages/go-common/
COPY astra-service/services/cart-service/go.mod astra-service/services/cart-service/go.sum ./astra-service/services/cart-service/
COPY astra-service/services/order-service/go.mod astra-service/services/order-service/go.sum* ./astra-service/services/order-service/
COPY astra-service/services/inventory-service/go.mod astra-service/services/inventory-service/go.sum* ./astra-service/services/inventory-service/
COPY astra-service/services/payment-service/go.mod astra-service/services/payment-service/go.sum* ./astra-service/services/payment-service/
COPY astra-service/services/api-gateway/go.mod astra-service/services/api-gateway/go.sum* ./astra-service/services/api-gateway/

RUN --mount=type=cache,target=/go/pkg/mod \
    cd astra-service && go mod download -x

COPY astra-service/packages/go-common/ ./astra-service/packages/go-common/
COPY astra-service/services/cart-service/ ./astra-service/services/cart-service/
COPY proto/ ./proto/

ARG TARGETOS
ARG TARGETARCH
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -ldflags="-s -w -extldflags=-static" \
    -o /tmp/cart-service \
    ./astra-service/services/cart-service/cmd/cart-service

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /tmp/cart-service /usr/local/bin/cart-service
USER nonroot:nonroot
EXPOSE 50051 8081
STOPSIGNAL SIGTERM
ENTRYPOINT ["/usr/local/bin/cart-service"]
