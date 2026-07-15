// Package config loads order-service configuration from environment variables.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all runtime configuration for the order-service.
type Config struct {
	Environment string

	GRPCPort string
	HTTPPort string

	DatabaseURL          string
	DatabaseMaxOpenConns int
	DatabaseMaxIdleConns int

	NatsURL string

	CartServiceTarget string

	OTLPEndpoint string
}

// Load reads configuration from environment variables and applies sensible
// defaults for local development.
func Load() (*Config, error) {
	maxOpen, err := strconv.Atoi(envOr("DATABASE_MAX_OPEN_CONNS", "20"))
	if err != nil {
		return nil, fmt.Errorf("config: invalid DATABASE_MAX_OPEN_CONNS: %w", err)
	}
	maxIdle, err := strconv.Atoi(envOr("DATABASE_MAX_IDLE_CONNS", "5"))
	if err != nil {
		return nil, fmt.Errorf("config: invalid DATABASE_MAX_IDLE_CONNS: %w", err)
	}

	return &Config{
		Environment:          envOr("ASTRA_ENV", "development"),
		GRPCPort:             envOr("GRPC_PORT", "8083"),
		HTTPPort:             envOr("HTTP_PORT", "8084"),
		DatabaseURL:          envOr("DATABASE_URL", "postgresql://astra:astra@localhost:5432/astra?sslmode=disable"),
		DatabaseMaxOpenConns: maxOpen,
		DatabaseMaxIdleConns: maxIdle,
		NatsURL:              envOr("NATS_URL", "nats://localhost:4222"),
		CartServiceTarget:    envOr("CART_SERVICE_TARGET", "localhost:8082"),
		OTLPEndpoint:         envOr("OTEL_EXPORTER_OTLP_ENDPOINT", ""),
	}, nil
}

// ShutdownTimeout is the grace period for closing listeners and draining
// in-flight requests.
func (c *Config) ShutdownTimeout() time.Duration {
	return 15 * time.Second
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
