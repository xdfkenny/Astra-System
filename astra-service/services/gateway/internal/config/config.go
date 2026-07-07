// Package config loads gateway runtime configuration from environment variables
// with strict validation. No configuration value is silently defaulted when it
// is required for safe operation.
package config

import (
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

// Downstream holds HTTP and gRPC addresses for one backend service.
type Downstream struct {
	HTTPBaseURL *url.URL
	GRPCAddr    string
}

// Config holds all runtime configuration for the API gateway.
type Config struct {
	Port           string
	Environment    string
	LogLevel       string
	AllowedOrigins []string
	RateLimitRPS   int
	RateLimitBurst int
	JWTIssuer      string
	JWTAudience    string
	PostgresURL    string
	RedisURL       string
	NatsURL        string
	EdDSAPublicKey ed25519.PublicKey
	RSAPublicKey   *rsa.PublicKey
	Services       map[string]Downstream
}

// Load reads configuration from the environment and validates it.
func Load() (*Config, error) {
	port, err := requireOrDefault("GATEWAY_PORT", "8080")
	if err != nil {
		return nil, err
	}
	env, err := requireOrDefault("ASTRA_ENV", "development")
	if err != nil {
		return nil, err
	}
	logLevel, err := requireOrDefault("ASTRA_LOG_LEVEL", "info")
	if err != nil {
		return nil, err
	}
	allowedOriginsStr, err := requireOrDefault("GATEWAY_ALLOWED_ORIGINS", "http://localhost:5170")
	if err != nil {
		return nil, err
	}
	rps, err := parseInt("GATEWAY_RATE_LIMIT_RPS", "50")
	if err != nil {
		return nil, err
	}
	burst, err := parseInt("GATEWAY_RATE_LIMIT_BURST", "100")
	if err != nil {
		return nil, err
	}
	issuer, err := requireOrDefault("GATEWAY_JWT_ISSUER", "astra-service")
	if err != nil {
		return nil, err
	}
	audience, err := requireOrDefault("GATEWAY_JWT_AUDIENCE", "astra-gateway")
	if err != nil {
		return nil, err
	}

	postgresURL, err := requireEnv("DATABASE_URL")
	if err != nil {
		return nil, err
	}
	redisURL, err := requireEnv("REDIS_URL")
	if err != nil {
		return nil, err
	}
	natsURL, err := requireEnv("NATS_URL")
	if err != nil {
		return nil, err
	}

	eddsaPub, rsaPub, err := loadJWTPublicKeys()
	if err != nil {
		return nil, err
	}
	if eddsaPub == nil && rsaPub == nil {
		return nil, fmt.Errorf("config: at least one of GATEWAY_JWT_EDDSA_PUBLIC_KEY/GATEWAY_JWT_EDDSA_PUBLIC_KEY_PATH or GATEWAY_JWT_RSA_PUBLIC_KEY/GATEWAY_JWT_RSA_PUBLIC_KEY_PATH must be set")
	}

	services, err := loadDownstreamServices()
	if err != nil {
		return nil, err
	}

	return &Config{
		Port:           port,
		Environment:    env,
		LogLevel:       logLevel,
		AllowedOrigins: strings.Split(allowedOriginsStr, ","),
		RateLimitRPS:   rps,
		RateLimitBurst: burst,
		JWTIssuer:      issuer,
		JWTAudience:    audience,
		PostgresURL:    postgresURL,
		RedisURL:       redisURL,
		NatsURL:        natsURL,
		EdDSAPublicKey: eddsaPub,
		RSAPublicKey:   rsaPub,
		Services:       services,
	}, nil
}

func requireEnv(key string) (string, error) {
	v := os.Getenv(key)
	if v == "" {
		return "", fmt.Errorf("config: %s is required", key)
	}
	return v, nil
}

func requireOrDefault(key, def string) (string, error) {
	v := os.Getenv(key)
	if v == "" {
		return def, nil
	}
	return v, nil
}

func parseInt(key, def string) (int, error) {
	raw := os.Getenv(key)
	if raw == "" {
		raw = def
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("config: %s must be an integer: %w", key, err)
	}
	return n, nil
}

func loadJWTPublicKeys() (ed25519.PublicKey, *rsa.PublicKey, error) {
	eddsaPub, err := loadPublicKey("GATEWAY_JWT_EDDSA_PUBLIC_KEY", "GATEWAY_JWT_EDDSA_PUBLIC_KEY_PATH", parseEdDSAPublicKey)
	if err != nil {
		return nil, nil, fmt.Errorf("config: EdDSA public key: %w", err)
	}
	rsaPub, err := loadPublicKey("GATEWAY_JWT_RSA_PUBLIC_KEY", "GATEWAY_JWT_RSA_PUBLIC_KEY_PATH", parseRSAPublicKey)
	if err != nil {
		return nil, nil, fmt.Errorf("config: RSA public key: %w", err)
	}
	return eddsaPub, rsaPub, nil
}

func loadPublicKey[T any](envKey, pathKey string, parser func([]byte) (T, error)) (T, error) {
	var zero T
	raw := os.Getenv(envKey)
	if raw == "" {
		path := os.Getenv(pathKey)
		if path == "" {
			return zero, nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return zero, fmt.Errorf("read %s: %w", pathKey, err)
		}
		raw = string(b)
	}
	decoded, err := decodeKey(raw)
	if err != nil {
		return zero, fmt.Errorf("decode %s: %w", envKey, err)
	}
	return parser(decoded)
}

func decodeKey(v string) ([]byte, error) {
	if strings.Contains(v, "BEGIN PUBLIC KEY") || strings.Contains(v, "BEGIN RSA PUBLIC KEY") || strings.Contains(v, "BEGIN ED25519 PUBLIC KEY") {
		return []byte(v), nil
	}
	decoded, err := base64.StdEncoding.DecodeString(v)
	if err != nil {
		return nil, fmt.Errorf("value is neither PEM nor valid base64: %w", err)
	}
	return decoded, nil
}

func parseEdDSAPublicKey(pemBytes []byte) (ed25519.PublicKey, error) {
	block, rest := pem.Decode(pemBytes)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}
	if len(rest) > 0 {
		return nil, fmt.Errorf("trailing data after PEM block")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	ed, ok := pub.(ed25519.PublicKey)
	if !ok {
		return nil, fmt.Errorf("expected Ed25519 public key, got %T", pub)
	}
	return ed, nil
}

func parseRSAPublicKey(pemBytes []byte) (*rsa.PublicKey, error) {
	block, rest := pem.Decode(pemBytes)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}
	if len(rest) > 0 {
		return nil, fmt.Errorf("trailing data after PEM block")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		rsaPub, pkcsErr := x509.ParsePKCS1PublicKey(block.Bytes)
		if pkcsErr != nil {
			return nil, fmt.Errorf("parse RSA public key: %w", err)
		}
		return rsaPub, nil
	}
	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("expected RSA public key, got %T", pub)
	}
	return rsaPub, nil
}

func loadDownstreamServices() (map[string]Downstream, error) {
	names := []string{"menu", "cart", "order", "inventory", "payment", "sync"}
	services := make(map[string]Downstream, len(names))
	for _, name := range names {
		httpEnv := fmt.Sprintf("%s_SERVICE_URL", strings.ToUpper(name))
		grpcEnv := fmt.Sprintf("%s_SERVICE_GRPC_ADDR", strings.ToUpper(name))
		httpURL, err := requireOrDefault(httpEnv, fmt.Sprintf("http://localhost:8081"))
		if err != nil {
			return nil, err
		}
		parsed, err := url.Parse(httpURL)
		if err != nil {
			return nil, fmt.Errorf("config: invalid %s: %w", httpEnv, err)
		}
		grpcAddr, err := requireOrDefault(grpcEnv, fmt.Sprintf("localhost:%d", 50050+len(services)+1))
		if err != nil {
			return nil, err
		}
		services[name] = Downstream{
			HTTPBaseURL: parsed,
			GRPCAddr:    grpcAddr,
		}
	}
	return services, nil
}
