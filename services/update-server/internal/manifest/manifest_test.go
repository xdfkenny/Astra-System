package manifest

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"testing"
)

func TestSignAndVerify(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	gen := NewGenerator(priv, "v1.2.3", "stable", map[string]Artifact{
		"kiosk-shell": {
			URL:       "https://cdn.astra-service.internal/kiosk-shell.tar.gz",
			Checksum:  "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			Platforms: []string{"linux/arm64", "linux/amd64"},
		},
	}, Rollout{
		Strategy:           "idle-only",
		MaxConcurrent:      1,
		HealthCheckSeconds: 300,
	})

	signed, err := gen.Generate()
	if err != nil {
		t.Fatalf("generate manifest: %v", err)
	}

	if signed.Version != "v1.2.3" {
		t.Errorf("expected version v1.2.3, got %s", signed.Version)
	}
	if signed.Channel != "stable" {
		t.Errorf("expected channel stable, got %s", signed.Channel)
	}
	if signed.Signature == "" {
		t.Fatal("expected non-empty signature")
	}

	sigBytes, err := base64.StdEncoding.DecodeString(signed.Signature)
	if err != nil {
		t.Fatalf("decode signature: %v", err)
	}

	if !Verify(priv.Public().(ed25519.PublicKey), signed.Manifest, sigBytes) {
		t.Fatal("signature verification failed")
	}
}

func TestCanonicalizationIsDeterministic(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	artifacts := map[string]Artifact{
		"sync-daemon": {
			URL:       "https://cdn.astra-service.internal/sync-daemon.tar.gz",
			Checksum:  "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			Platforms: []string{"linux/amd64"},
		},
		"kiosk-shell": {
			URL:       "https://cdn.astra-service.internal/kiosk-shell.tar.gz",
			Checksum:  "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			Platforms: []string{"linux/arm64", "linux/amd64"},
		},
	}

	gen1 := NewGenerator(priv, "v1.0.0", "stable", artifacts, Rollout{Strategy: "idle-only", MaxConcurrent: 1, HealthCheckSeconds: 300})
	gen2 := NewGenerator(priv, "v1.0.0", "stable", map[string]Artifact{
		"kiosk-shell": artifacts["kiosk-shell"],
		"sync-daemon": artifacts["sync-daemon"],
	}, Rollout{Strategy: "idle-only", MaxConcurrent: 1, HealthCheckSeconds: 300})

	s1, err := gen1.Generate()
	if err != nil {
		t.Fatalf("generate first manifest: %v", err)
	}
	s2, err := gen2.Generate()
	if err != nil {
		t.Fatalf("generate second manifest: %v", err)
	}

	b1, _ := json.Marshal(s1.Manifest)
	b2, _ := json.Marshal(s2.Manifest)
	if string(b1) != string(b2) {
		t.Fatalf("manifests differ with reordered artifacts:\n%s\n%s", b1, b2)
	}

	if s1.Signature != s2.Signature {
		t.Fatal("signatures differ with reordered artifacts")
	}
}

func TestVerifyDetectsTampering(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	gen := NewGenerator(priv, "v1.0.0", "stable", map[string]Artifact{
		"kiosk-shell": {
			URL:      "https://cdn.astra-service.internal/kiosk-shell.tar.gz",
			Checksum: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		},
	}, Rollout{Strategy: "idle-only", MaxConcurrent: 1, HealthCheckSeconds: 300})

	signed, err := gen.Generate()
	if err != nil {
		t.Fatalf("generate manifest: %v", err)
	}

	sigBytes, _ := base64.StdEncoding.DecodeString(signed.Signature)
	tampered := signed.Manifest
	tampered.Version = "v1.0.1"

	if Verify(priv.Public().(ed25519.PublicKey), tampered, sigBytes) {
		t.Fatal("verification should fail for tampered manifest")
	}
}
