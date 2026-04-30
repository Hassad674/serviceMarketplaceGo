package redis

// Unit tests for the pure helpers inside admin_notifier.go that do not
// require a Redis or PostgreSQL connection.
//
// `notifKey` is the shared key formatter — every Redis op derives its
// key from this one helper, so a regression here would silently route
// reads + writes to different keys.
//
// `parseRedisInt` is the defensive value parser invoked by GetAll. Its
// `err != nil` branch was the security audit's WARN-on-corruption
// landing site (the slog.Warn line in admin_notifier.go); we exercise
// it explicitly here so a regression that drops the WARN gets caught
// at unit-test time.

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNotifKey_ProducesNamespacedKey(t *testing.T) {
	id := uuid.MustParse("11111111-2222-3333-4444-555555555555")

	got := notifKey(id, "messages")

	assert.Equal(t, "admin:notif:11111111-2222-3333-4444-555555555555:messages", got)
}

func TestNotifKey_DifferentCategoriesProduceDifferentKeys(t *testing.T) {
	id := uuid.New()

	a := notifKey(id, "messages")
	b := notifKey(id, "billing")

	assert.NotEqual(t, a, b, "category boundary MUST split the key — same admin can have several counters")
}

func TestNotifKey_DifferentAdminsProduceDifferentKeys(t *testing.T) {
	cat := "messages"
	a := notifKey(uuid.New(), cat)
	b := notifKey(uuid.New(), cat)
	assert.NotEqual(t, a, b)
}

func TestParseRedisInt_NilReturnsZero(t *testing.T) {
	got := parseRedisInt(nil)
	assert.Equal(t, int64(0), got, "missing counter must surface as 0, never a panic")
}

func TestParseRedisInt_NonStringInputReturnsZero(t *testing.T) {
	// MGet results are always strings or nil, but defensive code is
	// cheap — if a future Redis adapter starts returning, say, an
	// `int`, the parser must surface 0 rather than panic.
	got := parseRedisInt(123)
	assert.Equal(t, int64(0), got)
}

func TestParseRedisInt_ValidIntegerString(t *testing.T) {
	got := parseRedisInt("42")
	assert.Equal(t, int64(42), got)
}

func TestParseRedisInt_NegativeIntegerString(t *testing.T) {
	got := parseRedisInt("-7")
	assert.Equal(t, int64(-7), got)
}

func TestParseRedisInt_LargeIntegerString(t *testing.T) {
	got := parseRedisInt("9223372036854775807") // max int64
	assert.Equal(t, int64(9223372036854775807), got)
}

func TestParseRedisInt_CorruptValueLogsWarnAndReturnsZero(t *testing.T) {
	// SEC-26 (Sscanf failure WARN): a non-integer string in a counter
	// slot must surface as 0 AND emit a slog.Warn so the operator can
	// detect the corruption without re-querying Redis.
	prev := slog.Default()
	defer slog.SetDefault(prev)

	buf := &bytes.Buffer{}
	slog.SetDefault(slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))

	got := parseRedisInt("not-a-number")

	assert.Equal(t, int64(0), got, "corrupt counter value MUST return 0")
	out := buf.String()
	assert.Contains(t, out, "admin notifier: parse counter value failed",
		"WARN must surface the corruption — silent swallowing was the pre-fix bug")
	assert.Contains(t, out, "not-a-number",
		"the offending value must appear in the log so on-call can correlate")
}

func TestParseRedisInt_EmptyStringIsCorrupt(t *testing.T) {
	prev := slog.Default()
	defer slog.SetDefault(prev)

	buf := &bytes.Buffer{}
	slog.SetDefault(slog.New(slog.NewTextHandler(buf, nil)))

	got := parseRedisInt("")

	assert.Equal(t, int64(0), got)
	assert.Contains(t, buf.String(), "admin notifier: parse counter value failed")
}
