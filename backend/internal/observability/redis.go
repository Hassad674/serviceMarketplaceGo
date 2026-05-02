package observability

import (
	"fmt"

	"github.com/redis/go-redis/extra/redisotel/v9"
	goredis "github.com/redis/go-redis/v9"
)

// InstrumentRedis attaches OTel tracing hooks to a *redis.Client.
// Every Redis command becomes one span when OTel is enabled. When
// disabled (no global TracerProvider beyond the no-op) the hook
// resolves to a non-recording span — same zero-overhead promise as
// the rest of the package.
//
// Span attributes recorded by redisotel:
//   - db.system = "redis"
//   - db.connection_string (host:port)
//   - db.statement = the redis command (e.g. "GET sessions:abc")
//
// Span attributes NOT recorded:
//   - command argument values that contain PII (token contents are
//     stored as the cache value, not the cache key — keys are typed
//     prefixes like "session_id:<uuid>" and safe to record).
//
// The function is a no-op when rdb is nil so callers can defer the
// nil check.
func InstrumentRedis(rdb *goredis.Client) error {
	if rdb == nil {
		return nil
	}
	if err := redisotel.InstrumentTracing(rdb); err != nil {
		return fmt.Errorf("redisotel: %w", err)
	}
	return nil
}
