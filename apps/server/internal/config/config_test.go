package config

import (
	"strings"
	"testing"
	"time"
)

func configure(t *testing.T) {
	t.Helper()
	for key, value := range map[string]string{
		"DATABASE_URL": "postgres://cloudmusic:secret@localhost:5432/cloudmusic_dev?sslmode=disable",
		"REDIS_URL":    "redis://localhost:6379/0",
		"HTTP_ADDR":    ":8080", "LOG_LEVEL": "info",
		"DB_MAX_OPEN_CONNS": "10", "DB_MAX_IDLE_CONNS": "5", "DB_CONN_MAX_LIFETIME": "30m",
		"DEPENDENCY_TIMEOUT": "3s", "HTTP_READ_HEADER_TIMEOUT": "5s",
		"HTTP_IDLE_TIMEOUT": "60s", "SHUTDOWN_TIMEOUT": "10s",
	} {
		t.Setenv(key, value)
	}
}

func TestLoad(t *testing.T) {
	configure(t)
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HTTPAddr != ":8080" || cfg.Database.MaxOpenConns != 10 || cfg.ShutdownTimeout != 10*time.Second {
		t.Fatalf("unexpected configuration: address=%s, max_open=%d, shutdown=%s", cfg.HTTPAddr, cfg.Database.MaxOpenConns, cfg.ShutdownTimeout)
	}
}

func TestRejectInvalidConfiguration(t *testing.T) {
	for _, test := range []struct{ name, value string }{
		{"DATABASE_URL", ""},
		{"DATABASE_URL", "mysql://user:secret@localhost/db"},
		{"DATABASE_URL", "postgres://user:secret%zz@localhost/db"},
		{"REDIS_URL", ""},
		{"REDIS_URL", "http://localhost"},
		{"HTTP_ADDR", "8080"},
		{"HTTP_ADDR", ":65536"},
		{"LOG_LEVEL", "invalid"},
		{"DB_MAX_OPEN_CONNS", "0"},
		{"DB_MAX_IDLE_CONNS", "11"},
		{"DB_MAX_IDLE_CONNS", "-1"},
		{"DB_CONN_MAX_LIFETIME", "0s"},
		{"DEPENDENCY_TIMEOUT", "-1s"},
		{"HTTP_READ_HEADER_TIMEOUT", "bad"},
		{"HTTP_IDLE_TIMEOUT", "0s"},
		{"SHUTDOWN_TIMEOUT", ""},
	} {
		t.Run(test.name+"="+test.value, func(t *testing.T) {
			configure(t)
			t.Setenv(test.name, test.value)
			_, err := Load()
			if err == nil || !strings.Contains(err.Error(), test.name) {
				t.Fatalf("expected error naming %s, got %v", test.name, err)
			}
			if strings.Contains(err.Error(), "secret") {
				t.Fatal("configuration error exposes credentials")
			}
		})
	}
}

func TestMigrationConfigDoesNotRequireRedis(t *testing.T) {
	configure(t)
	t.Setenv("REDIS_URL", "")
	if _, err := LoadDatabase(); err != nil {
		t.Fatal(err)
	}
}

func TestTrustedProxies(t *testing.T) {
	configure(t)
	t.Setenv("TRUSTED_PROXIES", " 192.0.2.1, 10.0.0.0/8, 2001:db8::/32 ")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.TrustedProxies) != 3 || cfg.TrustedProxies[0] != "192.0.2.1" || cfg.TrustedProxies[1] != "10.0.0.0/8" {
		t.Fatalf("unexpected proxies: %v", cfg.TrustedProxies)
	}
	for _, bad := range []string{"proxy.example", "10.0.0.0/33", "10.0.0.1,", " "} {
		t.Setenv("TRUSTED_PROXIES", bad)
		if _, err := Load(); err == nil {
			t.Fatalf("accepted %q", bad)
		}
	}
	t.Setenv("TRUSTED_PROXIES", "")
	cfg, err = Load()
	if err != nil || len(cfg.TrustedProxies) != 0 {
		t.Fatalf("default trusted proxies: %v %v", cfg.TrustedProxies, err)
	}
}
