// Package config loads webauthn-service configuration from environment variables.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all runtime configuration for the webauthn-service.
type Config struct {
	Environment string

	GRPCPort string
	HTTPPort string

	DatabaseURL          string
	DatabaseMaxOpenConns int
	DatabaseMaxIdleConns int

	RPID              string
	RPOrigin          string
	RPName            string
	OverrideJWTSecret string
	OverrideTokenTTL  time.Duration

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

	ttl, err := time.ParseDuration(envOr("OVERRIDE_TOKEN_TTL", "5m"))
	if err != nil {
		return nil, fmt.Errorf("config: invalid OVERRIDE_TOKEN_TTL: %w", err)
	}

	return &Config{
		Environment:          envOr("ASTRA_ENV", "development"),
		GRPCPort:             envOr("GRPC_PORT", "8090"),
		HTTPPort:             envOr("HTTP_PORT", "8091"),
		DatabaseURL:          envOr("DATABASE_URL", "postgresql://astra:astra@localhost:5432/astra?sslmode=disable"),
		DatabaseMaxOpenConns: maxOpen,
		DatabaseMaxIdleConns: maxIdle,
		RPID:                 envOr("WEBAUTHN_RP_ID", "localhost"),
		RPOrigin:             envOr("WEBAUTHN_RP_ORIGIN", "http://localhost:5170"),
		RPName:               envOr("WEBAUTHN_RP_NAME", "Astra"),
		OverrideJWTSecret:    envOr("OVERRIDE_JWT_SECRET", "dev-override-secret-minimum-32-bytes-long"),
		OverrideTokenTTL:     ttl,
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
