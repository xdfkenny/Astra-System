# syntax=docker/dockerfile:1.7

ARG GO_VERSION=1.25.2
ARG ALPINE_VERSION=3.21

FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine${ALPINE_VERSION} AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build
COPY astra-service/services/webauthn-service/go.mod astra-service/services/webauthn-service/go.sum* ./astra-service/services/webauthn-service/
COPY proto/go.mod proto/go.sum* ./proto/
COPY astra-service/packages/go-common/go.mod astra-service/packages/go-common/go.sum* ./astra-service/packages/go-common/

RUN --mount=type=cache,target=/go/pkg/mod \
    cd astra-service/services/webauthn-service && go mod download -x

COPY proto/ ./proto/
COPY astra-service/packages/go-common/ ./astra-service/packages/go-common/
COPY astra-service/services/webauthn-service/ ./astra-service/services/webauthn-service/

ARG TARGETOS
ARG TARGETARCH
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    cd astra-service/services/webauthn-service && \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -ldflags="-s -w -extldflags=-static" \
    -o /tmp/webauthn-service \
    ./cmd/webauthn-service

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /tmp/webauthn-service /usr/local/bin/webauthn-service
USER nonroot:nonroot
EXPOSE 8090 8091
STOPSIGNAL SIGTERM
ENTRYPOINT ["/usr/local/bin/webauthn-service"]
