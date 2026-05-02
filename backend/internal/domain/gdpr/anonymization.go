// Package gdpr defines the domain types and helpers used by the
// GDPR right-to-erasure + right-to-export endpoints (P5).
//
// This package has no persistence responsibilities. The repository
// interface lives in port/repository/gdpr_repository.go and the
// adapter in adapter/postgres/gdpr_repository.go. The orchestration
// (request → email → soft-delete → cron purge) lives in app/gdpr.
package gdpr

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"strings"
)

// ErrSaltRequired is returned by HashEmail when the configured
// anonymization salt is the empty string. Callers (the cron purge
// scheduler, the integration tests) MUST detect this at boot and
// fail loudly: a missing salt would cause every anonymized hash
// to collapse to the same deterministic digest and defeat the
// forensic value of the audit log.
var ErrSaltRequired = errors.New("gdpr: anonymization salt is required")

// HashEmail produces sha256(email + salt) as a lowercase hex string.
// Decision 4 of the P5 brief: this hash replaces the actor email in
// audit log metadata after a hard-purge so an investigator can still
// answer "did this person do X" by recomputing the hash, but the
// raw PII is gone.
//
// Email is lowercased + trimmed before hashing so the digest is
// stable across casing variants of the same logical address. An
// empty email returns a usable digest of just the salt — callers
// should pass a real email but the function does not fail on empty
// input to keep the call site terse in the SQL UPDATE path.
func HashEmail(email, salt string) (string, error) {
	if salt == "" {
		return "", ErrSaltRequired
	}
	normalized := strings.ToLower(strings.TrimSpace(email))
	sum := sha256.Sum256([]byte(normalized + salt))
	return hex.EncodeToString(sum[:]), nil
}

// TruncateIP zeroes the last two octets of an IPv4 address (or the
// last 96 bits of an IPv6 address) so the audit row keeps geolocation
// resolution at city/ISP level without retaining a unique device
// fingerprint. RGPD recital 26 lets us keep this attenuated form
// because it is no longer reasonably attributable to an identified
// person.
//
// Returns the input unchanged when the parse fails — the caller
// (a SQL UPDATE) prefers a malformed pass-through over a NULL,
// since the audit_logs.metadata column is JSONB free-form and
// this function runs on cold data the operator cannot fix.
//
// Examples:
//
//	TruncateIP("203.0.113.42")            → "203.0.x.x"
//	TruncateIP("2001:db8::1")             → "2001:db8::"
//	TruncateIP("not-an-ip")               → "not-an-ip"
//	TruncateIP("")                        → ""
func TruncateIP(ip string) string {
	if ip == "" {
		return ""
	}
	parsed := net.ParseIP(strings.TrimSpace(ip))
	if parsed == nil {
		return ip
	}
	if v4 := parsed.To4(); v4 != nil {
		return fmt.Sprintf("%d.%d.x.x", v4[0], v4[1])
	}
	// IPv6 — keep the first 32 bits (network prefix), zero the rest.
	masked := parsed.Mask(net.CIDRMask(32, 128))
	return masked.String()
}
