// Command webauthn-service verifies WebAuthn/Passkey assertions and issues
// short-lived override tokens for employee or admin override flows.
package main

import (
	"context"
	"database/sql"
	"log"
	"log/slog"
	"os/signal"
	"syscall"
	"time"

	"github.com/astra-service/go-common/observability"
	"github.com/astra-systems/astra-service/services/webauthn-service/internal/config"
	"github.com/astra-systems/astra-service/services/webauthn-service/internal/repository"
	"github.com/astra-systems/astra-service/services/webauthn-service/internal/server"
	"github.com/astra-systems/astra-service/services/webauthn-service/internal/service"
	"github.com/astra-systems/astra-service/services/webauthn-service/internal/webauthn"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("webauthn-service: fatal error: %v", err)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	shutdownObs, err := observability.Init(ctx, observability.Config{
		ServiceName:    "astra-webauthn-service",
		ServiceVersion: "0.1.0",
		Environment:    cfg.Environment,
		OTLPEndpoint:   cfg.OTLPEndpoint,
		SampleRatio:    1.0,
	})
	if err != nil {
		return err
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = shutdownObs(shutdownCtx)
	}()

	db, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()
	db.SetMaxOpenConns(cfg.DatabaseMaxOpenConns)
	db.SetMaxIdleConns(cfg.DatabaseMaxIdleConns)

	repo := repository.NewPostgresRepository(db)
	verifier, err := webauthn.NewLibraryVerifier(cfg.RPID, cfg.RPOrigin, cfg.RPName)
	if err != nil {
		return err
	}
	svc := service.NewAuthService(repo, verifier, []byte(cfg.OverrideJWTSecret), cfg.OverrideTokenTTL)
	webauthnSvc := service.NewWebAuthnService(repo, verifier)
	srv := server.NewWithWebAuthn(cfg.GRPCPort, cfg.HTTPPort, svc, webauthnSvc)

	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- srv.Start(ctx)
	}()

	observability.Info(ctx, "webauthn-service started",
		slog.String("grpc_port", cfg.GRPCPort),
		slog.String("http_port", cfg.HTTPPort),
	)

	select {
	case err := <-serverErrors:
		return err
	case <-ctx.Done():
		observability.Info(ctx, "webauthn-service: shutdown signal received, draining...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout())
		defer cancel()
		_ = srv.Start(shutdownCtx)
		observability.Info(ctx, "webauthn-service: graceful shutdown complete")
		return nil
	}
}
