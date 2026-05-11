package handler

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"

	"marketplace-backend/internal/app/auth"
	"marketplace-backend/internal/domain/gdpr"
)

// sessionFingerprint extracts the request-side metadata used to fill
// the user_sessions audit row (B.4 + SEC-SESSIONS / migration 150).
//
// Two flavours of data come out of this helper, written to two
// different sets of columns:
//
//  1. Forensic (security): UserAgentHash (SHA-256 first 16 hex of the
//     raw UA) + IPAnonymized (/24 IPv4 or /64 IPv6). Loaded onto the
//     pre-existing columns from migration 147. Untouched by the
//     SEC-SESSIONS work — security and audit tooling keep relying on
//     these.
//
//  2. Display (Sécurité page): DeviceLabel + Browser + OS parsed from
//     the raw UA by handler.ParseUserAgent, and the raw (un-anonymized)
//     RemoteIP used downstream by the GeoIP goroutine to resolve
//     {city, country_code}. The raw IP is NEVER persisted — only its
//     /24 truncation lands in the row. The full IP only exists in
//     memory for the short window of the GeoIP lookup.
//
// Returns an empty fingerprint (UserAgentHash + IPAnonymized both
// empty) when ipExtractor is nil or the request lacks an IP — the
// auth service then logs a slog.Warn and skips the audit row,
// keeping the auth flow itself unaffected.
func (h *AuthHandler) sessionFingerprint(r *http.Request) auth.SessionFingerprint {
	var ip string
	if h.ipExtractor != nil {
		ip = h.ipExtractor(r)
	}
	if ip == "" {
		return auth.SessionFingerprint{}
	}

	rawUA := r.UserAgent()
	parsed := ParseUserAgent(rawUA)

	return auth.SessionFingerprint{
		UserAgentHash: hashUserAgentForSession(rawUA),
		IPAnonymized:  gdpr.TruncateIP(ip),

		// SEC-SESSIONS — display columns + raw IP carrier.
		DeviceLabel: parsed.Label,
		Browser:     parsed.Browser,
		OS:          parsed.OS,
		RemoteIP:    ip,
	}
}

// hashUserAgentForSession returns the first 16 hex characters of
// SHA-256(ua). Empty input returns empty string so the caller can
// detect "no UA available" without parsing.
func hashUserAgentForSession(ua string) string {
	if ua == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(ua))
	return hex.EncodeToString(sum[:8]) // 8 bytes = 16 hex chars
}
