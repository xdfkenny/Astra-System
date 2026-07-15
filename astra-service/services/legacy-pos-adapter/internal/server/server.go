// Package server exposes the legacy-pos-adapter over gRPC and HTTP/REST.
package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	orderv1 "github.com/astra-systems/astra-service/proto/gen/go/order"
	"github.com/astra-systems/astra-service/services/legacy-pos-adapter/internal/repository"
	"github.com/astra-systems/astra-service/services/legacy-pos-adapter/internal/service"
	"google.golang.org/grpc"
)

// Server hosts both gRPC and HTTP listeners.
type Server struct {
	orderv1.UnimplementedOrderServiceServer
	grpcPort   string
	httpPort   string
	grpcServer *grpc.Server
	httpServer *http.Server
	service    *service.AdapterService
	repo       repository.Repository
}

// New constructs a server bound to the supplied service and repository.
func New(grpcPort, httpPort string, svc *service.AdapterService, repo repository.Repository) *Server {
	return &Server{
		grpcPort: grpcPort,
		httpPort: httpPort,
		service:  svc,
		repo:     repo,
	}
}

// Start registers handlers and begins serving gRPC and HTTP traffic. It blocks
// until the provided context is cancelled, then performs a graceful shutdown.
func (s *Server) Start(ctx context.Context) error {
	errCh := make(chan error, 2)

	s.grpcServer = grpc.NewServer()
	orderv1.RegisterOrderServiceServer(s.grpcServer, s)

	lis, err := net.Listen("tcp", ":"+s.grpcPort)
	if err != nil {
		return fmt.Errorf("server: listen grpc: %w", err)
	}

	go func() {
		if err := s.grpcServer.Serve(lis); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			errCh <- fmt.Errorf("server: grpc serve: %w", err)
		}
	}()

	s.httpServer = &http.Server{
		Addr:         ":" + s.httpPort,
		Handler:      s.mux(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("server: http serve: %w", err)
		}
	}()

	select {
	case <-ctx.Done():
		return s.shutdown()
	case err := <-errCh:
		_ = s.shutdown()
		return err
	}
}

func (s *Server) shutdown() error {
	if s.grpcServer != nil {
		s.grpcServer.GracefulStop()
	}
	if s.httpServer != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("server: http shutdown: %w", err)
		}
	}
	return nil
}

func (s *Server) mux() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/submissions/", s.handleSubmission)
	mux.HandleFunc("/healthz", s.handleHealthz)
	return mux
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleSubmission(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/v1/submissions/")
	if path == "" {
		writeError(w, http.StatusBadRequest, "submission id required")
		return
	}

	submission, err := s.repo.GetSubmission(r.Context(), path)
	if err != nil {
		if errors.Is(err, repository.ErrSubmissionNotFound) {
			writeError(w, http.StatusNotFound, "submission not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, submission)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		_, _ = w.Write([]byte(`{"error":"failed to encode response"}`))
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// compile-time interface assertion.
var _ orderv1.OrderServiceServer = (*Server)(nil)
