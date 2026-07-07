FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /legacy-pos-adapter ./cmd/legacy-pos-adapter

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /legacy-pos-adapter /legacy-pos-adapter

EXPOSE 8087 8088

ENTRYPOINT ["/legacy-pos-adapter"]
