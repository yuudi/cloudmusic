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
	"time"

	"github.com/yuudi/cloudmusic/apps/server/internal/auth"

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
	pool, err := database.Open(startup, cfg.Database)
	cancel()
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
	authService := &auth.Service{DB: pool}
	go func() {
		runCleanup := func() {
			cleanupCtx, cancel := context.WithTimeout(ctx, time.Minute)
			defer cancel()
			if err := authService.Cleanup(cleanupCtx); err != nil && ctx.Err() == nil {
				log.Error("account cleanup failed", "error", err)
			}
		}
		runCleanup()
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				runCleanup()
			}
		}
	}()
	router := httpserver.New(log, cfg.DependencyTimeout, pool.PingContext, func(ctx context.Context) error { return redis.Ping(ctx).Err() }, authService)
	if err := router.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		return fmt.Errorf("trusted proxies: %w", err)
	}
	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
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
