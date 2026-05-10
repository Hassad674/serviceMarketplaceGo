package handler

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"

	"marketplace-backend/internal/app/auth"
	"marketplace-backend/internal/domain/gdpr"
)

// sessionFingerprint extracts the (anonymized IP, hashed user-agent)
// pair used to fill the user_sessions audit row (B.4).
//
//   - IP comes from the configured ipExtractor (which already
//     applies the trusted-proxy allowlist) and is then truncated to
//     /24 (IPv4) or /64 (IPv6) via gdpr.TruncateIP. Persisted as
//     CIDR notation through the INET column.
//   - User-Agent is SHA-256 hashed and the first 16 hex characters
//     kept. 16 hex = 64 bits of identity which is plenty to
//     distinguish realistic UAs (browser+OS+major version) without
//     creating a per-device fingerprint that would itself be PII.
//
// Returns an empty fingerprint when ipExtractor is nil or the
// request lacks an IP — the auth service then logs a slog.Warn and
// skips the audit row, keeping the auth flow itself unaffected.
func (h *AuthHandler) sessionFingerprint(r *http.Request) auth.SessionFingerprint {
	var ip string
	if h.ipExtractor != nil {
		ip = h.ipExtractor(r)
	}
	if ip == "" {
		return auth.SessionFingerprint{}
	}
	return auth.SessionFingerprint{
		UserAgentHash: hashUserAgentForSession(r.UserAgent()),
		IPAnonymized:  gdpr.TruncateIP(ip),
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
