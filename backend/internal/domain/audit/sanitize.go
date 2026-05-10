package audit

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// sensitiveMetadataKeys lists the JSONB metadata keys whose VALUES must
// never be persisted in plaintext inside the audit_logs table.
//
// Reasoning (RGPD art. 5-1-c — data minimization): an audit row records
// who did what, when, and against which resource. It is NOT a place to
// store searchable PII. Persisting cleartext emails on every
// `auth.login_failure` event, for example, builds a permanent index of
// "this email exists on the platform" that survives the user's account
// deletion and broadens the blast radius of any DB compromise.
//
// The set is intentionally conservative — it covers the values we know
// are passed today (email, phone) plus close variants (to_email,
// from_email, recipient) and the highest-risk financial identifier we
// might add tomorrow (iban). Any value found at one of these keys is
// replaced by a deterministic short SHA-256 prefix so admins can still
// answer "is this the same actor as the row from last week?" without
// ever seeing the cleartext.
//
// Keys NOT in this list are preserved verbatim — `reason`, `user_agent`,
// `attempted_action`, and similar diagnostic fields remain readable.
var sensitiveMetadataKeys = map[string]struct{}{
	"email":      {},
	"to_email":   {},
	"from_email": {},
	"recipient":  {},
	"phone":      {},
	"iban":       {},
}

// sanitizedHashLength is the hex-character length of the truncated
// SHA-256 prefix written in place of a sensitive value. 16 chars =
// 64 bits of entropy — large enough to make accidental collisions
// negligible across the lifetime of the audit log, small enough to keep
// JSONB rows compact.
const sanitizedHashLength = 16

// SanitizeMetadata returns a copy of `meta` where every value reachable
// at a sensitive key (see `sensitiveMetadataKeys`) is replaced by a
// 16-hex-char SHA-256 prefix of its string representation.
//
// Behaviour rules (locked by sanitize_test.go):
//   - Top-level sensitive keys are replaced with their hashed prefix.
//   - Nested maps (`map[string]any`) are walked recursively; the same
//     rule applies at every depth.
//   - Non-sensitive keys keep their original value byte-for-byte.
//   - Non-string values at a sensitive key are first stringified via
//     fmt.Sprint to keep the helper total — the caller never has to
//     check whether `email` arrived as a `string` or as `[]byte`.
//   - Nil input returns nil (no allocation, safe for callers that pass
//     `entry.Metadata` directly).
//   - The function is pure: same input -> same output, no I/O, no time
//     dependency. Hashes are deterministic so admin tooling can match
//     "rows about the same actor" by comparing the hex prefix.
//
// SECURITY NOTE: this is a one-way obfuscation, NOT encryption. An
// attacker who knows a candidate email can confirm whether it appears
// in the log by computing the same hash. That is acceptable here —
// the alternative (storing cleartext) is strictly worse. For the
// stronger guarantee (impossible to confirm a guess), the audit row
// would have to be salted per row, which would defeat the "match the
// same actor across rows" use-case admins rely on.
func SanitizeMetadata(meta map[string]any) map[string]any {
	if meta == nil {
		return nil
	}
	out := make(map[string]any, len(meta))
	for k, v := range meta {
		if _, sensitive := sensitiveMetadataKeys[k]; sensitive {
			out[k] = hashSensitiveValue(v)
			continue
		}
		if nested, ok := v.(map[string]any); ok {
			out[k] = SanitizeMetadata(nested)
			continue
		}
		out[k] = v
	}
	return out
}

// hashSensitiveValue returns a deterministic 16-hex-char SHA-256
// prefix of v's string representation. Empty / nil values pass
// through as the empty string so the caller can still tell "no value
// recorded" apart from "value redacted".
func hashSensitiveValue(v any) string {
	if v == nil {
		return ""
	}
	str := stringify(v)
	if str == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(str))
	return hex.EncodeToString(sum[:])[:sanitizedHashLength]
}

// stringify produces a stable string form for hashing. Strings and
// byte slices are returned verbatim; every other type falls back to
// fmt.Sprint so the helper stays total — the caller never has to know
// the underlying Go type of the value flowing through the metadata map.
func stringify(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case []byte:
		return string(t)
	default:
		return fmt.Sprint(t)
	}
}
