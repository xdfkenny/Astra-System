// Command order-service owns the Order aggregate lifecycle: creation from a
// finalized cart, payment confirmation, and fulfillment. It exposes gRPC and
// HTTP/REST APIs and consumes NATS JetStream events.
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"log/slog"
	"os/signal"
	"syscall"
	"time"

	"github.com/astra-service/go-common/eventbus"
	"github.com/astra-service/go-common/observability"
	"github.com/astra-service/go-common/outbox"
	"github.com/astra-systems/astra-service/services/order-service/internal/cartclient"
	"github.com/astra-systems/astra-service/services/order-service/internal/config"
	"github.com/astra-systems/astra-service/services/order-service/internal/handler"
	"github.com/astra-systems/astra-service/services/order-service/internal/repository"
	"github.com/astra-systems/astra-service/services/order-service/internal/server"
	"github.com/astra-systems/astra-service/services/order-service/internal/service"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("order-service: fatal error: %v", err)
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
		ServiceName:    "astra-order-service",
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
		return fmt.Errorf("order-service: open db: %w", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(cfg.DatabaseMaxOpenConns)
	db.SetMaxIdleConns(cfg.DatabaseMaxIdleConns)

	bus, err := eventbus.Connect(ctx, cfg.NatsURL)
	if err != nil {
		return fmt.Errorf("order-service: connect nats: %w", err)
	}
	defer bus.Close()

	cartClient, err := cartclient.NewGRPCClient(cfg.CartServiceTarget)
	if err != nil {
		return fmt.Errorf("order-service: cart client: %w", err)
	}
	defer cartClient.Close()

	repo := repository.NewPostgresRepository(db)
	svc := service.NewOrderService(repo, cartClient)

	relay := outbox.NewRelay(db, bus, resolveSubject)
	go func() {
		if err := relay.Run(ctx); err != nil {
			observability.Error(ctx, "order-service: outbox relay stopped", err)
		}
	}()

	eventHandler := handler.NewEventHandler(svc, bus)
	consumers, err := eventHandler.RegisterConsumers(ctx)
	if err != nil {
		return err
	}
	defer func() {
		for _, c := range consumers {
			c.Stop()
		}
	}()

	srv := server.New(cfg.GRPCPort, cfg.HTTPPort, svc)

	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- srv.Start(ctx)
	}()

	observability.Info(ctx, "order-service started",
		slog.String("grpc_port", cfg.GRPCPort),
		slog.String("http_port", cfg.HTTPPort),
	)

	select {
	case err := <-serverErrors:
		return err
	case <-ctx.Done():
		observability.Info(ctx, "order-service: shutdown signal received, draining...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout())
		defer cancel()
		_ = srv.Start(shutdownCtx)
		observability.Info(ctx, "order-service: graceful shutdown complete")
		return nil
	}
}

func resolveSubject(eventType string) string {
	switch eventType {
	case service.EventTypeOrderCreated:
		return "astra.order.created.v1"
	case service.EventTypeOrderPaid:
		return "astra.order.paid.v1"
	case service.EventTypeOrderFulfilled:
		return "astra.order.fulfilled.v1"
	case service.EventTypeOrderCancelled:
		return "astra.order.cancelled.v1"
	default:
		return eventType
	}
}
