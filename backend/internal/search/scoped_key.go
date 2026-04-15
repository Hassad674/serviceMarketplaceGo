package search

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

// scoped_key.go implements Typesense's scoped search API key
// generation algorithm. The frontend uses these short-lived keys
// to query Typesense directly without ever holding the master key.
//
// The wire format documented at
// https://typesense.org/docs/latest/api/api-keys.html#generate-scoped-search-api-key
// is:
//
//	scoped_key = base64(
//	    base64(hmac_sha256(parent_key, embedded_params))
//	    + parent_key_prefix
//	    + embedded_params_json
//	)
//
// where `parent_key_prefix` is the FIRST 4 characters of the parent
// (master) API key. Typesense uses those 4 characters to look up the
// parent key on its side and validate the HMAC. The HMAC is
// base64-encoded, NOT hex — using hex produces a self-consistent but
// wrong output that Typesense silently rejects at query time.
//
// The function is a pure helper — no I/O, no state, deterministic
// output for fixed inputs. That makes it trivial to unit-test the
// HMAC step against a known fixture.

// EmbeddedSearchParams is the subset of search parameters baked into
// the scoped key. Typesense will refuse any query whose parameters
// contradict these — so we use it to enforce `filter_by` (persona
// scoping) and `expires_at` (TTL).
//
// Field order does NOT matter for HMAC correctness because Go's JSON
// marshaller is deterministic for struct types. We rely on that here
// so the same input always produces the same key.
type EmbeddedSearchParams struct {
	// FilterBy is appended to every search request sent with this
	// key by the Typesense server. Used to lock the persona so a
	// scoped freelance key can never reach an agency document.
	FilterBy string `json:"filter_by,omitempty"`

	// ExpiresAt is the Unix epoch (seconds) at which the key is
	// rejected by Typesense. We pass it explicitly instead of
	// relying on Typesense's `expires_in` so we can return it to
	// the frontend in the same response and let TanStack Query
	// rotate the key proactively.
	ExpiresAt int64 `json:"expires_at,omitempty"`
}

// GenerateScopedSearchKey computes the scoped key string for the
// given parent key + embedded parameters. Returns an error only on
// invalid input (empty parent key) — the HMAC + base64 steps cannot
// fail at runtime.
//
// This function is a method on *Client so callers can reach it via
// the existing dependency they already have, but it does not touch
// the client state — it could equally well be a free function.
func (c *Client) GenerateScopedSearchKey(parentKey string, params EmbeddedSearchParams) (string, error) {
	return generateScopedSearchKey(parentKey, params)
}

// generateScopedSearchKey is the package-private implementation. It
// is exposed via the Client method above + reused directly in tests
// where having a free function is more convenient than instantiating
// a client.
func generateScopedSearchKey(parentKey string, params EmbeddedSearchParams) (string, error) {
	trimmed := strings.TrimSpace(parentKey)
	if trimmed == "" {
		return "", fmt.Errorf("scoped key: parent key is required")
	}
	if len(trimmed) < 4 {
		return "", fmt.Errorf("scoped key: parent key must be at least 4 characters")
	}

	// Use a custom encoder with HTML escaping DISABLED. Go's
	// default json.Marshal encodes `&` as `\u0026`, which would
	// break Typesense's filter_by parser (it expects the literal
	// `&&` token). Typesense's reference implementation uses raw
	// JSON so we must match byte-for-byte.
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(params); err != nil {
		return "", fmt.Errorf("scoped key: marshal params: %w", err)
	}
	// json.Encoder.Encode appends a trailing newline; strip it so
	// the HMAC matches what an inline json.Marshal would produce
	// minus the HTML escaping.
	embedded := bytes.TrimRight(buf.Bytes(), "\n")

	mac := hmac.New(sha256.New, []byte(trimmed))
	if _, err := mac.Write(embedded); err != nil {
		// hmac.Hash never returns an error from Write, but stay
		// defensive in case the stdlib contract ever changes.
		return "", fmt.Errorf("scoped key: hmac write: %w", err)
	}
	digest := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	prefix := trimmed[:4]
	raw := digest + prefix + string(embedded)
	return base64.StdEncoding.EncodeToString([]byte(raw)), nil
}
