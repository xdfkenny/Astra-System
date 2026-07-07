// Package config loads payment-service runtime configuration from
// environment variables. All secrets (DB passwords, HMAC keys) are expected to
// be injected by the runtime environment (Vault / SOPS / age) — never baked
// into the container image.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds the service's runtime configuration.
type Config struct {
	Environment        string
	Port               string
	DatabaseURL        string
	NatsURL            string
	OfflineTokenHMAC   []byte
	SettlementEnabled  bool
	SettlementInterval time.Duration
	OTLPEndpoint       string
}

// Load reads configuration from the environment, applying sane defaults for
// local development. It returns an error if a required value is missing in a
// non-development environment.
func Load() (*Config, error) {
	cfg := &Config{
		Environment:        envOr("ASTRA_ENV", "development"),
		Port:               envOr("ASTRA_PORT", "8084"),
		DatabaseURL:        envOr("DATABASE_URL", "postgresql://astra:astra@localhost:5432/astra?sslmode=disable"),
		NatsURL:            envOr("NATS_URL", "nats://localhost:4222"),
		OfflineTokenHMAC:   []byte(envOr("ASTRA_OFFLINE_HMAC_KEY", "astra-dev-offline-hmac-key-change-in-production")),
		SettlementEnabled:  envOrBool("ASTRA_SETTLEMENT_ENABLED", true),
		SettlementInterval: envOrDuration("ASTRA_SETTLEMENT_INTERVAL", 30*time.Second),
		OTLPEndpoint:       envOr("OTEL_EXPORTER_OTLP_ENDPOINT", ""),
	}

	if cfg.Environment != "development" {
		if len(cfg.OfflineTokenHMAC) < 32 {
			return nil, fmt.Errorf("ASTRA_OFFLINE_HMAC_KEY must be at least 32 bytes in %s", cfg.Environment)
		}
	}

	return cfg, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envOrBool(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return fallback
		}
		return b
	}
	return fallback
}

func envOrDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return fallback
		}
		return d
	}
	return fallback
}
