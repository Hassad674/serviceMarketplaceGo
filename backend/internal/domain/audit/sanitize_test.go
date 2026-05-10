package audit

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// hashOf re-implements the prefix logic the production code uses so
// the tests pin the expected output without re-using the function
// under test. If `SanitizeMetadata` ever changes algorithm, this
// helper has to change too — that is intentional, the test should
// fail loudly if the on-disk format moves.
func hashOf(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])[:16]
}

func TestSanitizeMetadata_NilSafe(t *testing.T) {
	t.Parallel()

	got := SanitizeMetadata(nil)
	assert.Nil(t, got, "nil input must return nil output without allocating")
}

func TestSanitizeMetadata_EmptyMap(t *testing.T) {
	t.Parallel()

	got := SanitizeMetadata(map[string]any{})
	assert.NotNil(t, got, "empty input must still return a non-nil map (callers may type-assert)")
	assert.Empty(t, got)
}

func TestSanitizeMetadata_DoesNotMutateInput(t *testing.T) {
	t.Parallel()

	in := map[string]any{
		"email":  "alice@example.com",
		"reason": "invalid_password",
	}

	_ = SanitizeMetadata(in)

	assert.Equal(t, "alice@example.com", in["email"], "input map must not be mutated")
	assert.Equal(t, "invalid_password", in["reason"], "input map must not be mutated")
}

func TestSanitizeMetadata_SensitiveKeys(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		key  string
		val  string
	}{
		{name: "email replaced", key: "email", val: "alice@example.com"},
		{name: "to_email replaced", key: "to_email", val: "bob@corp.io"},
		{name: "from_email replaced", key: "from_email", val: "noreply@srv.io"},
		{name: "recipient replaced", key: "recipient", val: "carol@ops.io"},
		{name: "phone replaced", key: "phone", val: "+33612345678"},
		{name: "iban replaced", key: "iban", val: "FR7630006000011234567890189"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := SanitizeMetadata(map[string]any{tt.key: tt.val})

			require.Contains(t, got, tt.key)
			assert.Equal(t, hashOf(tt.val), got[tt.key], "sensitive value must be replaced by 16-hex SHA-256 prefix")
			assert.Len(t, got[tt.key].(string), 16, "hash prefix must be exactly 16 hex chars")
			assert.NotEqual(t, tt.val, got[tt.key], "cleartext must NEVER appear in sanitized output")
		})
	}
}

func TestSanitizeMetadata_NonSensitiveKeysUntouched(t *testing.T) {
	t.Parallel()

	in := map[string]any{
		"reason":           "invalid_password",
		"user_agent":       "curl/8.5.0",
		"attempted_action": "update",
		"old_status":       "draft",
		"new_status":       "published",
		"count":            42,
		"flag":             true,
	}

	got := SanitizeMetadata(in)

	assert.Equal(t, "invalid_password", got["reason"])
	assert.Equal(t, "curl/8.5.0", got["user_agent"])
	assert.Equal(t, "update", got["attempted_action"])
	assert.Equal(t, "draft", got["old_status"])
	assert.Equal(t, "published", got["new_status"])
	assert.Equal(t, 42, got["count"])
	assert.Equal(t, true, got["flag"])
}

func TestSanitizeMetadata_NestedMap(t *testing.T) {
	t.Parallel()

	in := map[string]any{
		"reason": "duplicate_account",
		"actor": map[string]any{
			"email":  "mallory@evil.io",
			"reason": "throttle_bypass",
			"deeper": map[string]any{
				"phone": "+33700000000",
				"label": "burner",
			},
		},
	}

	got := SanitizeMetadata(in)

	assert.Equal(t, "duplicate_account", got["reason"])

	actor, ok := got["actor"].(map[string]any)
	require.True(t, ok, "nested map must be preserved as map[string]any")
	assert.Equal(t, hashOf("mallory@evil.io"), actor["email"])
	assert.Equal(t, "throttle_bypass", actor["reason"])

	deeper, ok := actor["deeper"].(map[string]any)
	require.True(t, ok, "doubly-nested map must keep being walked")
	assert.Equal(t, hashOf("+33700000000"), deeper["phone"])
	assert.Equal(t, "burner", deeper["label"])
}

func TestSanitizeMetadata_NonStringSensitiveValueIsStringified(t *testing.T) {
	t.Parallel()

	// `email` arriving as a typed alias / wrapper / []byte should still be
	// hashed. We assert the helper is total — never panics on non-strings.
	tests := []struct {
		name string
		val  any
		want string
	}{
		{name: "byte slice", val: []byte("alice@example.com"), want: hashOf("alice@example.com")},
		{name: "int", val: 42, want: hashOf("42")},
		{name: "stringer-friendly bool", val: true, want: hashOf("true")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := SanitizeMetadata(map[string]any{"email": tt.val})

			assert.Equal(t, tt.want, got["email"])
		})
	}
}

func TestSanitizeMetadata_NilSensitiveValueStaysEmpty(t *testing.T) {
	t.Parallel()

	got := SanitizeMetadata(map[string]any{"email": nil})

	require.Contains(t, got, "email")
	assert.Equal(t, "", got["email"], "nil sensitive value must become empty string, not the literal 'nil'")
}

func TestSanitizeMetadata_DeterministicHash(t *testing.T) {
	t.Parallel()

	a := SanitizeMetadata(map[string]any{"email": "alice@example.com"})
	b := SanitizeMetadata(map[string]any{"email": "alice@example.com"})

	assert.Equal(t, a["email"], b["email"], "same input must produce same hash — admins rely on this to match rows about the same actor")
}

func TestSanitizeMetadata_DifferentInputsDifferentHashes(t *testing.T) {
	t.Parallel()

	a := SanitizeMetadata(map[string]any{"email": "alice@example.com"})
	b := SanitizeMetadata(map[string]any{"email": "bob@example.com"})

	assert.NotEqual(t, a["email"], b["email"])
}

func TestSanitizeMetadata_LoginFailureRealisticPayload(t *testing.T) {
	t.Parallel()

	// Mirrors the metadata shape produced by app/auth/service.go on
	// auth.login_failure events — locks down the contract so a future
	// refactor that adds another sensitive key can extend the redacted
	// set without surprising downstream consumers.
	in := map[string]any{
		"email":      "user@example.com",
		"reason":     "invalid_password",
		"user_agent": "Mozilla/5.0",
	}

	got := SanitizeMetadata(in)

	assert.Equal(t, hashOf("user@example.com"), got["email"])
	assert.Equal(t, "invalid_password", got["reason"])
	assert.Equal(t, "Mozilla/5.0", got["user_agent"])
}
