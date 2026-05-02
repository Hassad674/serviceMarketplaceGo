// Package vies implements service.VIESValidator against the European
// Commission's REST endpoint with a Redis-backed positive cache.
//
// The endpoint is documented at:
//
//	https://ec.europa.eu/taxation_customs/vies/rest-api/check-vat-number
//
// Positive results are cached for 24h — VIES is slow and frequently
// throttled, so we MUST avoid hitting it on every page load. Negative
// results are NEVER cached because a VAT number can be activated
// retroactively and we want the user to be able to retry quickly.
package vies

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"marketplace-backend/internal/observability"
	"marketplace-backend/internal/port/service"
)

// DefaultEndpoint is the public REST URL of the European Commission's
// VIES service. Override via WithEndpoint in tests so we can point the
// adapter at an httptest server.
const DefaultEndpoint = "https://ec.europa.eu/taxation_customs/vies/rest-api/check-vat-number"

// DefaultCacheTTL is the lifetime of a positive cache entry. 24h
// matches the VIES service's own freshness guarantee.
const DefaultCacheTTL = 24 * time.Hour

// httpTimeout caps every outbound call to VIES. Backend CLAUDE.md
// pins external HTTP calls at 10 seconds.
const httpTimeout = 10 * time.Second

// Client is the VIES adapter — a Redis cache fronting the EC REST API.
type Client struct {
	redisClient *goredis.Client
	httpClient  *http.Client
	endpoint    string
	cacheTTL    time.Duration
}

// Option configures the Client at construction time.
type Option func(*Client)

// WithEndpoint overrides the VIES URL — used in tests to point at
// an httptest server. Pass DefaultEndpoint or omit in production.
func WithEndpoint(url string) Option {
	return func(c *Client) {
		if strings.TrimSpace(url) != "" {
			c.endpoint = url
		}
	}
}

// WithCacheTTL overrides the positive-cache lifetime. Zero or negative
// values fall back to DefaultCacheTTL.
func WithCacheTTL(d time.Duration) Option {
	return func(c *Client) {
		if d > 0 {
			c.cacheTTL = d
		}
	}
}

// WithHTTPClient injects a custom *http.Client (mainly for tests).
// Production code should NOT use this — the default client already
// has the correct 10s timeout.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) {
		if hc != nil {
			c.httpClient = hc
		}
	}
}

