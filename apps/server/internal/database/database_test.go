package database

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/yuudi/cloudmusic/apps/server/internal/config"
)

func TestOpenRejectsInvalidURLWithoutCredentials(t *testing.T) {
	db, err := Open(context.Background(), config.Database{
		URL: "postgres://user:secret%zz@localhost/cloudmusic_dev",
	})
	if db != nil {
		db.Close()
		t.Fatal("invalid URL returned a database pool")
	}
	if err == nil || !strings.Contains(err.Error(), "DATABASE_URL") || strings.Contains(err.Error(), "secret") {
		t.Fatalf("expected a configuration error without credentials, got %v", err)
	}
}

func TestOpenHonorsCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	db, err := Open(ctx, config.Database{
		URL: "postgres://cloudmusic@localhost/cloudmusic_dev?sslmode=disable",
	})
	if db != nil {
		db.Close()
		t.Fatal("canceled startup returned a database pool")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}
}

func TestOpenPostgres(t *testing.T) {
	connectionURL := os.Getenv("TEST_DATABASE_URL")
	if connectionURL == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	db, err := Open(ctx, config.Database{
		URL: connectionURL, MaxOpenConns: 2, MaxIdleConns: 1, ConnMaxLifetime: time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if got := db.Stats().MaxOpenConnections; got != 2 {
		t.Fatalf("max open connections=%d, want 2", got)
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `CREATE TEMP TABLE connection_test (value text) ON COMMIT DROP`); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO connection_test VALUES ($1)`, "音乐 🎵"); err != nil {
		t.Fatal(err)
	}
	var value string
	if err = tx.QueryRowContext(ctx, `SELECT value FROM connection_test`).Scan(&value); err != nil || value != "音乐 🎵" {
		t.Fatalf("transaction read: value=%q error=%v", value, err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
}
