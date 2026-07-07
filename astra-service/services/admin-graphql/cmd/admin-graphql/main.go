// Command admin-graphql runs the admin-only GraphQL API.
package main

import (
	"context"
	"database/sql"
	"log"
	"os/signal"
	"syscall"
	"time"

	"github.com/astra-service/go-common/observability"
	"github.com/astra-systems/astra-service/services/admin-graphql/internal/config"
	"github.com/astra-systems/astra-service/services/admin-graphql/internal/repository"
	"github.com/astra-systems/astra-service/services/admin-graphql/internal/resolver"
	"github.com/astra-systems/astra-service/services/admin-graphql/internal/server"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("admin-graphql: fatal error: %v", err)
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
		ServiceName:    cfg.ServiceName,
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

	repo := repository.NewPostgresRepository(db)
	schema, err := resolver.NewSchema(repo)
	if err != nil {
		return err
	}

	srv := server.New(cfg, schema)
	return srv.Start(ctx)
}
