package database

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/yuudi/cloudmusic/apps/server/internal/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Open(ctx context.Context, cfg config.Database, log *slog.Logger) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(cfg.URL), &gorm.Config{
		DisableAutomaticPing: true,
		Logger: logger.New(slog.NewLogLogger(log.Handler(), slog.LevelWarn), logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logger.Warn,
			IgnoreRecordNotFoundError: true,
			ParameterizedQueries:      true,
		}),
	})
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}
	pool, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get postgres pool: %w", err)
	}
	pool.SetMaxOpenConns(cfg.MaxOpenConns)
	pool.SetMaxIdleConns(cfg.MaxIdleConns)
	pool.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	if err := pool.PingContext(ctx); err != nil {
		_ = pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	return db, nil
}
