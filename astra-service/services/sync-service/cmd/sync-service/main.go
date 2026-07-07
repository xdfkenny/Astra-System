// Command sync-service runs the cloud-side gateway for the Astra kiosk mesh.
// It exposes a gRPC SyncService plus a JSON REST fallback, authenticates kiosk
// leaders, ingests batches into PostgreSQL, publishes notifications over NATS,
// and computes delta sets for downstream kiosks.
package main

import (
	"context"
	"log"
	"log/slog"
	"os/signal"
	"syscall"
	"time"

	commonbus "github.com/astra-service/go-common/eventbus"
	"github.com/astra-service/go-common/observability"
	"github.com/astra-systems/astra-service/services/sync-service/internal/config"
	synceventbus "github.com/astra-systems/astra-service/services/sync-service/internal/eventbus"
	"github.com/astra-systems/astra-service/services/sync-service/internal/repository"
	"github.com/astra-systems/astra-service/services/sync-service/internal/server"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("sync-service: fatal startup error: %v", err)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	logger := slog.Default()

	shutdownObs, err := observability.Init(ctx, observability.Config{
		ServiceName:    "astra-sync-service",
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

	store, err := repository.NewPostgresStore(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer store.Close()

	bus, err := commonbus.Connect(ctx, cfg.NatsURL)
	if err != nil {
		return err
	}
	defer bus.Close()

	publisher := synceventbus.NewNATSPublisher(bus)

	health := observability.NewCompositeChecker()
	health.Register("postgres", observability.CheckFunc(func(ctx context.Context) error {
		return store.Ping(ctx)
	}))
	health.Register("nats", &observability.NATSCheck{Bus: bus})

	srv, err := server.New(server.Config{
		GRPCPort:       cfg.GRPCPort,
		HTTPPort:       cfg.HTTPPort,
		RequestTimeout: cfg.RequestTimeout,
	}, store, publisher, health, logger)
	if err != nil {
		return err
	}

	observability.Info(ctx, "sync-service started",
		slog.String("grpc_port", cfg.GRPCPort),
		slog.String("http_port", cfg.HTTPPort),
	)

	if err := srv.Start(ctx, logger); err != nil {
		return err
	}
	observability.Info(ctx, "sync-service shutdown complete")
	return nil
}
