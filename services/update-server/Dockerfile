# syntax=docker/dockerfile:1

FROM golang:1.26.0-alpine3.21 AS builder

RUN apk add --no-cache ca-certificates git

WORKDIR /src

COPY services/update-server/go.mod services/update-server/go.sum ./
RUN go mod download

COPY services/update-server/ ./

RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /bin/update-server ./cmd

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /bin/update-server /usr/local/bin/update-server

USER nonroot:nonroot
EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/update-server"]
