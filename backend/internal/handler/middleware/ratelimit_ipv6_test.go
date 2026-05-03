package middleware

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// F.5 S6 — IPv6 /64 normalisation. An attacker with a routed /64 has
// 2^64 distinct addresses; without prefix masking, the rate limiter
// is effectively disabled against IPv6 abuse. These tests pin the
// contract: every IPv6 address inside the same /64 hits the SAME
// limiter bucket; IPv4 keeps per-address granularity (any change to
// the /32 mask would over-throttle shared NATs).

func TestNormaliseIPForLimiter_IPv6_MasksTo64(t *testing.T) {
	cases := []struct {
		name string
		ip   string
		want string
	}{
		{"low order changes", "2001:db8:abcd:1234::1", "2001:db8:abcd:1234::/64"},
		{"high last byte", "2001:db8:abcd:1234:ffff:ffff:ffff:ffff", "2001:db8:abcd:1234::/64"},
		{"middle bits", "2001:db8:abcd:1234:5:6:7:8", "2001:db8:abcd:1234::/64"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ip := net.ParseIP(tc.ip)
			require.NotNil(t, ip, "test fixture must be parseable")
			got := normaliseIPForLimiter(ip)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestNormaliseIPForLimiter_IPv4_KeepsPerAddressGranularity(t *testing.T) {
	cases := []struct {
		name string
		ip   string
		want string
	}{
		{"private", "192.168.1.1", "192.168.1.1"},
		{"public", "203.0.113.42", "203.0.113.42"},
		{"loopback", "127.0.0.1", "127.0.0.1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ip := net.ParseIP(tc.ip)
			require.NotNil(t, ip)
			got := normaliseIPForLimiter(ip)
			assert.Equal(t, tc.want, got)
		})
	}
}

// TestRateLimit_IPv6_NormalizesTo64 is the brief's mandated case:
// 65 distinct IPv6 addresses inside the same /64 must trigger the
// throttle on the 65th request when the limit is set to 64.
func TestRateLimit_IPv6_NormalizesTo64(t *testing.T) {
	rl, _ := newRateLimiterTest(t)
	policy := RateLimitPolicy{Class: RateLimitClassGlobal, Limit: 64, Window: time.Minute}
	handler := rl.Middleware(policy, rl.IPKey())(newOKHandler())

	// 64 requests with distinct host bits in the /64 must all pass —
	// they collapse into the same bucket but the count starts at zero.
	for i := 0; i < 64; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		// Vary the last 64 bits, keep the prefix 2001:db8:1:2.
		req.RemoteAddr = (&net.TCPAddr{
			IP:   net.ParseIP("2001:db8:1:2:0:0:0:0"),
			Port: 12345,
		}).String()
		// Replace the trailing 4 hex chars to force distinct addresses.
		req.RemoteAddr = "[2001:db8:1:2::" + hexInt(i+1) + "]:12345"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		require.Equalf(t, http.StatusOK, rec.Code, "request %d in same /64 must pass", i+1)
	}

	// 65th distinct host bits in the SAME /64 must trip 429 — proves
	// the addresses are bucketed together, not separately.
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "[2001:db8:1:2::dead]:12345"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code,
		"65th address in the same /64 must hit the per-network throttle")
}

// TestRateLimit_IPv6_DifferentSlash64s_AreIndependent — defensive
// counterpart: separate /64 networks each get their own bucket. A
// future bug that masks too far (e.g. /48) would flunk this test by
// throttling unrelated networks.
func TestRateLimit_IPv6_DifferentSlash64s_AreIndependent(t *testing.T) {
	rl, _ := newRateLimiterTest(t)
	policy := RateLimitPolicy{Class: RateLimitClassGlobal, Limit: 1, Window: time.Minute}
	handler := rl.Middleware(policy, rl.IPKey())(newOKHandler())

	// /64 A — burn the budget.
	reqA := httptest.NewRequest(http.MethodGet, "/", nil)
	reqA.RemoteAddr = "[2001:db8:1:1::1]:12345"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, reqA)
	require.Equal(t, http.StatusOK, rec.Code)

	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, reqA)
	require.Equal(t, http.StatusTooManyRequests, rec.Code)

	// /64 B — must still pass (different prefix).
	reqB := httptest.NewRequest(http.MethodGet, "/", nil)
	reqB.RemoteAddr = "[2001:db8:1:2::1]:12345"
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, reqB)
	assert.Equal(t, http.StatusOK, rec.Code,
		"distinct /64 prefixes must own distinct buckets")
}

// hexInt formats a small int as a lowercase hex group for IPv6
// rendering. Keeps the test fixture readable without pulling fmt.
func hexInt(n int) string {
	const hexChars = "0123456789abcdef"
	if n == 0 {
		return "0"
	}
	var b [4]byte
	idx := 4
	for n > 0 && idx > 0 {
		idx--
		b[idx] = hexChars[n&0xf]
		n >>= 4
	}
	return string(b[idx:])
}
