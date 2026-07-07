package server

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/astra-service/go-common/observability"
	menupb "github.com/astra-systems/astra-service/proto/gen/go/menu"
	"github.com/astra-systems/astra-service/services/menu-service/internal/config"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

// Server hosts the gRPC and optional REST gateway servers.
type Server struct {
	cfg        *config.Config
	grpcServer *grpc.Server
	httpServer *http.Server
	menuSvc    menupb.MenuServiceServer
}

// NewServer creates a Server that serves the provided MenuService implementation.
func NewServer(cfg *config.Config, menuSvc menupb.MenuServiceServer) *Server {
	return &Server{
		cfg:     cfg,
		menuSvc: menuSvc,
	}
}

// Start launches the gRPC server and the REST gateway.
func (s *Server) Start(ctx context.Context) error {
	grpcAddr := ":" + s.cfg.GRPCPort
	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		return fmt.Errorf("server: listen grpc: %w", err)
	}

	s.grpcServer = grpc.NewServer(
		grpc.ConnectionTimeout(10 * time.Second),
	)
	menupb.RegisterMenuServiceServer(s.grpcServer, s.menuSvc)
	grpc_health_v1.RegisterHealthServer(s.grpcServer, health.NewServer())
	reflection.Register(s.grpcServer)

	errCh := make(chan error, 2)
	go func() {
		observability.Info(ctx, "menu gRPC server started", slog.String("addr", grpcAddr))
		if err := s.grpcServer.Serve(lis); err != nil {
			errCh <- fmt.Errorf("server: grpc serve: %w", err)
		}
	}()

	go func() {
		if err := s.startGateway(ctx, grpcAddr); err != nil {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return nil
	}
}

func (s *Server) startGateway(ctx context.Context, grpcAddr string) error {
	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	if err := menupb.RegisterMenuServiceHandlerFromEndpoint(ctx, mux, "localhost"+grpcAddr, opts); err != nil {
		return fmt.Errorf("server: register gateway handler: %w", err)
	}

	s.httpServer = &http.Server{
		Addr:         ":" + s.cfg.HTTPPort,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	observability.Info(ctx, "menu REST gateway started", slog.String("addr", s.httpServer.Addr))
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server: gateway serve: %w", err)
	}
	return nil
}

// Shutdown gracefully stops both servers.
func (s *Server) Shutdown(ctx context.Context) error {
	if s.grpcServer != nil {
		s.grpcServer.GracefulStop()
	}
	if s.httpServer != nil {
		if err := s.httpServer.Shutdown(ctx); err != nil {
			return fmt.Errorf("server: shutdown gateway: %w", err)
		}
	}
	return nil
}
