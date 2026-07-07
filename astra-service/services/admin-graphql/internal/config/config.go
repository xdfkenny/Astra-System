// Package config loads admin-graphql service configuration from environment variables.
package config

import (
	"os"
	"time"
)

// Config holds all runtime configuration for the admin-graphql service.
type Config struct {
	Environment string
	ServiceName string
	HTTPPort    string
	DatabaseURL string
	JWTSecret   []byte
	JWTIssuer   string
	JWTAudience string
	OTLPEndpoint string
}

// Load reads configuration from environment variables and applies sensible
// defaults for local development.
func Load() (*Config, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "dev-admin-graphql-secret-minimum-32-bytes-long"
	}
	return &Config{
		Environment:  envOr("ASTRA_ENV", "development"),
		ServiceName:  envOr("SERVICE_NAME", "astra-admin-graphql"),
		HTTPPort:     envOr("HTTP_PORT", "8092"),
		DatabaseURL:  envOr("DATABASE_URL", "postgresql://astra:astra@localhost:5432/astra?sslmode=disable"),
		JWTSecret:    []byte(secret),
		JWTIssuer:    envOr("JWT_ISSUER", "astra-service"),
		JWTAudience:  envOr("JWT_AUDIENCE", "astra-admin"),
		OTLPEndpoint: envOr("OTEL_EXPORTER_OTLP_ENDPOINT", ""),
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
