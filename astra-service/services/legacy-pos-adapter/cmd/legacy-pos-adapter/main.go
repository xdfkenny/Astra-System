// Command legacy-pos-adapter proxies completed Astra carts/orders to a legacy
// POS system when LEGACY_POS_URL is configured, while storing the submission
// outcome in Astra.
package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os/signal"
	"syscall"
	"time"

	"github.com/astra-service/go-common/eventbus"
	"github.com/astra-service/go-common/observability"
	"github.com/astra-systems/astra-service/services/legacy-pos-adapter/internal/client"
	"github.com/astra-systems/astra-service/services/legacy-pos-adapter/internal/config"
	"github.com/astra-systems/astra-service/services/legacy-pos-adapter/internal/handler"
	"github.com/astra-systems/astra-service/services/legacy-pos-adapter/internal/repository"
	"github.com/astra-systems/astra-service/services/legacy-pos-adapter/internal/server"
	"github.com/astra-systems/astra-service/services/legacy-pos-adapter/internal/service"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("legacy-pos-adapter: fatal error: %v", err)
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
		ServiceName:    "astra-legacy-pos-adapter",
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

	bus, err := eventbus.Connect(ctx, cfg.NatsURL)
	if err != nil {
		return fmt.Errorf("legacy-pos-adapter: connect nats: %w", err)
	}
	defer bus.Close()

	repo := repository.NewMemoryRepository()

	var posClient *client.Client
	if cfg.Enabled() {
		posClient = client.New(cfg.LegacyPOSURL, cfg.LegacyPOSAPIKey, cfg.LegacyPOSTimeout)
	}

	svc := service.New(repo, posClient, cfg.Enabled())

	eventHandler := handler.NewEventHandler(svc)
	consumers, err := eventHandler.RegisterConsumers(ctx, bus)
	if err != nil {
		return err
	}
	defer func() {
		for _, c := range consumers {
			c.Stop()
		}
	}()

	srv := server.New(cfg.GRPCPort, cfg.HTTPPort, svc, repo)

	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- srv.Start(ctx)
	}()

	observability.Info(ctx, "legacy-pos-adapter started",
		slog.String("grpc_port", cfg.GRPCPort),
		slog.String("http_port", cfg.HTTPPort),
		slog.Bool("legacy_pos_enabled", cfg.Enabled()),
	)

	select {
	case err := <-serverErrors:
		return err
	case <-ctx.Done():
		observability.Info(ctx, "legacy-pos-adapter: shutdown signal received, draining...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout())
		defer cancel()
		_ = srv.Start(shutdownCtx)
		observability.Info(ctx, "legacy-pos-adapter: graceful shutdown complete")
		return nil
	}
}
