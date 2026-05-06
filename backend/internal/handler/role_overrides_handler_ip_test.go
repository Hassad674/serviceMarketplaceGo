package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestRoleOverridesHandler_WithIPExtractor_TakesPrecedenceOverHeader is
// the V7 N7 regression guard. With the trust-aware extractor wired,
// the audit log MUST honour the rate-limiter's IP — not the raw
// X-Forwarded-For header — so a client cannot spoof the IP recorded
// against their action.
func TestRoleOverridesHandler_WithIPExtractor_TakesPrecedenceOverHeader(t *testing.T) {
	h := &RoleOverridesHandler{}
	h.WithIPExtractor(func(*http.Request) string {
		// Simulate the rate-limiter returning the trusted edge IP.
		return "10.0.0.42"
	})

	req := httptest.NewRequest("PATCH", "/orgs/x/role-permissions", nil)
	// Attempt to spoof X-Forwarded-For — the extractor must ignore it.
	req.Header.Set("X-Forwarded-For", "1.2.3.4, 5.6.7.8")
	req.RemoteAddr = "203.0.113.10:54321"

	got := h.resolveIP(req)
	assert.Equal(t, "10.0.0.42", got,
		"V7 N7: when WithIPExtractor is wired, the spoofable XFF header MUST be ignored")
}

// TestRoleOverridesHandler_NoExtractor_FallsBackToLegacy verifies the
// nil-extractor path keeps the test/dev fallback alive — the legacy
// clientIP() reader still parses XFF for callers that have not yet
// wired the trust-aware extractor.
func TestRoleOverridesHandler_NoExtractor_FallsBackToLegacy(t *testing.T) {
	h := &RoleOverridesHandler{} // no WithIPExtractor

	req := httptest.NewRequest("PATCH", "/orgs/x/role-permissions", nil)
	req.Header.Set("X-Forwarded-For", "9.9.9.9")
	req.RemoteAddr = "203.0.113.10:54321"

	got := h.resolveIP(req)
	// Legacy path picks the first XFF entry — proves the fallback is
	// intact for tests, while the production code path always wires
	// the extractor (see bootstrap.go).
	assert.Equal(t, "9.9.9.9", got)
}
