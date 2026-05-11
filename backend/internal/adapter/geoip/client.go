// Package geoip is the adapter that resolves an IP address to a
// {city, country_code} pair for the Sécurité page session list. The
// MVP uses the free ipapi.co endpoint — no API key, 1k req/day free
// tier which is plenty for B2B login volumes. A swap to MaxMind /
// ip-api.com / Cloudflare's IP geolocation is a one-line change in
// cmd/api/main.go.
//
// Design goals (all derived from the brief):
//   - Best-effort: a lookup failure must NEVER block session creation.
//     The auth flow fires-and-forgets through a goroutine; this adapter
//     contributes a strict 2s timeout per call.
//   - Private / loopback / link-local IPs return empty GeoLocation
//     without ever hitting the network — those addresses have no
//     city / country and a third-party probe would just waste a
//     request from the free quota.
//   - No third-party SDK. A standard net/http client + a tiny JSON
//     decode is enough; the surface stays under 100 LoC, easy to
//     audit and swap.
package geoip

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"marketplace-backend/internal/port/service"
)

// DefaultEndpoint is the public ipapi.co JSON endpoint. Tests inject a
// httptest server URL via NewClientWithEndpoint to avoid hitting the
// real service.
const DefaultEndpoint = "https://ipapi.co"

// DefaultTimeout caps every Lookup call. 2 seconds is generous for a
// healthy CDN-fronted endpoint and tight enough that a hanging
// provider never stalls the surrounding goroutine.
const DefaultTimeout = 2 * time.Second

// Client implements service.GeoIPLookup against the ipapi.co API.
type Client struct {
	endpoint string
	http     *http.Client
}

// NewClient returns a Client wired to the public ipapi.co endpoint
// with the default 2s timeout.
func NewClient() *Client {
	return &Client{
		endpoint: DefaultEndpoint,
		http:     &http.Client{Timeout: DefaultTimeout},
	}
}

// NewClientWithEndpoint is the constructor used by unit tests: it lets
// the caller swap the base URL to an httptest.Server. Production code
// always goes through NewClient.
func NewClientWithEndpoint(endpoint string, timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	return &Client{
		endpoint: strings.TrimRight(endpoint, "/"),
		http:     &http.Client{Timeout: timeout},
	}
}

// Lookup resolves ip to a {city, country_code} pair. Returns an empty
// GeoLocation (no error) when:
//   - the input is empty / unparseable
//   - the IP is private / loopback / link-local / unspecified
//   - the upstream call times out, rate-limits, or returns a non-2xx
//
// The "no error" contract is intentional: the caller (session audit
// goroutine) treats any failure as "unknown location" and stores the
// empty default. We do NOT want a transient ipapi 5xx to surface as
// an error log spike — the WARN logs below are enough.
func (c *Client) Lookup(ctx context.Context, ip string) (service.GeoLocation, error) {
	if c == nil {
		return service.GeoLocation{}, nil
	}
	trimmed := strings.TrimSpace(ip)
	if trimmed == "" {
		return service.GeoLocation{}, nil
	}
	parsed := net.ParseIP(trimmed)
	if parsed == nil || isNonRoutable(parsed) {
		return service.GeoLocation{}, nil
	}

	endpoint, err := url.JoinPath(c.endpoint, trimmed, "json")
	if err != nil {
		// JoinPath only fails on truly broken endpoints. Treat as
		// "configuration issue" and return empty silently — the
		// surrounding goroutine is fire-and-forget.
		return service.GeoLocation{}, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return service.GeoLocation{}, nil
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "marketplace-service-geoip/1.0")

	resp, err := c.http.Do(req)
	if err != nil {
		// Timeout, DNS, network — all best-effort. WARN once and
		// return empty so the goroutine completes cleanly.
		slog.Warn("geoip: lookup failed", "ip", trimmed, "error", err)
		return service.GeoLocation{}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		slog.Warn("geoip: non-2xx response", "ip", trimmed, "status", resp.StatusCode)
		return service.GeoLocation{}, nil
	}

	// ipapi.co JSON shape (subset we use):
	//   {"city": "Paris", "country_code": "FR", "error": false, ...}
	// On error the body looks like:
	//   {"error": true, "reason": "RateLimited"}
	var body struct {
		City        string `json:"city"`
		CountryCode string `json:"country_code"`
		Error       bool   `json:"error"`
		Reason      string `json:"reason"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		slog.Warn("geoip: decode failed", "ip", trimmed, "error", err)
		return service.GeoLocation{}, nil
	}
	if body.Error {
		slog.Warn("geoip: api error", "ip", trimmed, "reason", body.Reason)
		return service.GeoLocation{}, nil
	}

	return service.GeoLocation{
		City:        strings.TrimSpace(body.City),
		CountryCode: strings.ToUpper(strings.TrimSpace(body.CountryCode)),
	}, nil
}

// isNonRoutable returns true for IPs that have no meaningful geo
// location: loopback (127.0.0.0/8, ::1), link-local (169.254.0.0/16,
// fe80::/10), private (RFC1918, fc00::/7), unspecified (0.0.0.0, ::).
// Calling ipapi.co for any of these wastes a free-tier slot.
func isNonRoutable(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsUnspecified() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}
	if ip.IsPrivate() {
		return true
	}
	return false
}

// ensure Client satisfies the port interface at compile time.
var _ service.GeoIPLookup = (*Client)(nil)

// guard against an unused-import lint when errors is not referenced
// elsewhere in the file (kept reserved for future enrichment paths
// that need errors.Is on transport errors).
var _ = errors.New
var _ = fmt.Sprintf
