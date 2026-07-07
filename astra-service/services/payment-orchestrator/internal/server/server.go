// Package server wires the gRPC and REST servers together and exposes graceful
// start/shutdown helpers.
package server

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/astra-service/go-common/observability"
	paymentv1 "github.com/astra-systems/astra-service/proto/gen/go/payment"
	"github.com/astra-systems/astra-service/services/payment-orchestrator/internal/config"
	"github.com/astra-systems/astra-service/services/payment-orchestrator/internal/handler"
	"github.com/astra-systems/astra-service/services/payment-orchestrator/internal/service"
	"github.com/gofiber/fiber/v3"
	"google.golang.org/grpc"
)

// Server holds the gRPC and REST listeners.
type Server struct {
	cfg        *config.Config
	grpc       *grpc.Server
	rest       *fiber.App
	handler    *handler.REST
	grpcImpl   *service.Payment
	health     observability.Checkable
	grpcLn     net.Listener
	restLn     net.Listener
}

// New creates a combined gRPC/REST server.
func New(cfg *config.Config, grpcImpl *service.Payment, h *handler.REST, health observability.Checkable) (*Server, error) {
	grpcLn, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		return nil, fmt.Errorf("server: listen grpc: %w", err)
	}
	restLn, err := net.Listen("tcp", ":"+cfg.Port)
	if err != nil {
		return nil, fmt.Errorf("server: listen rest: %w", err)
	}

	grpcServer := grpc.NewServer()
	paymentv1.RegisterPaymentOrchestratorServer(grpcServer, grpcImpl)

	restApp := fiber.New(fiber.Config{
		AppName:      "astra-payment-orchestrator",
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	})
	h.RegisterRoutes(restApp, health)

	return &Server{
		cfg:      cfg,
		grpc:     grpcServer,
		rest:     restApp,
		handler:  h,
		grpcImpl: grpcImpl,
		health:   health,
		grpcLn:   grpcLn,
		restLn:   restLn,
	}, nil
}

// Start runs both servers in goroutines and returns a channel for errors.
func (s *Server) Start(ctx context.Context) (<-chan error, error) {
	errCh := make(chan error, 2)

	go func() {
		observability.Info(ctx, "gRPC server started", slog.String("port", s.cfg.GRPCPort))
		if err := s.grpc.Serve(s.grpcLn); err != nil {
			errCh <- fmt.Errorf("grpc serve: %w", err)
		}
	}()

	go func() {
		observability.Info(ctx, "REST server started", slog.String("port", s.cfg.Port))
		if err := s.rest.Server().Serve(s.restLn); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("rest serve: %w", err)
		}
	}()

	return errCh, nil
}

// Shutdown gracefully stops both servers.
func (s *Server) Shutdown(ctx context.Context) error {
	s.grpc.GracefulStop()

	shutdownRest, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := s.rest.ShutdownWithContext(shutdownRest); err != nil {
		return fmt.Errorf("server: rest shutdown: %w", err)
	}
	return nil
}
