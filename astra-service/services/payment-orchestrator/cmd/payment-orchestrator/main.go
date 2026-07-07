// Command payment-orchestrator runs the payment state-machine service.
package main

import (
	"context"
	"database/sql"
	"log"
	"log/slog"
	"os/signal"
	"syscall"
	"time"

	"github.com/astra-service/go-common/eventbus"
	"github.com/astra-service/go-common/observability"
	"github.com/astra-service/go-common/outbox"
	"github.com/astra-systems/astra-service/services/payment-orchestrator/internal/client"
	"github.com/astra-systems/astra-service/services/payment-orchestrator/internal/config"
	"github.com/astra-systems/astra-service/services/payment-orchestrator/internal/events"
	"github.com/astra-systems/astra-service/services/payment-orchestrator/internal/handler"
	"github.com/astra-systems/astra-service/services/payment-orchestrator/internal/idempotency"
	"github.com/astra-systems/astra-service/services/payment-orchestrator/internal/offline"
	"github.com/astra-systems/astra-service/services/payment-orchestrator/internal/repository"
	"github.com/astra-systems/astra-service/services/payment-orchestrator/internal/server"
	"github.com/astra-systems/astra-service/services/payment-orchestrator/internal/service"
	"github.com/redis/go-redis/v9"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("payment-orchestrator: fatal startup error: %v", err)
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
		ServiceName:    "astra-payment-orchestrator",
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
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(15 * time.Minute)

	var rdb *redis.Client
	if cfg.RedisURL != "" {
		opt, err := redis.ParseURL(cfg.RedisURL)
		if err != nil {
			return err
		}
		rdb = redis.NewClient(opt)
	}

	bus, err := eventbus.Connect(ctx, cfg.NatsURL)
	if err != nil {
		return err
	}
	defer bus.Close()

	health := observability.NewCompositeChecker()
	health.Register("postgres", &observability.DBCheck{DB: db})
	health.Register("nats", &observability.NATSCheck{Bus: bus})

	repo := repository.NewPaymentRepository(db)
	idemStore := idempotency.NewStore(db, rdb)

	vf, err := client.New(cfg.VerifoneGRPCAddr, cfg.VerifoneHTTPURL)
	if err != nil {
		return err
	}
	defer vf.Close()

	offlineVerifier := offline.NewVerifier(cfg.OfflineTokenSecret)
	offlineSettler := offline.NewSettler(offlineVerifier, vf)

	paymentSvc := service.NewPayment(repo, idemStore, vf)
	h := handler.NewREST(paymentSvc, vf, cfg.WebhookSecret, offlineSettler)

	srv, err := server.New(cfg, paymentSvc, h, health)
	if err != nil {
		return err
	}

	relay := outbox.NewRelay(db, bus, events.Subject)
	relayCtx, relayCancel := context.WithCancel(ctx)
	defer relayCancel()
	relayErrors := make(chan error, 1)
	go func() {
		relayErrors <- relay.Run(relayCtx)
	}()

	serverErrors, err := srv.Start(ctx)
	if err != nil {
		return err
	}

	observability.Info(ctx, "payment-orchestrator started",
		slog.String("grpc_port", cfg.GRPCPort),
		slog.String("rest_port", cfg.Port),
	)

	select {
	case err := <-serverErrors:
		return err
	case err := <-relayErrors:
		return err
	case <-ctx.Done():
		observability.Info(ctx, "payment-orchestrator: shutdown signal received, draining...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		relayCancel()
		return srv.Shutdown(shutdownCtx)
	}
}
