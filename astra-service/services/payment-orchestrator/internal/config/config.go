// Package config loads payment-orchestrator configuration from environment
// variables with sensible defaults for local development.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all runtime settings for the service.
type Config struct {
	Port               string
	GRPCPort           string
	Environment        string
	DatabaseURL        string
	RedisURL           string
	NatsURL            string
	VerifoneHTTPURL    string
	VerifoneGRPCAddr   string
	WebhookSecret      []byte
	OfflineTokenSecret []byte
	SettlementEnabled  bool
	SettlementInterval time.Duration
	OTLPEndpoint       string
}

// Load reads configuration from the environment.
func Load() (*Config, error) {
	port, err := strconv.Atoi(envOr("PAYMENT_ORCHESTRATOR_PORT", "8086"))
	if err != nil {
		return nil, fmt.Errorf("config: invalid PAYMENT_ORCHESTRATOR_PORT: %w", err)
	}
	grpcPort, err := strconv.Atoi(envOr("PAYMENT_ORCHESTRATOR_GRPC_PORT", "50086"))
	if err != nil {
		return nil, fmt.Errorf("config: invalid PAYMENT_ORCHESTRATOR_GRPC_PORT: %w", err)
	}

	return &Config{
		Port:               strconv.Itoa(port),
		GRPCPort:           strconv.Itoa(grpcPort),
		Environment:        envOr("ASTRA_ENV", "development"),
		DatabaseURL:        envOr("DATABASE_URL", "postgresql://astra:astra@localhost:5432/astra?sslmode=disable"),
		RedisURL:           envOr("REDIS_URL", "redis://localhost:6379/0"),
		NatsURL:            envOr("NATS_URL", "nats://localhost:4222"),
		VerifoneHTTPURL:    envOr("VERIFONE_HTTP_URL", "http://localhost:8963"),
		VerifoneGRPCAddr:   envOr("VERIFONE_GRPC_ADDR", ""),
		WebhookSecret:      []byte(envOr("PAYMENT_WEBHOOK_SECRET", "astra-dev-webhook-secret-change-in-production")),
		OfflineTokenSecret: []byte(envOr("OFFLINE_TOKEN_SECRET", "astra-dev-offline-token-secret-change-in-production")),
		SettlementEnabled:  envOrBool("PAYMENT_SETTLEMENT_ENABLED", false),
		SettlementInterval: envOrDuration("PAYMENT_SETTLEMENT_INTERVAL", 30*time.Second),
		OTLPEndpoint:       envOr("OTEL_EXPORTER_OTLP_ENDPOINT", ""),
	}, nil
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
