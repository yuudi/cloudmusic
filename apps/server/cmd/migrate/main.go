package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pressly/goose/v3"
	"github.com/yuudi/cloudmusic/apps/server/internal/config"
	"github.com/yuudi/cloudmusic/apps/server/internal/database"
	"github.com/yuudi/cloudmusic/apps/server/migrations"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, log); err != nil {
		log.Error("migration failed", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, log *slog.Logger) error {
	if len(os.Args) != 2 {
		return fmt.Errorf("usage: migrate up|down|status")
	}
	command := os.Args[1]
	if command != "up" && command != "down" && command != "status" {
		return fmt.Errorf("usage: migrate up|down|status")
	}
	cfg, err := config.LoadDatabase()
	if err != nil {
		return err
	}
	startup, cancel := context.WithTimeout(ctx, 5*time.Second)
	db, err := database.Open(startup, cfg, log)
	cancel()
	if err != nil {
		return err
	}
	pool, err := db.DB()
	if err != nil {
		return err
	}
	defer pool.Close()
	provider, err := goose.NewProvider(goose.DialectPostgres, pool, migrations.Files)
	if err != nil {
		return err
	}
	switch command {
	case "up":
		_, err = provider.Up(ctx)
	case "down":
		_, err = provider.Down(ctx)
	case "status":
		statuses, statusErr := provider.Status(ctx)
		if statusErr != nil {
			return statusErr
		}
		for _, status := range statuses {
			log.Info("migration status", "version", status.Source.Version, "state", status.State)
		}
	}
	if err == nil {
		log.Info("migration complete", "command", command)
	}
	return err
}
