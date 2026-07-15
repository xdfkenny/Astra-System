package testutil

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// NewPostgresContainer starts a PostgreSQL container and applies migrations.
// It skips the test if Docker is not available.
func NewPostgresContainer(t *testing.T) (*sql.DB, func()) {
	t.Helper()
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not installed")
	}
	ctx := context.Background()
	dockerCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := exec.CommandContext(dockerCtx, "docker", "info").Run(); err != nil {
		t.Skip("docker not available")
	}

	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("astra"),
		postgres.WithUsername("astra"),
		postgres.WithPassword("astra"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}

	host, err := pgContainer.Host(ctx)
	if err != nil {
		_ = pgContainer.Terminate(ctx)
		t.Fatalf("postgres host: %v", err)
	}
	port, err := pgContainer.MappedPort(ctx, "5432")
	if err != nil {
		_ = pgContainer.Terminate(ctx)
		t.Fatalf("postgres port: %v", err)
	}

	dsn := fmt.Sprintf("postgresql://astra:astra@%s:%s/astra?sslmode=disable", host, port.Port())
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		_ = pgContainer.Terminate(ctx)
		t.Fatalf("open db: %v", err)
	}
	db.SetMaxOpenConns(10)

	if err := applyMigrations(db); err != nil {
		_ = db.Close()
		_ = pgContainer.Terminate(ctx)
		t.Fatalf("apply migrations: %v", err)
	}

	cleanup := func() {
		_ = db.Close()
		_ = pgContainer.Terminate(ctx)
	}
	return db, cleanup
}

func applyMigrations(db *sql.DB) error {
	_, b, _, _ := runtime.Caller(0)
	base := filepath.Join(filepath.Dir(b), "..", "..", "..", "..", "..", "database", "migrations")
	files := []string{"0001_init.sql", "0002_outbox_relay.sql"}
	for _, f := range files {
		data, err := os.ReadFile(filepath.Join(base, f))
		if err != nil {
			return fmt.Errorf("read migration %s: %w", f, err)
		}
		if _, err := db.Exec(migrationUp(string(data))); err != nil {
			return fmt.Errorf("exec migration %s: %w", f, err)
		}
	}
	return nil
}

func migrationUp(sqlText string) string {
	if idx := strings.Index(strings.ToLower(sqlText), "-- down"); idx >= 0 {
		return sqlText[:idx]
	}
	return sqlText
}
