package server

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/astra-service/update-server/internal/config"
	"github.com/astra-service/update-server/internal/manifest"
)

func TestServer_ManifestEndpoint(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/manifest.json", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var signed manifest.SignedManifest
	if err := json.Unmarshal(rec.Body.Bytes(), &signed); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}

	if signed.Version != "v1.2.3" {
		t.Errorf("expected version v1.2.3, got %s", signed.Version)
	}
	if signed.Signature == "" {
		t.Fatal("expected signature")
	}

	sigBytes, err := base64.StdEncoding.DecodeString(signed.Signature)
	if err != nil {
		t.Fatalf("decode signature: %v", err)
	}

	if !manifest.Verify(srv.cfg.PrivateKey.Public().(ed25519.PublicKey), signed.Manifest, sigBytes) {
		t.Fatal("manifest signature did not verify")
	}
}

func TestServer_HealthWebhook(t *testing.T) {
	srv := newTestServer(t)

	body := `{"kioskId":"K-123","version":"v1.2.3","healthy":true}`
	req := httptest.NewRequest(http.MethodPost, "/webhook/health", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected status 202, got %d: %s", rec.Code, rec.Body.String())
	}

	report, ok := srv.HealthReport("K-123")
	if !ok {
		t.Fatal("health report not recorded")
	}
	if report.KioskID != "K-123" || report.Version != "v1.2.3" || !report.Healthy {
		t.Fatalf("unexpected report: %+v", report)
	}
}

func TestServer_HealthWebhookInvalidBody(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/webhook/health", bytes.NewReader([]byte(`{"healthy":true}`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestServer_Healthz(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
}

func TestValidateWebhookPayload(t *testing.T) {
	cases := []struct {
		name    string
		body    string
		wantErr bool
	}{
		{"valid", `{"kioskId":"K-1","version":"v1","healthy":true}`, false},
		{"missing kioskId", `{"version":"v1","healthy":true}`, true},
		{"invalid json", `{`, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ValidateWebhookPayload([]byte(tc.body))
			if tc.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func newTestServer(t *testing.T) *Server {
	t.Helper()

	_, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	dir := t.TempDir()
	artifactsPath := filepath.Join(dir, "artifacts.json")
	if err := os.WriteFile(artifactsPath, []byte(testArtifactsJSON), 0644); err != nil {
		t.Fatalf("write artifacts: %v", err)
	}

	cfg := &config.Config{
		Port:          "8080",
		PrivateKey:    priv,
		Version:       "v1.2.3",
		Channel:       "stable",
		ArtifactsFile: artifactsPath,
		Rollout: config.Rollout{
			Strategy:           "idle-only",
			MaxConcurrent:      1,
			HealthCheckSeconds: 300,
		},
	}

	srv, err := New(cfg)
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	return srv
}

func TestPrivateKeyHexParsing(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	hexFull := hex.EncodeToString(priv)
	hexSeed := hex.EncodeToString(priv.Seed())

	for _, tc := range []struct {
		name string
		hex  string
	}{
		{"full private key", hexFull},
		{"seed", hexSeed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("ASTRA_UPDATE_ED25519_PRIVATE_KEY", tc.hex)
			t.Setenv("ASTRA_UPDATE_ARTIFACTS_FILE", "artifacts.json")
			cfg, err := config.Load()
			if err != nil {
				t.Fatalf("load config: %v", err)
			}
			if len(cfg.PrivateKey) != ed25519.PrivateKeySize {
				t.Fatalf("expected private key size %d, got %d", ed25519.PrivateKeySize, len(cfg.PrivateKey))
			}
		})
	}
}

const testArtifactsJSON = `{
  "kiosk-shell": {
    "url": "https://cdn.astra-service.internal/kiosk-shell/v1.2.3/astra-kiosk-shell.tar.gz",
    "checksum": "sha256:0000000000000000000000000000000000000000000000000000000000000000",
    "platforms": ["linux/amd64", "linux/arm64"]
  }
}`
