// Package manifest builds a cryptographically signed update manifest.
package manifest

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"
)

// Artifact describes a single downloadable OTA artifact.
type Artifact struct {
	URL       string   `json:"url"`
	Checksum  string   `json:"checksum"`
	Platforms []string `json:"platforms"`
}

// Rollout controls how updates are distributed to kiosks.
type Rollout struct {
	Strategy           string `json:"strategy"`
	MaxConcurrent      int    `json:"maxConcurrent"`
	HealthCheckSeconds int    `json:"healthCheckSeconds"`
}

// Manifest is the unsigned update payload.
type Manifest struct {
	Version    string              `json:"version"`
	Channel    string              `json:"channel"`
	ReleasedAt string              `json:"releasedAt"`
	Artifacts  map[string]Artifact `json:"artifacts"`
	Rollout    Rollout             `json:"rollout"`
}

// SignedManifest is a manifest plus a detached Ed25519 signature.
type SignedManifest struct {
	Manifest
	Signature string `json:"signature"`
}

// Generator produces signed manifests from a static artifact catalog.
type Generator struct {
	privateKey ed25519.PrivateKey
	artifacts  map[string]Artifact
	version    string
	channel    string
	rollout    Rollout
}

// NewGenerator creates a generator with the provided signing key and catalog.
func NewGenerator(privateKey ed25519.PrivateKey, version, channel string, artifacts map[string]Artifact, rollout Rollout) *Generator {
	return &Generator{
		privateKey: privateKey,
		artifacts:  artifacts,
		version:    version,
		channel:    channel,
		rollout:    rollout,
	}
}

// LoadArtifacts reads an artifact catalog from a JSON file.
func LoadArtifacts(path string) (map[string]Artifact, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read artifacts file: %w", err)
	}
	var artifacts map[string]Artifact
	if err := json.Unmarshal(data, &artifacts); err != nil {
		return nil, fmt.Errorf("parse artifacts file: %w", err)
	}
	return artifacts, nil
}

// Generate creates a fresh signed manifest for the current time.
func (g *Generator) Generate() (*SignedManifest, error) {
	m := Manifest{
		Version:    g.version,
		Channel:    g.channel,
		ReleasedAt: time.Now().UTC().Round(time.Second).Format(time.RFC3339),
		Artifacts:  g.artifacts,
		Rollout:    g.rollout,
	}

	sigBytes, err := Sign(g.privateKey, m)
	if err != nil {
		return nil, fmt.Errorf("sign manifest: %w", err)
	}

	return &SignedManifest{
		Manifest:  m,
		Signature: base64.StdEncoding.EncodeToString(sigBytes),
	}, nil
}

// Sign produces a detached Ed25519 signature for a manifest.
func Sign(privateKey ed25519.PrivateKey, m Manifest) ([]byte, error) {
	canonical, err := canonicalBytes(m)
	if err != nil {
		return nil, err
	}
	return ed25519.Sign(privateKey, canonical), nil
}

// Verify checks a detached Ed25519 signature for a manifest.
func Verify(publicKey ed25519.PublicKey, m Manifest, signature []byte) bool {
	canonical, err := canonicalBytes(m)
	if err != nil {
		return false
	}
	return ed25519.Verify(publicKey, canonical, signature)
}

// canonicalBytes returns a deterministic, stable serialization of a manifest.
// Artifacts are sorted by name so reordering the source file does not change
// the signature.
func canonicalBytes(m Manifest) ([]byte, error) {
	names := make([]string, 0, len(m.Artifacts))
	for name := range m.Artifacts {
		names = append(names, name)
	}
	sort.Strings(names)

	sortedArtifacts := make(map[string]Artifact, len(m.Artifacts))
	for _, name := range names {
		art := m.Artifacts[name]
		platforms := make([]string, len(art.Platforms))
		copy(platforms, art.Platforms)
		sort.Strings(platforms)
		sortedArtifacts[name] = Artifact{
			URL:       art.URL,
			Checksum:  art.Checksum,
			Platforms: platforms,
		}
	}

	canonical := struct {
		Version    string              `json:"version"`
		Channel    string              `json:"channel"`
		ReleasedAt string              `json:"releasedAt"`
		Artifacts  map[string]Artifact `json:"artifacts"`
		Rollout    Rollout             `json:"rollout"`
	}{
		Version:    m.Version,
		Channel:    m.Channel,
		ReleasedAt: m.ReleasedAt,
		Artifacts:  sortedArtifacts,
		Rollout:    m.Rollout,
	}

	return json.Marshal(canonical)
}
