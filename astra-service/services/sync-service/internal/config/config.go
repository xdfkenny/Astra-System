package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all runtime configuration for the sync-service.
type Config struct {
	GRPCPort       string
	HTTPPort       string
	Environment    string
	DatabaseURL    string
	NatsURL        string
	OTLPEndpoint   string
	RequestTimeout time.Duration
}

// Load reads configuration from environment variables and returns a validated
// Config. Every variable has a sensible default for local development.
func Load() (*Config, error) {
	grpcPort, err := strconv.Atoi(envOr("SYNC_SERVICE_GRPC_PORT", "50051"))
	if err != nil {
		return nil, fmt.Errorf("config: invalid SYNC_SERVICE_GRPC_PORT: %w", err)
	}
	httpPort, err := strconv.Atoi(envOr("SYNC_SERVICE_HTTP_PORT", "8087"))
	if err != nil {
		return nil, fmt.Errorf("config: invalid SYNC_SERVICE_HTTP_PORT: %w", err)
	}
	timeout, err := time.ParseDuration(envOr("SYNC_SERVICE_REQUEST_TIMEOUT", "30s"))
	if err != nil {
		return nil, fmt.Errorf("config: invalid SYNC_SERVICE_REQUEST_TIMEOUT: %w", err)
	}

	cfg := &Config{
		GRPCPort:       strconv.Itoa(grpcPort),
		HTTPPort:       strconv.Itoa(httpPort),
		Environment:    envOr("ASTRA_ENV", "development"),
		DatabaseURL:    envOr("DATABASE_URL", "postgresql://astra:astra@localhost:5432/astra?sslmode=disable"),
		NatsURL:        envOr("NATS_URL", "nats://localhost:4222"),
		OTLPEndpoint:   envOr("OTEL_EXPORTER_OTLP_ENDPOINT", ""),
		RequestTimeout: timeout,
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("config: DATABASE_URL is required")
	}
	if cfg.NatsURL == "" {
		return nil, fmt.Errorf("config: NATS_URL is required")
	}

	return cfg, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
