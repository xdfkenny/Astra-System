// Command gateway runs the Astra-Service Fiber v3 API gateway.
//
//	@title			Astra-Service Gateway API
//	@version		0.1.0
//	@description	API gateway for Astra-Service kiosk applications.
//	@BasePath		/
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

	"github.com/astra-systems/astra-service/services/gateway/internal/clients"
	"github.com/astra-systems/astra-service/services/gateway/internal/config"
	"github.com/astra-systems/astra-service/services/gateway/internal/health"
	"github.com/astra-systems/astra-service/services/gateway/internal/routes"
	"github.com/astra-systems/astra-service/services/gateway/internal/server"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("gateway: fatal startup error: %v", err)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	db, err := openPostgres(cfg.PostgresURL)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			slog.LogAttrs(context.Background(), slog.LevelError, "postgres_close_error", slog.String("error", closeErr.Error()))
		}
	}()

	redisOpts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		return fmt.Errorf("parse redis url: %w", err)
	}
	redisClient := redis.NewClient(redisOpts)
	defer func() {
		if closeErr := redisClient.Close(); closeErr != nil {
			slog.LogAttrs(context.Background(), slog.LevelError, "redis_close_error", slog.String("error", closeErr.Error()))
		}
	}()

	natsConn, err := nats.Connect(cfg.NatsURL)
	if err != nil {
		return fmt.Errorf("connect to nats: %w", err)
	}
	defer natsConn.Close()

	serviceClients, err := clients.NewRegistry(cfg)
	if err != nil {
		return fmt.Errorf("create service clients: %w", err)
	}
	defer func() {
		if closeErr := serviceClients.Close(); closeErr != nil {
			slog.LogAttrs(context.Background(), slog.LevelError, "grpc_clients_close_error", slog.String("error", closeErr.Error()))
		}
	}()

	checker := health.NewCompositeChecker(db, redisClient, natsConn)
	app := server.New(cfg, redisClient)
	routes.Register(app, cfg, checker, serviceClients)

	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- app.Listen(":" + cfg.Port)
	}()

	slog.LogAttrs(ctx, slog.LevelInfo, "gateway_started", slog.String("port", cfg.Port))

	select {
	case err := <-serverErrors:
		return err
	case <-ctx.Done():
		slog.LogAttrs(ctx, slog.LevelInfo, "gateway_shutdown_signal_received")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := app.ShutdownWithContext(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown: %w", err)
		}
		slog.LogAttrs(ctx, slog.LevelInfo, "gateway_graceful_shutdown_complete")
		return nil
	}
}

func openPostgres(url string) (*sql.DB, error) {
	db, err := sql.Open("pgx", url)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(15 * time.Minute)
	return db, nil
}
