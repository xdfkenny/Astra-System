// Package config centralizes environment-driven configuration for the API
// gateway. All values have safe production defaults EXCEPT secrets, which
// intentionally have no default and fail fast at boot if missing — a
// gateway must never silently start with a blank signing key.
package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Port           string
	NatsURL        string
	RedisURL       string
	HMACSigningKey []byte
	RateLimitRPS   int
	RateLimitBurst int
	JWTIssuer      string
	OTLPEndpoint   string
	Environment    string
}

func Load() (*Config, error) {
	signingKey := os.Getenv("GATEWAY_HMAC_SIGNING_KEY")
	if len(signingKey) < 32 {
		return nil, fmt.Errorf("config: GATEWAY_HMAC_SIGNING_KEY must be set and >=32 bytes")
	}

	rps, err := strconv.Atoi(envOr("GATEWAY_RATE_LIMIT_RPS", "50"))
	if err != nil {
		return nil, fmt.Errorf("config: invalid GATEWAY_RATE_LIMIT_RPS: %w", err)
	}
	burst, err := strconv.Atoi(envOr("GATEWAY_RATE_LIMIT_BURST", "100"))
	if err != nil {
		return nil, fmt.Errorf("config: invalid GATEWAY_RATE_LIMIT_BURST: %w", err)
	}

	return &Config{
		Port:           envOr("GATEWAY_PORT", "8080"),
		NatsURL:        envOr("NATS_URL", "nats://localhost:4222"),
		RedisURL:       envOr("REDIS_URL", "redis://localhost:6379/0"),
		HMACSigningKey: []byte(signingKey),
		RateLimitRPS:   rps,
		RateLimitBurst: burst,
		JWTIssuer:      envOr("GATEWAY_JWT_ISSUER", "astra-service"),
		OTLPEndpoint:   envOr("OTEL_EXPORTER_OTLP_ENDPOINT", ""),
		Environment:    envOr("ASTRA_ENV", "development"),
	}, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
