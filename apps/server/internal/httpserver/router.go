package httpserver

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type Pinger func(context.Context) error

func New(log *slog.Logger, timeout time.Duration, postgres, redis Pinger) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	// Proxy trust must be configured explicitly when a reverse proxy is added.
	_ = router.SetTrustedProxies(nil)
	router.Use(requestLog(log), recovery(log))
	router.HandleMethodNotAllowed = true
	router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
	})
	router.NoMethod(func(c *gin.Context) {
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "method_not_allowed"})
	})
	api := router.Group("/api/v1")
	api.GET("/health/live", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	api.GET("/health/ready", func(c *gin.Context) {
		// Both checks share one deadline, bounding the complete probe duration.
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()
		checks := make(map[string]string, 2)
		status := http.StatusOK
		for _, dependency := range []struct {
			name string
			ping Pinger
		}{{"postgres", postgres}, {"redis", redis}} {
			checks[dependency.name] = "ok"
			if err := dependency.ping(ctx); err != nil {
				checks[dependency.name] = "unavailable"
				status = http.StatusServiceUnavailable
				log.WarnContext(ctx, "dependency health check failed", "dependency", dependency.name, "error", err)
			}
		}
		state := "ok"
		if status != http.StatusOK {
			state = "unavailable"
		}
		c.JSON(status, gin.H{"status": state, "checks": checks})
	})
	return router
}

func requestLog(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		// Omit query strings, request bodies and headers, which may carry secrets.
		log.InfoContext(c.Request.Context(), "http request",
			"method", c.Request.Method, "path", c.Request.URL.Path,
			"status", c.Writer.Status(), "duration", time.Since(start))
	}
}

func recovery(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if value := recover(); value != nil {
				log.ErrorContext(c.Request.Context(), "http panic", "error", fmt.Sprint(value))
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal_server_error"})
			}
		}()
		c.Next()
	}
}
