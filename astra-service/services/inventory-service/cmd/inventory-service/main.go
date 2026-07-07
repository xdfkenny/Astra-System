// Command inventory-service owns real-time stock levels and the
// reserved-stock soft-hold pattern. It exposes gRPC and REST interfaces,
// persists inventory as a Postgres ledger, caches derived levels in Redis,
// and publishes domain events via NATS.
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/astra-service/go-common/eventbus"
	"github.com/astra-service/go-common/observability"
	"github.com/astra-service/go-common/outbox"
	"github.com/astra-systems/astra-service/services/inventory-service/internal/cache"
	"github.com/astra-systems/astra-service/services/inventory-service/internal/config"
	"github.com/astra-systems/astra-service/services/inventory-service/internal/publisher"
	"github.com/astra-systems/astra-service/services/inventory-service/internal/repository"
	"github.com/astra-systems/astra-service/services/inventory-service/internal/server"
	"github.com/astra-systems/astra-service/services/inventory-service/internal/service"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/redis/go-redis/v9"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("inventory-service: fatal startup error: %v", err)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	shutdownObs, err := observability.Init(ctx, observability.Config{
		ServiceName:    "astra-inventory-service",
		ServiceVersion: "0.1.0",
		Environment:    cfg.Environment,
		OTLPEndpoint:   cfg.OTELExporterURL,
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
		return fmt.Errorf("inventory-service: open db: %w", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(15 * time.Minute)

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("inventory-service: ping db: %w", err)
	}

	repo := repository.NewPostgresRepository(db)
	if err := repo.Migrate(ctx); err != nil {
		return fmt.Errorf("inventory-service: migrate: %w", err)
	}

	redisOpts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		return fmt.Errorf("inventory-service: parse redis url: %w", err)
	}
	redisClient := redis.NewClient(redisOpts)
	defer redisClient.Close()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("inventory-service: ping redis: %w", err)
	}

	bus, err := eventbus.Connect(ctx, cfg.NatsURL)
	if err != nil {
		return fmt.Errorf("inventory-service: connect nats: %w", err)
	}
	defer bus.Close()

	pub := publisher.NewNATSPublisher(bus, "astra.inventory")
	inventoryCache := cache.NewRedisCache(redisClient)

	inventorySvc := service.NewInventory(repo, inventoryCache, pub, cfg.ReservationTTL, cfg.ReservationSweep)
	inventorySvc.Start(ctx)
	defer inventorySvc.Stop()

	relay := outbox.NewRelay(db, bus, func(eventType string) string {
		return "astra.inventory." + eventType
	})
	go func() {
		if err := relay.Run(ctx); err != nil {
			observability.Error(ctx, "inventory-service: outbox relay stopped", err)
		}
	}()

	srv := server.New(inventorySvc, cfg.GRPCPort, cfg.Port, cfg.EnableReflection)

	observability.Info(ctx, "inventory-service started", slog.String("grpc_port", cfg.GRPCPort), slog.String("http_port", cfg.Port))

	if err := srv.Start(ctx); err != nil {
		return err
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}
	observability.Info(ctx, "inventory-service: graceful shutdown complete")
	return nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
