package config

import (
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Database struct {
	URL             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

type Config struct {
	Database          Database
	RedisURL          string
	HTTPAddr          string
	TrustedProxies    []string
	LogLevel          slog.Level
	DependencyTimeout time.Duration
	ReadHeaderTimeout time.Duration
	IdleTimeout       time.Duration
	ShutdownTimeout   time.Duration
}

func Load() (Config, error) {
	var cfg Config
	var err error
	if cfg.Database, err = LoadDatabase(); err != nil {
		return cfg, err
	}
	if cfg.RedisURL, err = connectionURL("REDIS_URL", "redis", "rediss"); err != nil {
		return cfg, err
	}
	cfg.HTTPAddr = env("HTTP_ADDR", ":8080")
	_, port, err := net.SplitHostPort(cfg.HTTPAddr)
	if err != nil {
		return cfg, fmt.Errorf("HTTP_ADDR must be a host:port address")
	}
	if n, err := strconv.Atoi(port); err != nil || n < 1 || n > 65535 {
		return cfg, fmt.Errorf("HTTP_ADDR port must be between 1 and 65535")
	}
	if raw := os.Getenv("TRUSTED_PROXIES"); raw != "" {
		for _, value := range strings.Split(raw, ",") {
			value = strings.TrimSpace(value)
			if net.ParseIP(value) == nil {
				if _, _, err := net.ParseCIDR(value); err != nil {
					return cfg, fmt.Errorf("TRUSTED_PROXIES must contain IP addresses or CIDR ranges")
				}
			}
			cfg.TrustedProxies = append(cfg.TrustedProxies, value)
		}
	}
	if err = cfg.LogLevel.UnmarshalText([]byte(env("LOG_LEVEL", "info"))); err != nil {
		return cfg, fmt.Errorf("invalid LOG_LEVEL")
	}
	for _, item := range []struct {
		name     string
		fallback string
		target   *time.Duration
	}{
		{"DEPENDENCY_TIMEOUT", "3s", &cfg.DependencyTimeout},
		{"HTTP_READ_HEADER_TIMEOUT", "5s", &cfg.ReadHeaderTimeout},
		{"HTTP_IDLE_TIMEOUT", "60s", &cfg.IdleTimeout},
		{"SHUTDOWN_TIMEOUT", "10s", &cfg.ShutdownTimeout},
	} {
		if *item.target, err = duration(item.name, item.fallback); err != nil {
			return cfg, err
		}
	}
	return cfg, nil
}

// LoadDatabase also serves the migration command, which does not require Redis.
func LoadDatabase() (Database, error) {
	var cfg Database
	var err error
	if cfg.URL, err = connectionURL("DATABASE_URL", "postgres", "postgresql"); err != nil {
		return cfg, err
	}
	if cfg.MaxOpenConns, err = integer("DB_MAX_OPEN_CONNS", "10", 1); err != nil {
		return cfg, err
	}
	if cfg.MaxIdleConns, err = integer("DB_MAX_IDLE_CONNS", "5", 0); err != nil {
		return cfg, err
	}
	if cfg.MaxIdleConns > cfg.MaxOpenConns {
		return cfg, fmt.Errorf("DB_MAX_IDLE_CONNS must not exceed DB_MAX_OPEN_CONNS")
	}
	cfg.ConnMaxLifetime, err = duration("DB_CONN_MAX_LIFETIME", "30m")
	return cfg, err
}

func env(name, fallback string) string {
	if value, ok := os.LookupEnv(name); ok {
		return value
	}
	return fallback
}

func connectionURL(name string, schemes ...string) (string, error) {
	value := os.Getenv(name)
	u, err := url.Parse(value)
	if err == nil && u.Hostname() != "" {
		for _, scheme := range schemes {
			if u.Scheme == scheme {
				return value, nil
			}
		}
	}
	// Never include connection strings or credentials in configuration errors.
	return "", fmt.Errorf("%s must be a valid %v connection URL", name, schemes)
}

func integer(name, fallback string, min int) (int, error) {
	value, err := strconv.Atoi(env(name, fallback))
	if err != nil || value < min {
		return 0, fmt.Errorf("%s must be an integer >= %d", name, min)
	}
	return value, nil
}

func duration(name, fallback string) (time.Duration, error) {
	value, err := time.ParseDuration(env(name, fallback))
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("%s must be a positive duration", name)
	}
	return value, nil
}
