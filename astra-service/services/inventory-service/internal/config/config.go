// Package config loads service configuration from environment variables.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all runtime configuration for the inventory service.
type Config struct {
	Environment       string
	Port              string
	GRPCPort          string
	DatabaseURL       string
	RedisURL          string
	NatsURL           string
	ReservationTTL    time.Duration
	ReservationSweep  time.Duration
	CacheTTL          time.Duration
	OTELExporterURL   string
	EnableReflection  bool
}

// Load reads configuration from the environment and returns sane defaults for
// local development.
func Load() (*Config, error) {
	reservationTTL, err := parseDuration("ASTRA_RESERVATION_TTL", 5*time.Minute)
	if err != nil {
		return nil, err
	}
	sweepInterval, err := parseDuration("ASTRA_RESERVATION_SWEEP", 30*time.Second)
	if err != nil {
		return nil, err
	}
	cacheTTL, err := parseDuration("ASTRA_CACHE_TTL", 5*time.Second)
	if err != nil {
		return nil, err
	}

	return &Config{
		Environment:      envOr("ASTRA_ENV", "development"),
		Port:             envOr("PORT", "8082"),
		GRPCPort:         envOr("GRPC_PORT", "9092"),
		DatabaseURL:      envOr("DATABASE_URL", "postgresql://astra:astra@localhost:5432/astra?sslmode=disable"),
		RedisURL:         envOr("REDIS_URL", "redis://localhost:6379/0"),
		NatsURL:          envOr("NATS_URL", "nats://localhost:4222"),
		ReservationTTL:   reservationTTL,
		ReservationSweep: sweepInterval,
		CacheTTL:         cacheTTL,
		OTELExporterURL:  envOr("OTEL_EXPORTER_OTLP_ENDPOINT", ""),
		EnableReflection: envOr("GRPC_REFLECTION", "true") == "true",
	}, nil
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
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("config: invalid duration %s: %w", key, err)
	}
	return d, nil
}

// parseBool parses a boolean environment variable with the given default.
func parseBool(key string, fallback bool) (bool, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return false, fmt.Errorf("config: invalid bool %s: %w", key, err)
	}
	return b, nil
}
