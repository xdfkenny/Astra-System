// Command jwtgen generates EdDSA key pairs and JWTs for Astra-Service kiosk-to-gateway auth.
//
// Usage:
//
//	# Generate a new Ed25519 key pair
//	go run . -generate -out ./keys
//
//	# Sign a JWT for a kiosk
//	go run . -sign -key ./keys/kiosk-eddsa-private.pem -sub kiosk-dev-001
//
//	# Generate keys AND sign in one step
//	go run . -generate -sign -sub kiosk-dev-001
package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	defaultIss      = "astra-service"
	defaultAud      = "astra-gateway"
	defaultSub      = "kiosk-dev-001"
	defaultTTLHours = 87600 // 10 years
)

func main() {
	var (
		generate  = flag.Bool("generate", false, "Generate Ed25519 key pair")
		sign      = flag.Bool("sign", false, "Sign a JWT using existing private key")
		outDir    = flag.String("out", ".", "Output directory for generated keys")
		keyPath   = flag.String("key", "kiosk-eddsa-private.pem", "Path to Ed25519 private key PEM file")
		sub       = flag.String("sub", defaultSub, "Subject claim (kiosk ID)")
		iss       = flag.String("iss", defaultIss, "Issuer claim")
		aud       = flag.String("aud", defaultAud, "Audience claim")
		ttlHours  = flag.Int("ttl", defaultTTLHours, "Token TTL in hours")
	)
	flag.Parse()

	if *generate {
		if err := os.MkdirAll(*outDir, 0700); err != nil {
			fmt.Fprintf(os.Stderr, "create output dir: %v\n", err)
			os.Exit(1)
		}
		pub, priv, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			fmt.Fprintf(os.Stderr, "generate key: %v\n", err)
			os.Exit(1)
		}
		if err := writePrivateKey(priv, filepath.Join(*outDir, "kiosk-eddsa-private.pem")); err != nil {
			fmt.Fprintf(os.Stderr, "write private key: %v\n", err)
			os.Exit(1)
		}
		if err := writePublicKey(pub, filepath.Join(*outDir, "kiosk-eddsa-public.pem")); err != nil {
			fmt.Fprintf(os.Stderr, "write public key: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Generated Ed25519 key pair in %s\n", *outDir)
		fmt.Fprintf(os.Stderr, "  Private: %s\n", filepath.Join(*outDir, "kiosk-eddsa-private.pem"))
		fmt.Fprintf(os.Stderr, "  Public:  %s\n", filepath.Join(*outDir, "kiosk-eddsa-public.pem"))
	}

	if *sign {
		tokenStr, err := signToken(*keyPath, *sub, *iss, *aud, *ttlHours)
		if err != nil {
			fmt.Fprintf(os.Stderr, "sign token: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(tokenStr)
	}
}

func signToken(privKeyPath, sub, iss, aud string, ttlHours int) (string, error) {
	privBytes, err := os.ReadFile(privKeyPath)
	if err != nil {
		return "", fmt.Errorf("read private key: %w", err)
	}
	block, _ := pem.Decode(privBytes)
	if block == nil {
		return "", fmt.Errorf("decode PEM block from %s", privKeyPath)
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("parse private key: %w", err)
	}
	priv, ok := parsed.(ed25519.PrivateKey)
	if !ok {
		return "", fmt.Errorf("expected Ed25519 private key, got %T", parsed)
	}

	now := time.Now()
	claims := jwt.MapClaims{
		"iss": iss,
		"aud": aud,
		"sub": sub,
		"iat": now.Unix(),
		"exp": now.Add(time.Duration(ttlHours) * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	tokenStr, err := token.SignedString(priv)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return tokenStr, nil
}

func writePrivateKey(key ed25519.PrivateKey, path string) error {
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return err
	}
	return writePEM(path, "PRIVATE KEY", der)
}

func writePublicKey(key ed25519.PublicKey, path string) error {
	der, err := x509.MarshalPKIXPublicKey(key)
	if err != nil {
		return err
	}
	return writePEM(path, "PUBLIC KEY", der)
}

func writePEM(path, blockType string, der []byte) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := pem.Encode(f, &pem.Block{Type: blockType, Bytes: der}); err != nil {
		return err
	}
	return f.Close()
}
