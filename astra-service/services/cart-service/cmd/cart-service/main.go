// Command cart-service runs the Cart domain microservice. It exposes a gRPC
// API (and REST gateway) defined by astra.cart.v1.CartService, persists carts
// to Postgres with optimistic locking, caches active sessions in Redis, and
// publishes domain events through the transactional outbox + NATS.
package main

import (
	"context"
	"database/sql"
	"log"
	"os/signal"
	"syscall"
	"time"

	"github.com/astra-service/go-common/eventbus"
	"github.com/astra-service/go-common/observability"
	commonoutbox "github.com/astra-service/go-common/outbox"
	"github.com/astra-systems/astra-service/services/cart-service/internal/cache"
	"github.com/astra-systems/astra-service/services/cart-service/internal/config"
	"github.com/astra-systems/astra-service/services/cart-service/internal/inventory"
	"github.com/astra-systems/astra-service/services/cart-service/internal/outbox"
	"github.com/astra-systems/astra-service/services/cart-service/internal/repository"
	"github.com/astra-systems/astra-service/services/cart-service/internal/server"
	"github.com/astra-systems/astra-service/services/cart-service/internal/service"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/redis/go-redis/v9"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	cfg, err := config.LoadFromEnv()
	if err != nil {
		log.Fatalf("cart-service: load config: %v", err)
	}

	shutdownObs, err := observability.Init(ctx, observability.Config{
		ServiceName:    cfg.ServiceName,
		ServiceVersion: cfg.ServiceVersion,
		Environment:    cfg.Environment,
		OTLPEndpoint:   cfg.OTELExporterEndpoint,
		SampleRatio:    1.0,
	})
	if err != nil {
		log.Fatalf("cart-service: observability init error: %v", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = shutdownObs(shutdownCtx)
	}()

	db, err := openDB(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("cart-service: open db: %v", err)
	}
	defer db.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	defer redisClient.Close()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("cart-service: redis ping: %v", err)
	}

	bus, err := eventbus.Connect(ctx, cfg.NATSURL)
	if err != nil {
		log.Fatalf("cart-service: connect nats: %v", err)
	}
	defer bus.Close()

	invClient, err := inventory.NewClient(ctx, cfg.InventoryServiceAddr)
	if err != nil {
		log.Fatalf("cart-service: inventory client: %v", err)
	}
	defer invClient.Close()

	repo := repository.NewCartRepository(db)
	cartCache := cache.NewCartCache(redisClient, cfg.RedisSessionTTL)
	cartSvc := service.NewCartService(repo, cartCache, invClient, "USD")

	relay := commonoutbox.NewRelay(db, bus, outbox.SubjectResolver)
	go func() {
		if err := relay.Run(ctx); err != nil {
			observability.Error(ctx, "cart-service: outbox relay stopped", err)
		}
	}()

	srv := server.New(cfg, cartSvc)
	if err := srv.ListenAndServe(ctx); err != nil {
		log.Fatalf("cart-service: server error: %v", err)
	}
}

func openDB(ctx context.Context, url string) (*sql.DB, error) {
	db, err := sql.Open("pgx", url)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(15 * time.Minute)

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}
