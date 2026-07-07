// Package config loads legacy-pos-adapter configuration from environment
// variables.
package config

import (
	"fmt"
	"os"
	"time"
)

// Config holds all runtime configuration for the legacy-pos-adapter.
type Config struct {
	Environment string

	GRPCPort string
	HTTPPort string

	NatsURL string

	LegacyPOSURL     string
	LegacyPOSTimeout time.Duration
	LegacyPOSAPIKey  string

	OTLPEndpoint string
}

// Load reads configuration from environment variables and applies sensible
// defaults for local development.
func Load() (*Config, error) {
	timeout, err := parseDuration("LEGACY_POS_TIMEOUT", 10*time.Second)
	if err != nil {
		return nil, fmt.Errorf("config: invalid LEGACY_POS_TIMEOUT: %w", err)
	}

	return &Config{
		Environment:      envOr("ASTRA_ENV", "development"),
		GRPCPort:         envOr("GRPC_PORT", "8087"),
		HTTPPort:         envOr("HTTP_PORT", "8088"),
		NatsURL:          envOr("NATS_URL", "nats://localhost:4222"),
		LegacyPOSURL:     envOr("LEGACY_POS_URL", ""),
		LegacyPOSTimeout: timeout,
		LegacyPOSAPIKey:  envOr("LEGACY_POS_API_KEY", ""),
		OTLPEndpoint:     envOr("OTEL_EXPORTER_OTLP_ENDPOINT", ""),
	}, nil
}

// Enabled returns true when the legacy POS integration is configured.
func (c *Config) Enabled() bool {
	return c.LegacyPOSURL != ""
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

func parseDuration(key string, fallback time.Duration) (time.Duration, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	return time.ParseDuration(v)
}
