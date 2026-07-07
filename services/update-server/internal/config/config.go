// Package config loads update-server settings from the environment.
package config

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Rollout describes how an update should be distributed to kiosks.
type Rollout struct {
	Strategy           string
	MaxConcurrent      int
	HealthCheckSeconds int
}

// Config holds runtime configuration for the update server.
type Config struct {
	Port          string
	PrivateKey    ed25519.PrivateKey
	Version       string
	Channel       string
	ArtifactsFile string
	Rollout       Rollout
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	pkHex := strings.TrimSpace(os.Getenv("ASTRA_UPDATE_ED25519_PRIVATE_KEY"))
	if pkHex == "" {
		return nil, fmt.Errorf("ASTRA_UPDATE_ED25519_PRIVATE_KEY is required")
	}

	pkBytes, err := hex.DecodeString(pkHex)
	if err != nil {
		return nil, fmt.Errorf("ASTRA_UPDATE_ED25519_PRIVATE_KEY must be hex encoded: %w", err)
	}

	var privateKey ed25519.PrivateKey
	switch len(pkBytes) {
	case ed25519.SeedSize:
		privateKey = ed25519.NewKeyFromSeed(pkBytes)
	case ed25519.PrivateKeySize:
		privateKey = ed25519.PrivateKey(pkBytes)
	default:
		return nil, fmt.Errorf("ASTRA_UPDATE_ED25519_PRIVATE_KEY must be %d or %d bytes, got %d",
			ed25519.SeedSize, ed25519.PrivateKeySize, len(pkBytes))
	}

	port := os.Getenv("ASTRA_UPDATE_PORT")
	if port == "" {
		port = "8080"
	}

	version := os.Getenv("ASTRA_UPDATE_VERSION")
	if version == "" {
		version = "dev"
	}

	channel := os.Getenv("ASTRA_UPDATE_CHANNEL")
	if channel == "" {
		channel = "stable"
	}

	artifactsFile := os.Getenv("ASTRA_UPDATE_ARTIFACTS_FILE")
	if artifactsFile == "" {
		artifactsFile = "artifacts.json"
	}

	rollout := Rollout{
		Strategy:           "idle-only",
		MaxConcurrent:      1,
		HealthCheckSeconds: 300,
	}
	if v := os.Getenv("ASTRA_UPDATE_ROLLOUT_STRATEGY"); v != "" {
		rollout.Strategy = v
	}
	if v := os.Getenv("ASTRA_UPDATE_ROLLOUT_MAX_CONCURRENT"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("ASTRA_UPDATE_ROLLOUT_MAX_CONCURRENT: %w", err)
		}
		rollout.MaxConcurrent = n
	}
	if v := os.Getenv("ASTRA_UPDATE_HEALTH_CHECK_SECONDS"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("ASTRA_UPDATE_HEALTH_CHECK_SECONDS: %w", err)
		}
		rollout.HealthCheckSeconds = n
	}

	return &Config{
		Port:          port,
		PrivateKey:    privateKey,
		Version:       version,
		Channel:       channel,
		ArtifactsFile: artifactsFile,
		Rollout:       rollout,
	}, nil
}
