package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all runtime configuration for the cart-service.
type Config struct {
	ServiceName    string
	ServiceVersion string
	Environment    string

	GRPCPort string
	HTTPPort string

	DatabaseURL     string
	RedisAddr       string
	RedisPassword   string
	RedisDB         int
	RedisSessionTTL time.Duration

	NATSURL string

	InventoryServiceAddr string

	OTELExporterEndpoint string
}

// LoadFromEnv populates a Config from environment variables, applying sensible
// defaults for local development when a variable is absent.
func LoadFromEnv() (*Config, error) {
	ttl, err := parseDuration("CART_REDIS_SESSION_TTL", 30*time.Minute)
	if err != nil {
		return nil, fmt.Errorf("config: invalid CART_REDIS_SESSION_TTL: %w", err)
	}

	redisDB, err := strconv.Atoi(envOr("CART_REDIS_DB", "0"))
	if err != nil {
		return nil, fmt.Errorf("config: invalid CART_REDIS_DB: %w", err)
	}

	return &Config{
		ServiceName:    envOr("CART_SERVICE_NAME", "astra-cart-service"),
		ServiceVersion: envOr("CART_SERVICE_VERSION", "0.1.0"),
		Environment:    envOr("ASTRA_ENV", "development"),

		GRPCPort: envOr("CART_GRPC_PORT", "50051"),
		HTTPPort: envOr("CART_HTTP_PORT", "8081"),

		DatabaseURL:     envOr("DATABASE_URL", "postgresql://astra:astra@localhost:5432/astra?sslmode=disable"),
		RedisAddr:       envOr("CART_REDIS_ADDR", "localhost:6379"),
		RedisPassword:   envOr("CART_REDIS_PASSWORD", ""),
		RedisDB:         redisDB,
		RedisSessionTTL: ttl,

		NATSURL: envOr("NATS_URL", "nats://localhost:4222"),

		InventoryServiceAddr: envOr("INVENTORY_SERVICE_ADDR", "localhost:50052"),

		OTELExporterEndpoint: envOr("OTEL_EXPORTER_OTLP_ENDPOINT", ""),
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
	return time.ParseDuration(v)
}
