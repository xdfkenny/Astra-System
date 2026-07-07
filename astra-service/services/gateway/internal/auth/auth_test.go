package auth

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"
	"time"

	"github.com/astra-systems/astra-service/services/gateway/internal/config"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

func TestParse_EdDSA(t *testing.T) {
	cfg, signer := testConfigEdDSA(t)
	token := signEdDSA(t, signer, cfg.JWTIssuer, cfg.JWTAudience, "kiosk-1")

	claims, err := Parse(token, cfg)
	require.NoError(t, err)
	require.Equal(t, "kiosk-1", claims["sub"])
}

func TestParse_RS256(t *testing.T) {
	cfg, signer := testConfigRSA(t)
	token := signRSA(t, signer, cfg.JWTIssuer, cfg.JWTAudience, "kiosk-1")

	claims, err := Parse(token, cfg)
	require.NoError(t, err)
	require.Equal(t, "kiosk-1", claims["sub"])
}

func TestParse_WrongAlgorithm(t *testing.T) {
	cfg, _ := testConfigEdDSA(t)
	// Sign with HMAC even though no key is configured.
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iss": cfg.JWTIssuer,
		"aud": cfg.JWTAudience,
		"sub": "kiosk-1",
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	signed, err := token.SignedString([]byte("some-secret"))
	require.NoError(t, err)

	_, err = Parse(signed, cfg)
	require.Error(t, err)
}

func TestParse_ExpiredToken(t *testing.T) {
	cfg, signer := testConfigEdDSA(t)
	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, jwt.MapClaims{
		"iss": cfg.JWTIssuer,
		"aud": cfg.JWTAudience,
		"sub": "kiosk-1",
		"iat": time.Now().Add(-2 * time.Hour).Unix(),
		"exp": time.Now().Add(-1 * time.Hour).Unix(),
	})
	signed, err := token.SignedString(signer)
	require.NoError(t, err)

	_, err = Parse(signed, cfg)
	require.Error(t, err)
}

func TestParse_MissingSubject(t *testing.T) {
	cfg, signer := testConfigEdDSA(t)
	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, jwt.MapClaims{
		"iss": cfg.JWTIssuer,
		"aud": cfg.JWTAudience,
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	signed, err := token.SignedString(signer)
	require.NoError(t, err)

	claims, err := Parse(signed, cfg)
	require.NoError(t, err)
	require.Empty(t, claims["sub"])
}

func testConfigEdDSA(t *testing.T) (*config.Config, ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	return &config.Config{
		JWTIssuer:      "astra-service",
		JWTAudience:    "astra-gateway",
		EdDSAPublicKey: pub,
	}, priv
}

func testConfigRSA(t *testing.T) (*config.Config, *rsa.PrivateKey) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	return &config.Config{
		JWTIssuer:    "astra-service",
		JWTAudience:  "astra-gateway",
		RSAPublicKey: &priv.PublicKey,
	}, priv
}

func signEdDSA(t *testing.T, signer ed25519.PrivateKey, issuer, audience, subject string) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, jwt.MapClaims{
		"iss": issuer,
		"aud": audience,
		"sub": subject,
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	signed, err := token.SignedString(signer)
	require.NoError(t, err)
	return signed
}

func signRSA(t *testing.T, signer *rsa.PrivateKey, issuer, audience, subject string) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"iss": issuer,
		"aud": audience,
		"sub": subject,
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	signed, err := token.SignedString(signer)
	require.NoError(t, err)
	return signed
}

func marshalPublicKeyPEM(t *testing.T, pub interface{}) []byte {
	t.Helper()
	b, err := x509.MarshalPKIXPublicKey(pub)
	require.NoError(t, err)
	return pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: b})
}
