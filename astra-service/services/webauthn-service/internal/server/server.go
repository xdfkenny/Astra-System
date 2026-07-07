// Package server exposes the AuthService over gRPC and HTTP/REST.
package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	authv1 "github.com/astra-systems/astra-service/proto/gen/go/auth"
	webauthnv1 "github.com/astra-systems/astra-service/proto/gen/go/webauthn"
	"github.com/astra-systems/astra-service/services/webauthn-service/internal/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// Server hosts both gRPC and HTTP listeners.
type Server struct {
	authv1.UnimplementedAuthServiceServer
	webauthnv1.UnimplementedWebAuthnServiceServer
	grpcPort    string
	httpPort    string
	grpcServer  *grpc.Server
	httpServer  *http.Server
	service     *service.AuthService
	webauthnSvc *service.WebAuthnService
}

// New constructs a server bound to the supplied auth service.
func New(grpcPort, httpPort string, svc *service.AuthService) *Server {
	return &Server{
		grpcPort: grpcPort,
		httpPort: httpPort,
		service:  svc,
	}
}

// NewWithWebAuthn constructs a server bound to both auth and WebAuthn services.
func NewWithWebAuthn(grpcPort, httpPort string, svc *service.AuthService, webauthnSvc *service.WebAuthnService) *Server {
	return &Server{
		grpcPort:    grpcPort,
		httpPort:    httpPort,
		service:     svc,
		webauthnSvc: webauthnSvc,
	}
}

// Start registers handlers and begins serving gRPC and HTTP traffic. It blocks
// until the provided context is cancelled, then performs a graceful shutdown.
func (s *Server) Start(ctx context.Context) error {
	errCh := make(chan error, 2)

	s.grpcServer = grpc.NewServer()
	authv1.RegisterAuthServiceServer(s.grpcServer, s)
	if s.webauthnSvc != nil {
		webauthnv1.RegisterWebAuthnServiceServer(s.grpcServer, s)
	}

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
	mux.HandleFunc("/v1/auth/webauthn/begin", s.handleBeginVerification)
	mux.HandleFunc("/v1/auth/webauthn/verify", s.handleVerifyAssertion)
	mux.HandleFunc("/v1/auth/override/validate", s.handleValidateOverrideToken)
	if s.webauthnSvc != nil {
		mux.HandleFunc("/v1/webauthn/register/begin", s.handleBeginRegistration)
		mux.HandleFunc("/v1/webauthn/register/finish", s.handleFinishRegistration)
		mux.HandleFunc("/v1/webauthn/authenticate/begin", s.handleBeginAuthentication)
		mux.HandleFunc("/v1/webauthn/authenticate/finish", s.handleFinishAuthentication)
	}
	mux.HandleFunc("/healthz", s.handleHealthz)
	return mux
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func (s *Server) handleBeginRegistration(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req webauthnv1.BeginRegistrationRequest
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("read body: %v", err))
		return
	}
	if err := protojson.Unmarshal(body, &req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}
	resp, err := s.webauthnSvc.BeginRegistration(r.Context(), &req)
	if err != nil {
		writeGRPCError(w, err)
		return
	}
	writeProtoJSON(w, http.StatusOK, resp)
}

func (s *Server) handleFinishRegistration(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req webauthnv1.FinishRegistrationRequest
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("read body: %v", err))
		return
	}
	if err := protojson.Unmarshal(body, &req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}
	resp, err := s.webauthnSvc.FinishRegistration(r.Context(), &req)
	if err != nil {
		writeGRPCError(w, err)
		return
	}
	writeProtoJSON(w, http.StatusOK, resp)
}

func (s *Server) handleBeginAuthentication(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req webauthnv1.BeginAuthenticationRequest
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("read body: %v", err))
		return
	}
	if err := protojson.Unmarshal(body, &req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}
	resp, err := s.webauthnSvc.BeginAuthentication(r.Context(), &req)
	if err != nil {
		writeGRPCError(w, err)
		return
	}
	writeProtoJSON(w, http.StatusOK, resp)
}

func (s *Server) handleFinishAuthentication(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req webauthnv1.FinishAuthenticationRequest
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("read body: %v", err))
		return
	}
	if err := protojson.Unmarshal(body, &req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}
	resp, err := s.webauthnSvc.FinishAuthentication(r.Context(), &req)
	if err != nil {
		writeGRPCError(w, err)
		return
	}
	writeProtoJSON(w, http.StatusOK, resp)
}

func (s *Server) handleBeginVerification(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req authv1.BeginVerificationRequest
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("read body: %v", err))
		return
	}
	if err := protojson.Unmarshal(body, &req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}
	resp, err := s.service.BeginVerification(r.Context(), &req)
	if err != nil {
		writeGRPCError(w, err)
		return
	}
	writeProtoJSON(w, http.StatusOK, resp)
}

