package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all runtime configuration for the menu service.
type Config struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	GRPCPort       string
	HTTPPort       string
	DatabaseURL    string
	RedisURL       string
	RedisPassword  string
	RedisDB        int
	NATSURL        string
	OTLPEndpoint   string
	LogLevel       string
	CacheTTL       time.Duration
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	grpcPort, err := strconv.Atoi(envOr("MENU_SERVICE_GRPC_PORT", "50051"))
	if err != nil {
		return nil, fmt.Errorf("config: invalid MENU_SERVICE_GRPC_PORT: %w", err)
	}
	httpPort, err := strconv.Atoi(envOr("MENU_SERVICE_HTTP_PORT", "8085"))
	if err != nil {
		return nil, fmt.Errorf("config: invalid MENU_SERVICE_HTTP_PORT: %w", err)
	}
	redisDB, err := strconv.Atoi(envOr("REDIS_DB", "0"))
	if err != nil {
		return nil, fmt.Errorf("config: invalid REDIS_DB: %w", err)
	}
	cacheTTL, err := time.ParseDuration(envOr("CACHE_TTL", "5m"))
	if err != nil {
		return nil, fmt.Errorf("config: invalid CACHE_TTL: %w", err)
	}

	return &Config{
		ServiceName:    envOr("SERVICE_NAME", "astra-menu-service"),
		ServiceVersion: envOr("SERVICE_VERSION", "0.1.0"),
		Environment:    envOr("ASTRA_ENV", "development"),
		GRPCPort:       strconv.Itoa(grpcPort),
		HTTPPort:       strconv.Itoa(httpPort),
		DatabaseURL:    envOr("DATABASE_URL", "postgresql://astra:astra@localhost:5432/astra?sslmode=disable"),
		RedisURL:       envOr("REDIS_URL", "localhost:6379"),
		RedisPassword:  envOr("REDIS_PASSWORD", ""),
		RedisDB:        redisDB,
		NATSURL:        envOr("NATS_URL", "nats://localhost:4222"),
		OTLPEndpoint:   envOr("OTEL_EXPORTER_OTLP_ENDPOINT", ""),
		LogLevel:       envOr("LOG_LEVEL", "info"),
		CacheTTL:       cacheTTL,
	}, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
