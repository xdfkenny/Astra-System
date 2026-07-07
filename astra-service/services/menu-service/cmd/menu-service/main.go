// Command menu-service runs the gRPC menu catalog service with a REST gateway.
package main

import (
	"context"
	"database/sql"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/astra-service/go-common/eventbus"
	"github.com/astra-service/go-common/observability"
	"github.com/astra-systems/astra-service/services/menu-service/internal/cache"
	"github.com/astra-systems/astra-service/services/menu-service/internal/config"
	"github.com/astra-systems/astra-service/services/menu-service/internal/repository"
	"github.com/astra-systems/astra-service/services/menu-service/internal/relay"
	"github.com/astra-systems/astra-service/services/menu-service/internal/server"
	"github.com/astra-systems/astra-service/services/menu-service/internal/service"
	"github.com/redis/go-redis/v9"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("menu-service: fatal startup error: %v", err)
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
		ServiceName:    cfg.ServiceName,
		ServiceVersion: cfg.ServiceVersion,
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
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(15 * time.Minute)

	repo, err := repository.NewRepository(db)
	if err != nil {
		return err
	}
	defer repo.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisURL,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	defer redisClient.Close()

	menuCache := cache.New(redisClient)
	if err := menuCache.Ping(ctx); err != nil {
		observability.Warn(ctx, "redis ping failed; continuing without cache", slog.Any("error", err))
	}

	bus, err := eventbus.Connect(ctx, cfg.NATSURL)
	if err != nil {
		return err
	}
	defer bus.Close()

	relay := relay.New(db, bus)
	relayCtx, relayCancel := context.WithCancel(ctx)
	defer relayCancel()
	go func() {
		if err := relay.Run(relayCtx); err != nil {
			observability.Error(relayCtx, "outbox relay stopped", err)
		}
	}()

	menuSvc := service.NewMenuService(repo, menuCache, cfg.CacheTTL)
	srv := server.NewServer(cfg, menuSvc)

	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- srv.Start(ctx)
	}()

	observability.Info(ctx, "menu-service started",
		slog.String("grpc_port", cfg.GRPCPort),
		slog.String("http_port", cfg.HTTPPort),
	)

	select {
	case err := <-serverErrors:
		return err
	case <-ctx.Done():
		observability.Info(ctx, "menu-service: shutdown signal received, draining...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		relayCancel()
		return srv.Shutdown(shutdownCtx)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
