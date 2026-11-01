package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

func Open(ctx context.Context, connectionURL string, timeout time.Duration) (*redis.Client, error) {
	options, err := redis.ParseURL(connectionURL)
	if err != nil {
		return nil, fmt.Errorf("invalid REDIS_URL")
	}
	options.DialTimeout = timeout
	options.ReadTimeout = timeout
	options.WriteTimeout = timeout
	options.ContextTimeoutEnabled = true
	client := redis.NewClient(options)
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("ping redis: %w", err)
	}
	return client, nil
}
