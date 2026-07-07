package config

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoad_DefaultValues(t *testing.T) {
	clearEnv(t)
	setRequired(t)

	cfg, err := Load()
	require.NoError(t, err)
	require.Equal(t, "8080", cfg.Port)
	require.Equal(t, "development", cfg.Environment)
	require.Equal(t, []string{"http://localhost:5170"}, cfg.AllowedOrigins)
	require.Equal(t, 50, cfg.RateLimitRPS)
	require.Equal(t, 100, cfg.RateLimitBurst)
}

func TestLoad_MissingPostgresURL(t *testing.T) {
	clearEnv(t)
	setRequired(t)
	require.NoError(t, os.Unsetenv("DATABASE_URL"))

	_, err := Load()
	require.ErrorContains(t, err, "DATABASE_URL")
}

func TestLoad_InvalidRateLimit(t *testing.T) {
	clearEnv(t)
	setRequired(t)
	t.Setenv("GATEWAY_RATE_LIMIT_RPS", "not-a-number")

	_, err := Load()
	require.ErrorContains(t, err, "GATEWAY_RATE_LIMIT_RPS")
}

func TestLoad_EdDSAKeyFromEnv(t *testing.T) {
	clearEnv(t)
	setRequired(t)
	pub, _ := generateEdDSAKey(t)
	t.Setenv("GATEWAY_JWT_EDDSA_PUBLIC_KEY", base64.StdEncoding.EncodeToString(pubPEM(pub)))

	cfg, err := Load()
	require.NoError(t, err)
	require.NotNil(t, cfg.EdDSAPublicKey)
}

func TestLoad_EdDSAKeyFromFile(t *testing.T) {
	clearEnv(t)
	setRequired(t)
	pub, _ := generateEdDSAKey(t)
	path := filepath.Join(t.TempDir(), "eddsa.pub")
	require.NoError(t, os.WriteFile(path, pubPEM(pub), 0600))
	t.Setenv("GATEWAY_JWT_EDDSA_PUBLIC_KEY_PATH", path)

	cfg, err := Load()
	require.NoError(t, err)
	require.NotNil(t, cfg.EdDSAPublicKey)
}

func TestLoad_RSAKeyFromEnv(t *testing.T) {
	clearEnv(t)
	setRequired(t)
	pub := generateRSAKey(t)
	t.Setenv("GATEWAY_JWT_RSA_PUBLIC_KEY", base64.StdEncoding.EncodeToString(rsaPubPEM(pub)))

	cfg, err := Load()
	require.NoError(t, err)
	require.NotNil(t, cfg.RSAPublicKey)
}

func TestLoad_NoKeyFails(t *testing.T) {
	clearEnv(t)
	setRequired(t)
	require.NoError(t, os.Unsetenv("GATEWAY_JWT_EDDSA_PUBLIC_KEY"))
	require.NoError(t, os.Unsetenv("GATEWAY_JWT_EDDSA_PUBLIC_KEY_PATH"))
	require.NoError(t, os.Unsetenv("GATEWAY_JWT_RSA_PUBLIC_KEY"))
	require.NoError(t, os.Unsetenv("GATEWAY_JWT_RSA_PUBLIC_KEY_PATH"))

	_, err := Load()
	require.ErrorContains(t, err, "at least one")
}

func setRequired(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_URL", "postgresql://astra:astra@localhost:5432/astra?sslmode=disable")
	t.Setenv("REDIS_URL", "redis://localhost:6379/0")
	t.Setenv("NATS_URL", "nats://localhost:4222")
	pub, _ := generateEdDSAKey(t)
	t.Setenv("GATEWAY_JWT_EDDSA_PUBLIC_KEY", base64.StdEncoding.EncodeToString(pubPEM(pub)))
}

func clearEnv(t *testing.T) {
	t.Helper()
	keys := []string{
		"GATEWAY_PORT", "ASTRA_ENV", "ASTRA_LOG_LEVEL", "GATEWAY_ALLOWED_ORIGINS",
		"GATEWAY_RATE_LIMIT_RPS", "GATEWAY_RATE_LIMIT_BURST", "GATEWAY_JWT_ISSUER",
		"GATEWAY_JWT_AUDIENCE", "DATABASE_URL", "REDIS_URL", "NATS_URL",
		"GATEWAY_JWT_EDDSA_PUBLIC_KEY", "GATEWAY_JWT_EDDSA_PUBLIC_KEY_PATH",
		"GATEWAY_JWT_RSA_PUBLIC_KEY", "GATEWAY_JWT_RSA_PUBLIC_KEY_PATH",
	}
	for _, k := range keys {
		require.NoError(t, os.Unsetenv(k))
	}
}

func generateEdDSAKey(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	return pub, priv
}

func generateRSAKey(t *testing.T) *rsa.PublicKey {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	return &priv.PublicKey
}

func pubPEM(pub ed25519.PublicKey) []byte {
	b, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		panic(err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: b})
}

func rsaPubPEM(pub *rsa.PublicKey) []byte {
	b, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		panic(err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: b})
}
