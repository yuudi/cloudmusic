package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/yuudi/cloudmusic/apps/server/internal/cache"
	"github.com/yuudi/cloudmusic/apps/server/internal/config"
	"github.com/yuudi/cloudmusic/apps/server/internal/database"
	"github.com/yuudi/cloudmusic/apps/server/internal/httpserver"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg, err := config.Load()
	if err != nil {
		log.Error("invalid configuration", "error", err)
		os.Exit(1)
	}
	log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, cfg, log); err != nil {
		log.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, cfg config.Config, log *slog.Logger) error {
	startup, cancel := context.WithTimeout(ctx, cfg.DependencyTimeout)
	db, err := database.Open(startup, cfg.Database, log)
	cancel()
	if err != nil {
		return err
	}
	pool, err := db.DB()
	if err != nil {
		return err
	}
	defer pool.Close()
	startup, cancel = context.WithTimeout(ctx, cfg.DependencyTimeout)
	redis, err := cache.Open(startup, cfg.RedisURL, cfg.DependencyTimeout)
	cancel()
	if err != nil {
		return err
	}
	defer redis.Close()
	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           httpserver.New(log, cfg.DependencyTimeout, pool.PingContext, func(ctx context.Context) error { return redis.Ping(ctx).Err() }),
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		IdleTimeout:       cfg.IdleTimeout,
		ErrorLog:          slog.NewLogLogger(log.Handler(), slog.LevelError),
	}
	errorsCh := make(chan error, 1)
	go func() { errorsCh <- server.ListenAndServe() }()
	log.Info("server starting", "address", cfg.HTTPAddr)
	select {
	case err := <-errorsCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("serve http: %w", err)
	case <-ctx.Done():
		log.Info("server shutting down")
	}
	shutdown, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := server.Shutdown(shutdown); err != nil {
		_ = server.Close()
		return fmt.Errorf("shutdown http: %w", err)
	}
	return nil
}
