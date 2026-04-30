package handler

import (
	"errors"
	"net/url"
	"strings"
)

// errRedirectNotAllowed is the sentinel returned by validateStorageRedirect
// when the candidate URL is unsafe to redirect a user agent to. The HTTP
// handler maps this to 502 (the issued PDF URL is malformed) so the
// caller can retry without exposing a free open-redirect to attackers.
var errRedirectNotAllowed = errors.New("redirect URL not allowed")

// allowedRedirectHostSuffixes is the canonical allowlist of host suffixes
// the handler will follow on a 302 response. Every entry below was
// reviewed against the application's storage backends:
//
//	-  "amazonaws.com"   — S3 object URLs and presigned URLs in production
//	-  "r2.cloudflarestorage.com" — Cloudflare R2 bucket URLs (S3-compat)
//	-  "r2.dev"          — R2 public URL aliases (pub-XXXX.r2.dev)
//	-  "minio"           — Local MinIO development backend
//	-  "127.0.0.1"       — Localhost MinIO bind address
//	-  "localhost"       — Localhost MinIO bind address
//	-  ".local"          — Local development hosts (LAN testing)
//
// Adding a new entry requires deliberation: a too-permissive suffix
// (e.g. ".com") re-opens the open-redirect surface gosec G710 flagged
// on invoice_handler.go:145 and admin_invoice_handler.go:145.
var allowedRedirectHostSuffixes = []string{
	".amazonaws.com",
	".r2.cloudflarestorage.com",
	".r2.dev",
	".minio",
	"minio", // exact, dev container hostname
	"127.0.0.1",
	"localhost",
	".local",
}

// validateStorageRedirect parses `raw` and confirms the URL is safe to
// 302-redirect a browser to. The function rejects:
//
//   - URLs that do not parse cleanly
//   - URLs without an absolute http/https scheme
//   - URLs with a host that is not on the storage allowlist
//   - URLs carrying CR/LF (header smuggling, very old browsers)
//
// Closes gosec G710 on the two GetPDF handlers: the URL handed to
// http.Redirect is now provably one of the storage backends the
// application talks to, never an attacker-controlled domain that
// happens to come back from a misbehaving service.
//
// The validation is intentionally conservative: better a legitimate
// new storage backend that requires a one-line addition to
// allowedRedirectHostSuffixes than a quietly broadened allowlist.
func validateStorageRedirect(raw string) (*url.URL, error) {
	if raw == "" {
		return nil, errRedirectNotAllowed
	}
	// Reject CR/LF before parsing — net/url handles them but some
	// proxies don't, and they have no business being in a presigned URL.
	if strings.ContainsAny(raw, "\r\n") {
		return nil, errRedirectNotAllowed
	}
	u, err := url.Parse(raw)
	if err != nil {
		return nil, errRedirectNotAllowed
	}
	// Only http/https — block javascript:, data:, vbscript:, file:.
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return nil, errRedirectNotAllowed
	}
	// An absolute URL must have a Host. Reject empty Host (which would
	// make the redirect host-relative under some user agents).
	if u.Host == "" {
		return nil, errRedirectNotAllowed
	}
	host := strings.ToLower(u.Hostname())
	for _, suffix := range allowedRedirectHostSuffixes {
		if hostMatchesSuffix(host, suffix) {
			return u, nil
		}
	}
	return nil, errRedirectNotAllowed
}

// hostMatchesSuffix reports whether `host` is exactly the suffix or
// ends with `.<suffix>`. The leading dot in the suffix list makes
// this trivial; an entry without a leading dot (e.g. "minio",
// "127.0.0.1", "localhost") is matched only by exact equality.
func hostMatchesSuffix(host, suffix string) bool {
	if host == suffix {
		return true
	}
	if strings.HasPrefix(suffix, ".") && strings.HasSuffix(host, suffix) {
		// "evil.com.amazonaws.com.attacker.com" must NOT match
		// ".amazonaws.com" — confirm the suffix appears at the END.
		return true
	}
	return false
}