func (s *Server) handleVerifyAssertion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req authv1.VerifyAssertionRequest
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("read body: %v", err))
		return
	}
	if err := protojson.Unmarshal(body, &req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}
	resp, err := s.service.VerifyAssertion(r.Context(), &req)
	if err != nil {
		writeGRPCError(w, err)
		return
	}
	writeProtoJSON(w, http.StatusOK, resp)
}

func (s *Server) handleValidateOverrideToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req authv1.ValidateOverrideTokenRequest
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("read body: %v", err))
		return
	}
	if err := protojson.Unmarshal(body, &req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}
	resp, err := s.service.ValidateOverrideToken(r.Context(), &req)
	if err != nil {
		writeGRPCError(w, err)
		return
	}
	writeProtoJSON(w, http.StatusOK, resp)
}

// gRPC implementation of AuthServiceServer.

// BeginVerification implements the generated AuthServiceServer interface.
func (s *Server) BeginVerification(ctx context.Context, req *authv1.BeginVerificationRequest) (*authv1.BeginVerificationResponse, error) {
	return s.service.BeginVerification(ctx, req)
}

// VerifyAssertion implements the generated AuthServiceServer interface.
func (s *Server) VerifyAssertion(ctx context.Context, req *authv1.VerifyAssertionRequest) (*authv1.VerifyAssertionResponse, error) {
	return s.service.VerifyAssertion(ctx, req)
}

// ValidateOverrideToken implements the generated AuthServiceServer interface.
func (s *Server) ValidateOverrideToken(ctx context.Context, req *authv1.ValidateOverrideTokenRequest) (*authv1.ValidateOverrideTokenResponse, error) {
	return s.service.ValidateOverrideToken(ctx, req)
}

// gRPC implementation of WebAuthnServiceServer.

// BeginRegistration implements the generated WebAuthnServiceServer interface.
func (s *Server) BeginRegistration(ctx context.Context, req *webauthnv1.BeginRegistrationRequest) (*webauthnv1.BeginRegistrationResponse, error) {
	return s.webauthnSvc.BeginRegistration(ctx, req)
}

// FinishRegistration implements the generated WebAuthnServiceServer interface.
func (s *Server) FinishRegistration(ctx context.Context, req *webauthnv1.FinishRegistrationRequest) (*webauthnv1.FinishRegistrationResponse, error) {
	return s.webauthnSvc.FinishRegistration(ctx, req)
}

// BeginAuthentication implements the generated WebAuthnServiceServer interface.
func (s *Server) BeginAuthentication(ctx context.Context, req *webauthnv1.BeginAuthenticationRequest) (*webauthnv1.BeginAuthenticationResponse, error) {
	return s.webauthnSvc.BeginAuthentication(ctx, req)
}

// FinishAuthentication implements the generated WebAuthnServiceServer interface.
func (s *Server) FinishAuthentication(ctx context.Context, req *webauthnv1.FinishAuthenticationRequest) (*webauthnv1.FinishAuthenticationResponse, error) {
	return s.webauthnSvc.FinishAuthentication(ctx, req)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		_, _ = w.Write([]byte(`{"error":"failed to encode response"}`))
	}
}

func writeProtoJSON(w http.ResponseWriter, status int, msg proto.Message) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	b, err := protojson.Marshal(msg)
	if err != nil {
		_, _ = w.Write([]byte(`{"error":"failed to encode response"}`))
		return
	}
	_, _ = w.Write(b)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func writeGRPCError(w http.ResponseWriter, err error) {
	st, ok := status.FromError(err)
	if !ok {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpStatus := http.StatusInternalServerError
	switch st.Code() {
	case codes.InvalidArgument:
		httpStatus = http.StatusBadRequest
	case codes.NotFound:
		httpStatus = http.StatusNotFound
	case codes.AlreadyExists:
		httpStatus = http.StatusConflict
	case codes.FailedPrecondition:
		httpStatus = http.StatusPreconditionFailed
	case codes.Unauthenticated:
		httpStatus = http.StatusUnauthorized
	case codes.PermissionDenied:
		httpStatus = http.StatusForbidden
	}
	writeError(w, httpStatus, st.Message())
}

// stripServicePrefix removes the /v1/auth prefix when proxied through the gateway.
func stripServicePrefix(path string) string {
	prefix := "/v1/auth"
	if strings.HasPrefix(path, prefix) {
		return strings.TrimPrefix(path, prefix)
	}
	return path
}

// Ensure Server satisfies the generated AuthServiceServer interface.
var _ authv1.AuthServiceServer = (*Server)(nil)

// Ensure Server satisfies the generated WebAuthnServiceServer interface.
var _ webauthnv1.WebAuthnServiceServer = (*Server)(nil)
