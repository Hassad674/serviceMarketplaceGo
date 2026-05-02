package redis

import (
	"context"
	"fmt"
	"log/slog"

	goredis "github.com/redis/go-redis/v9"

	"marketplace-backend/internal/observability"
)

func NewClient(redisURL string) (*goredis.Client, error) {
	opts, err := goredis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}

	// Pool tuning — sessions are read on every authenticated request
	opts.PoolSize = 50
	opts.MinIdleConns = 10
	opts.MaxRetries = 3

	client := goredis.NewClient(opts)

	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("redis ping: %w", err)
	}

	// Attach OTel tracing hooks. When OTel is disabled the global
	// tracer is the SDK no-op so the hooks resolve to non-recording
	// spans. Tracing failure is non-fatal — Redis operations still
	// run, only observability is lost.
	if err := observability.InstrumentRedis(client); err != nil {
		slog.Warn("redis: otel instrumentation failed", "error", err)
	}

	return client, nil
}
