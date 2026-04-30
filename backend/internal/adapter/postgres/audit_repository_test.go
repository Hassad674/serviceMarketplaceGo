package postgres

// Unit tests for the metadata-parsing path of audit_repository.
//
// These tests live in `package postgres` (not `postgres_test`) so they
// can exercise the unexported parseAuditMetadata helper and the
// auditMetadataCorruptKey sentinel without going through the live DB
// adapter — closing BUG-20 with a fast, deterministic regression test.

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captureLogs swaps slog.Default for one writing into a buffer so
// tests can assert on the WARN line emitted by parseAuditMetadata.
func captureLogs(t *testing.T) (*bytes.Buffer, func()) {
	t.Helper()
	prev := slog.Default()
	buf := &bytes.Buffer{}
	slog.SetDefault(slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	return buf, func() { slog.SetDefault(prev) }
}

func TestParseAuditMetadata_NilBytesReturnsEmptyMap(t *testing.T) {
	got := parseAuditMetadata(uuid.New(), nil)

	assert.NotNil(t, got, "metadata must never be nil")
	assert.Empty(t, got)
}

func TestParseAuditMetadata_EmptyBytesReturnsEmptyMap(t *testing.T) {
	got := parseAuditMetadata(uuid.New(), []byte{})

	assert.NotNil(t, got)
	assert.Empty(t, got)
}

func TestParseAuditMetadata_ValidJSONIsDecoded(t *testing.T) {
	raw := []byte(`{"action":"login_success","ip":"1.2.3.4","attempt":3}`)
	got := parseAuditMetadata(uuid.New(), raw)

	assert.Equal(t, "login_success", got["action"])
	assert.Equal(t, "1.2.3.4", got["ip"])
	// JSON numbers decode to float64 by default.
	assert.Equal(t, float64(3), got["attempt"])
	// No corrupt sentinel for valid JSON.
	_, corrupt := got[auditMetadataCorruptKey]
	assert.False(t, corrupt)
}

func TestParseAuditMetadata_ValidEmptyObjectIsAllowed(t *testing.T) {
	raw := []byte(`{}`)
	got := parseAuditMetadata(uuid.New(), raw)

	assert.NotNil(t, got)
	assert.Empty(t, got)
}

// BUG-20 — corrupt metadata used to be silently swallowed. Now:
//  1. WARN logged with audit_entry_id and metadata_size,
//  2. returned map is tagged with auditMetadataCorruptKey so admin UIs
//     can flag the row.
func TestParseAuditMetadata_CorruptJSONLogsAndSentinels(t *testing.T) {
	logs, restore := captureLogs(t)
	defer restore()

	id := uuid.New()
	raw := []byte(`{this is not valid json`)

	got := parseAuditMetadata(id, raw)

	// Sentinel returned.
	require.NotNil(t, got)
	val, ok := got[auditMetadataCorruptKey]
	assert.True(t, ok, "corrupt metadata must carry the sentinel key")
	assert.IsType(t, "", val)
	assert.NotEmpty(t, val.(string))

	// WARN was emitted with structured fields.
	out := logs.String()
	assert.Contains(t, out, "audit: metadata unmarshal failed")
	assert.Contains(t, out, id.String())
	assert.Contains(t, out, "metadata_size=")
}

func TestParseAuditMetadata_NonObjectJSONIsCorrupt(t *testing.T) {
	logs, restore := captureLogs(t)
	defer restore()

	// Well-formed JSON but a string, not a map. Unmarshal into map[string]any
	// fails — that's exactly the kind of corruption the sentinel exists for.
	raw := []byte(`"i am a string"`)
	got := parseAuditMetadata(uuid.New(), raw)

	_, corrupt := got[auditMetadataCorruptKey]
	assert.True(t, corrupt)
	assert.Contains(t, logs.String(), "audit: metadata unmarshal failed")
}

func TestParseAuditMetadata_NullJSONReturnsEmptyMap(t *testing.T) {
	// Postgres can store the literal JSON "null" — that's a successful
	// Unmarshal returning a nil map. The contract says Metadata must
	// always be non-nil, so we get the empty map.
	raw := []byte(`null`)
	got := parseAuditMetadata(uuid.New(), raw)

	assert.NotNil(t, got)
	assert.Empty(t, got)
}

func TestParseAuditMetadata_LargePayload_DoesNotLoopForever(t *testing.T) {
	// Defensive: a very long invalid JSON should still return promptly
	// — json.Unmarshal is bounded. The test fails by timeout rather
	// than by assertion if a regression introduces a quadratic path.
	raw := []byte("{" + strings.Repeat("a", 10_000))

	got := parseAuditMetadata(uuid.New(), raw)
	_, corrupt := got[auditMetadataCorruptKey]
	assert.True(t, corrupt)
}
