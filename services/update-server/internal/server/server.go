// Package server exposes the update-server HTTP API.
package server

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/astra-service/update-server/internal/config"
	"github.com/astra-service/update-server/internal/manifest"
)

// Server implements the update-server HTTP handlers.
type Server struct {
	cfg           *config.Config
	gen           *manifest.Generator
	healthReports map[string]HealthReport
	mu            sync.RWMutex
}

// HealthReport is a kiosk-reported health payload sent after an update.
type HealthReport struct {
	KioskID    string    `json:"kioskId"`
	Version    string    `json:"version"`
	Healthy    bool      `json:"healthy"`
	Error      string    `json:"error,omitempty"`
	ReceivedAt time.Time `json:"receivedAt"`
}

// New creates an update server from configuration.
func New(cfg *config.Config) (*Server, error) {
	artifacts, err := manifest.LoadArtifacts(cfg.ArtifactsFile)
	if err != nil {
		return nil, err
	}

	rollout := manifest.Rollout{
		Strategy:           cfg.Rollout.Strategy,
		MaxConcurrent:      cfg.Rollout.MaxConcurrent,
		HealthCheckSeconds: cfg.Rollout.HealthCheckSeconds,
	}

	gen := manifest.NewGenerator(cfg.PrivateKey, cfg.Version, cfg.Channel, artifacts, rollout)

	return &Server{
		cfg:           cfg,
		gen:           gen,
		healthReports: make(map[string]HealthReport),
	}, nil
}

// Handler returns an http.Handler with all routes registered.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /manifest.json", s.handleManifest)
	mux.HandleFunc("POST /webhook/health", s.handleHealthWebhook)
	mux.HandleFunc("GET /healthz", s.handleHealthz)
	return mux
}

func (s *Server) handleManifest(w http.ResponseWriter, r *http.Request) {
	signed, err := s.gen.Generate()
	if err != nil {
		slog.Error("failed to generate manifest", "error", err)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "manifest_generation_failed"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(signed)
}

func (s *Server) handleHealthWebhook(w http.ResponseWriter, r *http.Request) {
	var report HealthReport
	if err := json.NewDecoder(r.Body).Decode(&report); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid_body"})
		return
	}

	if report.KioskID == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "missing_kiosk_id"})
		return
	}

	report.ReceivedAt = time.Now().UTC()

	s.mu.Lock()
	s.healthReports[report.KioskID] = report
	s.mu.Unlock()

	slog.Info("received kiosk health report", "kiosk_id", report.KioskID, "healthy", report.Healthy, "version", report.Version)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]any{"status": "accepted"})
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// HealthReport returns the most recent health report for a kiosk.
func (s *Server) HealthReport(kioskID string) (HealthReport, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	report, ok := s.healthReports[kioskID]
	return report, ok
}

type errorResponse struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

// ServeHTTP implements http.Handler for convenience.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.Handler().ServeHTTP(w, r)
}

// ValidateWebhookPayload validates a health webhook payload without mutating state.
func ValidateWebhookPayload(body []byte) (HealthReport, error) {
	var report HealthReport
	if err := json.Unmarshal(body, &report); err != nil {
		return HealthReport{}, err
	}
	if report.KioskID == "" {
		return HealthReport{}, errors.New("missing kioskId")
	}
	return report, nil
}