// NewClient builds a VIES adapter. redisClient must be non-nil — the
// cache is mandatory in production. Tests can pass a miniredis-backed
// client to keep the cache contract intact without spinning a real
// Redis instance.
func NewClient(redisClient *goredis.Client, opts ...Option) *Client {
	c := &Client{
		redisClient: redisClient,
		httpClient: &http.Client{
			Timeout:   httpTimeout,
			Transport: observability.HTTPClientTransport(http.DefaultTransport, "vies"),
		},
		endpoint: DefaultEndpoint,
		cacheTTL: DefaultCacheTTL,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// viesRequest is the JSON body POSTed to the EC REST endpoint.
type viesRequest struct {
	CountryCode string `json:"countryCode"`
	VATNumber   string `json:"vatNumber"`
}

// viesResponse models the EC REST endpoint response. All fields are
// permissive — VIES occasionally drops fields we don't expect, so
// missing values become empty strings rather than decode errors.
type viesResponse struct {
	IsValid           bool   `json:"isValid"`
	RequestDate       string `json:"requestDate"`
	UserError         string `json:"userError"`
	Name              string `json:"name"`
	Address           string `json:"address"`
	TraderName        string `json:"traderName"`
	TraderAddress     string `json:"traderAddress"`
	RequestIdentifier string `json:"requestIdentifier"`
	OriginalVATNumber string `json:"originalVatNumber"`
	VATNumber         string `json:"vatNumber"`
	CountryCode       string `json:"countryCode"`
}

// cacheKey returns the Redis key storing a positive VIES result.
// Format: vies:<CC><number>. Both inputs are uppercased + trimmed
// before being assembled so casing variants share a slot.
func cacheKey(cc, vat string) string {
	return "vies:" + strings.ToUpper(strings.TrimSpace(cc)) + strings.ToUpper(strings.TrimSpace(vat))
}

// Validate checks a VAT number against VIES, with a 24h Redis cache on
// positive results. Negative results bypass the cache so retries reach
// VIES immediately.
func (c *Client) Validate(ctx context.Context, countryCode, vatNumber string) (service.VIESResult, error) {
	cc := strings.ToUpper(strings.TrimSpace(countryCode))
	vat := strings.ToUpper(strings.TrimSpace(vatNumber))
	if cc == "" || vat == "" {
		return service.VIESResult{}, fmt.Errorf("vies: country code and VAT number are required")
	}

	key := cacheKey(cc, vat)

	// Cache lookup — only positive results are ever stored, so a hit
	// always means Valid=true.
	if c.redisClient != nil {
		if raw, err := c.redisClient.Get(ctx, key).Bytes(); err == nil && len(raw) > 0 {
			var cached service.VIESResult
			if err := json.Unmarshal(raw, &cached); err == nil {
				return cached, nil
			}
			// Corrupted entry — ignore and re-fetch.
		}
	}

	result, err := c.fetch(ctx, cc, vat)
	if err != nil {
		return service.VIESResult{}, err
	}

	// Cache positive results only.
	if result.Valid && c.redisClient != nil {
		if blob, err := json.Marshal(result); err == nil {
			// Redis failure here is non-fatal — the user gets the live
			// answer, the next call simply re-validates against VIES.
			_ = c.redisClient.Set(ctx, key, blob, c.cacheTTL).Err()
		}
	}

	return result, nil
}

// fetch performs the actual HTTPS round-trip against VIES.
func (c *Client) fetch(ctx context.Context, cc, vat string) (service.VIESResult, error) {
	body, err := json.Marshal(viesRequest{CountryCode: cc, VATNumber: vat})
	if err != nil {
		return service.VIESResult{}, fmt.Errorf("vies: marshal request: %w", err)
	}

	httpCtx, cancel := context.WithTimeout(ctx, httpTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(httpCtx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return service.VIESResult{}, fmt.Errorf("vies: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return service.VIESResult{}, fmt.Errorf("vies: http call failed: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return service.VIESResult{}, fmt.Errorf("vies: read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return service.VIESResult{}, fmt.Errorf("vies: unexpected status %d", resp.StatusCode)
	}

	var parsed viesResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return service.VIESResult{}, fmt.Errorf("vies: decode response: %w", err)
	}

	checkedAt := parseVIESDate(parsed.RequestDate)
	if checkedAt == 0 {
		checkedAt = time.Now().Unix()
	}

	// Prefer the canonical "name"/"address" fields; some country
	// gateways populate trader* instead, so fall through.
	registeredName := firstNonEmpty(parsed.Name, parsed.TraderName)
	registeredAddr := firstNonEmpty(parsed.Address, parsed.TraderAddress)
	respCC := firstNonEmpty(parsed.CountryCode, cc)
	respVAT := firstNonEmpty(parsed.VATNumber, vat)

	return service.VIESResult{
		Valid:          parsed.IsValid,
		CountryCode:    strings.ToUpper(respCC),
		VATNumber:      strings.ToUpper(respVAT),
		RegisteredName: registeredName,
		RegisteredAddr: registeredAddr,
		RawPayload:     raw,
		CheckedAt:      checkedAt,
	}, nil
}

// parseVIESDate parses the optional VIES requestDate. The format VIES
// publishes is RFC3339-ish — we accept a couple of common variants and
// fall back to 0 (caller substitutes time.Now) if parsing fails.
func parseVIESDate(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02T15:04:05.000Z07:00"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.Unix()
		}
	}
	return 0
}

// firstNonEmpty returns the first argument with a non-empty trimmed
// value — used to pick between alternative VIES field spellings.
func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
