// Command payment-service runs the Astra-Service payment orchestrator.
// It records payment authorizations, verifies offline payment tokens, and
// settles queued offline tokens with the payment processor.
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os/signal"
	"syscall"
	"time"

	"github.com/astra-service/go-common/eventbus"
	"github.com/astra-service/go-common/observability"
	"github.com/astra-service/payment-service/internal/config"
	"github.com/astra-service/payment-service/internal/handler"
	"github.com/astra-service/payment-service/internal/repository"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("payment-service: fatal startup error: %v", err)
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
		ServiceName:    "astra-payment-service",
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
		return fmt.Errorf("payment-service: open db: %w", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)

	bus, err := eventbus.Connect(ctx, cfg.NatsURL)
	if err != nil {
		return fmt.Errorf("payment-service: connect nats: %w", err)
	}
	defer bus.Close()

	repo := repository.NewPaymentRepository(db, cfg.OfflineTokenHMAC)
	h := handler.NewPaymentHandler(repo, bus, cfg.SettlementEnabled, cfg.SettlementInterval)

	// Background settlement loop for offline tokens.
	go h.RunSettlementLoop(ctx)

	consumeAuth, err := bus.Subscribe(ctx, "ASTRA_PAYMENT", "payment-service-record-auth",
		"astra.payment.record_authorization", h.HandleRecordAuthorization)
	if err != nil {
		return fmt.Errorf("payment-service: subscribe record-auth: %w", err)
	}
	defer consumeAuth.Stop()

	consumeOffline, err := bus.Subscribe(ctx, "ASTRA_PAYMENT", "payment-service-offline-token",
		"astra.payment.offline_token_received", h.HandleOfflineToken)
	if err != nil {
		return fmt.Errorf("payment-service: subscribe offline-token: %w", err)
	}
	defer consumeOffline.Stop()

	observability.Info(ctx, "payment-service started, awaiting payment commands")
	<-ctx.Done()
	observability.Info(ctx, "payment-service: shutdown signal received, draining...")
	return nil
}
