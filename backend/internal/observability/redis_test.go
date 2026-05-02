package observability

import (
	"testing"

	goredis "github.com/redis/go-redis/v9"
)

// TestInstrumentRedis_NilClient is a no-op so callers can defer the
// nil check at construction time without crashing on environments
// where Redis is intentionally absent (e.g. unit tests).
func TestInstrumentRedis_NilClient(t *testing.T) {
	if err := InstrumentRedis(nil); err != nil {
		t.Errorf("InstrumentRedis(nil) = %v, want nil", err)
	}
}

// TestInstrumentRedis_AttachesHook attaches the redisotel hook to a
// client and asserts the call returns nil. We do not need a live
// Redis to verify the hook installation — the call only configures
// the client, it does not dial.
func TestInstrumentRedis_AttachesHook(t *testing.T) {
	client := goredis.NewClient(&goredis.Options{
		Addr: "localhost:0", // never dialled
	})
	defer client.Close()

	if err := InstrumentRedis(client); err != nil {
		t.Errorf("InstrumentRedis returned error: %v", err)
	}
}
