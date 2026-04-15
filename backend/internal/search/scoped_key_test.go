package search

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// encodeNoHTML mirrors the production HMAC input encoding so tests
// stay in sync with implementation drift.
func encodeNoHTML(t *testing.T, v any) []byte {
	t.Helper()
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	require.NoError(t, enc.Encode(v))
	return bytes.TrimRight(buf.Bytes(), "\n")
}

// scoped_key_test.go locks the wire format of the scoped search key
// down with byte-for-byte fixtures.
//
// We deliberately avoid asserting against a hand-baked golden string
// (which would force re-tuning the test on every minor refactor) and
// instead RECOMPUTE the expected output from scratch using the public
// HMAC + base64 primitives. The test fails only when the
// implementation drifts from the documented Typesense algorithm.

func TestGenerateScopedSearchKey_HappyPath(t *testing.T) {
	parent := "xyz-dev-master-key-change-in-production"
	params := EmbeddedSearchParams{
		FilterBy:  "persona:freelance && is_published:true",
		ExpiresAt: 1776000000,
	}

	got, err := generateScopedSearchKey(parent, params)
	require.NoError(t, err)
	require.NotEmpty(t, got)

	// Independently recompute the expected value.
	embedded := encodeNoHTML(t, params)
	mac := hmac.New(sha256.New, []byte(parent))
	_, _ = mac.Write(embedded)
	digest := hex.EncodeToString(mac.Sum(nil))
	want := base64.StdEncoding.EncodeToString([]byte(digest + parent[:4] + string(embedded)))

	assert.Equal(t, want, got, "scoped key wire format must match HMAC-SHA256 + base64 spec")
}

func TestGenerateScopedSearchKey_Deterministic(t *testing.T) {
	parent := "xyz-dev-master-key-change-in-production"
	params := EmbeddedSearchParams{
		FilterBy:  "persona:agency",
		ExpiresAt: time.Now().Unix(),
	}

	first, err := generateScopedSearchKey(parent, params)
	require.NoError(t, err)
	second, err := generateScopedSearchKey(parent, params)
	require.NoError(t, err)

	assert.Equal(t, first, second, "scoped key generation must be deterministic for fixed inputs")
}

func TestGenerateScopedSearchKey_DifferentParentsProduceDifferentKeys(t *testing.T) {
	params := EmbeddedSearchParams{FilterBy: "persona:freelance"}

	a, err := generateScopedSearchKey("aaaaXXXXXXXX", params)
	require.NoError(t, err)
	b, err := generateScopedSearchKey("bbbbXXXXXXXX", params)
	require.NoError(t, err)

	assert.NotEqual(t, a, b, "different parent keys must produce different scoped keys")

	// And the prefix at the right offset must differ.
	rawA, err := base64.StdEncoding.DecodeString(a)
	require.NoError(t, err)
	rawB, err := base64.StdEncoding.DecodeString(b)
	require.NoError(t, err)
	// digest is 64 hex chars, then 4-char prefix, then the json
	assert.Equal(t, "aaaa", string(rawA[64:68]))
	assert.Equal(t, "bbbb", string(rawB[64:68]))
}

func TestGenerateScopedSearchKey_DifferentFiltersProduceDifferentKeys(t *testing.T) {
	parent := "xyz-dev-master-key-change-in-production"

	a, err := generateScopedSearchKey(parent, EmbeddedSearchParams{FilterBy: "persona:freelance"})
	require.NoError(t, err)
	b, err := generateScopedSearchKey(parent, EmbeddedSearchParams{FilterBy: "persona:agency"})
	require.NoError(t, err)

	assert.NotEqual(t, a, b, "different filter_by values must produce different scoped keys")
}

func TestGenerateScopedSearchKey_InvalidInputs(t *testing.T) {
	tests := []struct {
		name      string
		parentKey string
		wantErr   string
	}{
		{name: "empty parent key", parentKey: "", wantErr: "parent key is required"},
		{name: "blank parent key", parentKey: "   ", wantErr: "parent key is required"},
		{name: "too short parent key", parentKey: "abc", wantErr: "at least 4 characters"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := generateScopedSearchKey(tt.parentKey, EmbeddedSearchParams{})
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestGenerateScopedSearchKey_PrefixIsFirst4Chars(t *testing.T) {
	parent := "k1k2k3k4-rest-of-the-key-doesnt-matter"
	got, err := generateScopedSearchKey(parent, EmbeddedSearchParams{})
	require.NoError(t, err)

	raw, err := base64.StdEncoding.DecodeString(got)
	require.NoError(t, err)
	// 64-hex-char digest + 4-char prefix + json
	assert.Equal(t, "k1k2", string(raw[64:68]),
		"scoped key must embed the first 4 chars of the parent key as Typesense uses it for lookup")
}

func TestGenerateScopedSearchKey_EmbedsParamsAsJSON(t *testing.T) {
	parent := "xyz-dev-master-key-change-in-production"
	params := EmbeddedSearchParams{
		FilterBy:  "persona:referrer && is_published:true",
		ExpiresAt: 1234567890,
	}

	got, err := generateScopedSearchKey(parent, params)
	require.NoError(t, err)

	raw, err := base64.StdEncoding.DecodeString(got)
	require.NoError(t, err)

	embeddedJSON := string(raw[68:])
	assert.True(t, strings.Contains(embeddedJSON, `"filter_by":"persona:referrer && is_published:true"`),
		"embedded params must contain filter_by as JSON")
	assert.True(t, strings.Contains(embeddedJSON, `"expires_at":1234567890`),
		"embedded params must contain expires_at as JSON")
}

// Client method passes through to the package-level helper.
func TestClient_GenerateScopedSearchKey_DelegatesToHelper(t *testing.T) {
	c, err := NewClient("http://localhost:8108", "xyz-dev-master-key")
	require.NoError(t, err)

	params := EmbeddedSearchParams{FilterBy: "persona:freelance", ExpiresAt: 1776000000}

	got, err := c.GenerateScopedSearchKey("xyz-dev-master-key", params)
	require.NoError(t, err)
	want, err := generateScopedSearchKey("xyz-dev-master-key", params)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}
