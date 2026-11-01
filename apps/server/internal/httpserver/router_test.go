package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestHealthChecks(t *testing.T) {
	for _, test := range []struct {
		name     string
		path     string
		failed   string
		wantCode int
		calls    int
	}{
		{"live while dependencies fail", "/api/v1/health/live", "both", http.StatusOK, 0},
		{"ready", "/api/v1/health/ready", "", http.StatusOK, 2},
		{"postgres unavailable", "/api/v1/health/ready", "postgres", http.StatusServiceUnavailable, 2},
		{"redis unavailable", "/api/v1/health/ready", "redis", http.StatusServiceUnavailable, 2},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			ping := func(name string) Pinger {
				return func(context.Context) error {
					calls++
					if name == test.failed || test.failed == "both" {
						return errors.New("private connection details")
					}
					return nil
				}
			}
			router := New(slog.New(slog.NewJSONHandler(io.Discard, nil)), time.Second, ping("postgres"), ping("redis"))
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, test.path, nil))
			if response.Code != test.wantCode || calls != test.calls {
				t.Fatalf("status=%d calls=%d; want status=%d calls=%d", response.Code, calls, test.wantCode, test.calls)
			}
			var body struct {
				Status string            `json:"status"`
				Checks map[string]string `json:"checks"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if test.calls > 0 {
				for _, name := range []string{"postgres", "redis"} {
					want := "ok"
					if name == test.failed {
						want = "unavailable"
					}
					if body.Checks[name] != want {
						t.Fatalf("%s health=%q, want %q", name, body.Checks[name], want)
					}
				}
			}
			if strings.Contains(response.Body.String(), "private") {
				t.Fatal("health response exposes internal errors")
			}
		})
	}
}

func TestReadinessDeadline(t *testing.T) {
	ping := func(ctx context.Context) error {
		if _, ok := ctx.Deadline(); !ok {
			t.Fatal("dependency check has no deadline")
		}
		<-ctx.Done()
		return ctx.Err()
	}
	router := New(slog.New(slog.NewJSONHandler(io.Discard, nil)), time.Millisecond, ping, ping)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/health/ready", nil))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d, want 503", response.Code)
	}
}

func TestRecoveryAndRequestLogging(t *testing.T) {
	var logs bytes.Buffer
	ping := func(context.Context) error { return nil }
	router := New(slog.New(slog.NewJSONHandler(&logs, nil)), time.Second, ping, ping)
	router.GET("/panic", func(*gin.Context) { panic("handler failed") })
	for _, test := range []struct {
		method string
		path   string
		status int
	}{
		{http.MethodGet, "/panic?token=secret", http.StatusInternalServerError},
		{http.MethodGet, "/api/v1/missing", http.StatusNotFound},
		{http.MethodPost, "/api/v1/health/live", http.StatusMethodNotAllowed},
	} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(test.method, test.path, nil))
		if response.Code != test.status || !json.Valid(response.Body.Bytes()) {
			t.Fatalf("%s: status=%d body=%s", test.path, response.Code, response.Body.String())
		}
	}
	if strings.Contains(logs.String(), "secret") || !strings.Contains(logs.String(), "http panic") || !strings.Contains(logs.String(), `"status":500`) {
		t.Fatalf("unexpected request logs: %s", logs.String())
	}
}
