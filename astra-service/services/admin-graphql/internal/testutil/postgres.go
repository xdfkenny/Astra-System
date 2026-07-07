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

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// NewPostgresContainer starts a PostgreSQL container and applies migrations.
func NewPostgresContainer(t *testing.T) (*sql.DB, func()) {
	t.Helper()

	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not available")
	}
	if os.Getenv("CI") != "" && os.Getenv("SKIP_TESTCONTAINERS") != "" {
		t.Skip("testcontainers disabled")
	}

	defer func() {
		if r := recover(); r != nil {
			t.Skipf("testcontainers unavailable: %v", r)
		}
	}()

	ctx := context.Background()

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
		t.Skipf("testcontainers unavailable: %v", err)
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
	files := []string{"0001_init.sql"}
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
