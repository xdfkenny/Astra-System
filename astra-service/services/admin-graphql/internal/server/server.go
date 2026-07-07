// Package server exposes the admin GraphQL API over HTTP.
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/astra-service/go-common/observability"
	"github.com/astra-systems/astra-service/services/admin-graphql/internal/auth"
	"github.com/astra-systems/astra-service/services/admin-graphql/internal/config"
	"github.com/graphql-go/graphql"
)

// Server hosts the HTTP GraphQL server.
type Server struct {
	cfg        *config.Config
	httpServer *http.Server
	schema     graphql.Schema
}

// New creates a Server from the supplied config and GraphQL schema.
func New(cfg *config.Config, schema graphql.Schema) *Server {
	return &Server{cfg: cfg, schema: schema}
}

// Start launches the HTTP server and blocks until the context is cancelled.
func (s *Server) Start(ctx context.Context) error {
	authCfg := auth.Config{
		Secret:   s.cfg.JWTSecret,
		Issuer:   s.cfg.JWTIssuer,
		Audience: s.cfg.JWTAudience,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealthz)
	mux.Handle("/graphql", auth.Middleware(authCfg)(http.HandlerFunc(s.handleGraphQL)))

	s.httpServer = &http.Server{
		Addr:         ":" + s.cfg.HTTPPort,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	observability.Info(ctx, "admin-graphql started", slog.String("http_port", s.cfg.HTTPPort))

	serverErrors := make(chan error, 1)
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErrors <- fmt.Errorf("server: http serve: %w", err)
		}
	}()

	select {
	case err := <-serverErrors:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), s.cfg.ShutdownTimeout())
		defer cancel()
		return s.httpServer.Shutdown(shutdownCtx)
	}
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func (s *Server) handleGraphQL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req graphQLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}

	result := graphql.Do(graphql.Params{
		Schema:         s.schema,
		RequestString:  req.Query,
		VariableValues: req.Variables,
		OperationName:  req.OperationName,
		Context:        r.Context(),
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(result)
}

type graphQLRequest struct {
	Query         string                 `json:"query"`
	Variables     map[string]interface{} `json:"variables,omitempty"`
	OperationName string                 `json:"operationName,omitempty"`
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(`{"errors":[{"message":"` + message + `"}]}`))
}
