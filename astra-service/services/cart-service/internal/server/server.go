// Package server bootstraps the gRPC and REST servers for cart-service.
package server

import (
	"context"
	"fmt"
	"log/slog"
	"net"

	"github.com/astra-service/go-common/observability"
	cartv1 "github.com/astra-systems/astra-service/proto/gen/go/cart"
	"github.com/astra-systems/astra-service/services/cart-service/internal/config"
	"github.com/astra-systems/astra-service/services/cart-service/internal/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// Server holds the gRPC listener.
type Server struct {
	cfg        *config.Config
	cartSvc    *service.CartService
	grpcServer *grpc.Server
}

// New creates a Server that serves the cart gRPC service.
func New(cfg *config.Config, cartSvc *service.CartService) *Server {
	grpcServer := grpc.NewServer()
	cartv1.RegisterCartServiceServer(grpcServer, cartSvc)
	reflection.Register(grpcServer)

	return &Server{
		cfg:        cfg,
		cartSvc:    cartSvc,
		grpcServer: grpcServer,
	}
}

// ListenAndServe starts the gRPC server.
func (s *Server) ListenAndServe(ctx context.Context) error {
	errCh := make(chan error, 1)

	go func() {
		if err := s.serveGRPC(ctx); err != nil {
			errCh <- fmt.Errorf("grpc server: %w", err)
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return s.Shutdown(context.Background())
	}
}

func (s *Server) serveGRPC(ctx context.Context) error {
	lis, err := net.Listen("tcp", ":"+s.cfg.GRPCPort)
	if err != nil {
		return fmt.Errorf("listen grpc port %s: %w", s.cfg.GRPCPort, err)
	}
	observability.Info(ctx, "cart-service gRPC listening", slog.String("port", s.cfg.GRPCPort))
	if err := s.grpcServer.Serve(lis); err != nil {
		return fmt.Errorf("serve grpc: %w", err)
	}
	return nil
}

// Shutdown gracefully stops the gRPC server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.grpcServer.GracefulStop()
	return nil
}
