package nominatim

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"marketplace-backend/internal/port/service"
)

// Geocoder is the port/service.Geocoder implementation backed by
// the public OpenStreetMap Nominatim API. It performs forward
// geocoding only (city + country to lat/lng), with a strict 2s
// hard timeout and a graceful-degradation contract: every failure
// returns service.ErrGeocodingFailed wrapped in the original error
// so the caller can match with errors.Is and log a warning
// without failing the outer profile save.
//
// No caching, no retries — the profile save cadence (rare, human-
// driven) makes both premature. Adding them later is a drop-in
// change that does not touch the port.
type Geocoder struct {
	client    *http.Client
	endpoint  string
	userAgent string
}

// NewGeocoder returns a geocoder ready to talk to the public
// Nominatim instance. The userAgent MUST identify the application
// per https://operations.osmfoundation.org/policies/nominatim/ —
// Nominatim will rate-limit or block clients that do not.
func NewGeocoder(userAgent string) *Geocoder {
	return &Geocoder{
		client:    newHTTPClient(),
		endpoint:  defaultEndpoint,
		userAgent: userAgent,
	}
}

// nominatimResult is the JSON shape returned by the public API.
// We only care about lat/lon; every other field is ignored.
type nominatimResult struct {
	Lat string `json:"lat"`
	Lon string `json:"lon"`
}

// Geocode resolves (city, countryCode) to decimal coordinates.
// Implements port/service.Geocoder.
//
// Behavior on failure paths (every branch returns
// service.ErrGeocodingFailed wrapped in the original cause):
//
//   - empty city: immediate fail (Nominatim requires at least one
//     locality-level parameter)
//   - request-building error: fail
//   - transport error / timeout: fail
//   - non-2xx HTTP status: fail
//   - malformed JSON / empty result array: fail
//   - lat/lon cannot be parsed as float64: fail
//
// On success the first result is returned — Nominatim sorts by
// relevance so the first entry is the best match for the city.
func (g *Geocoder) Geocode(ctx context.Context, city, countryCode string) (float64, float64, error) {
	if city == "" {
		return 0, 0, service.ErrGeocodingFailed
	}

	req, err := g.buildRequest(ctx, city, countryCode)
	if err != nil {
		return 0, 0, err
	}

	resp, err := g.client.Do(req)
	if err != nil {
		return 0, 0, fmt.Errorf("%w: transport: %v", service.ErrGeocodingFailed, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, 0, fmt.Errorf("%w: status %d", service.ErrGeocodingFailed, resp.StatusCode)
	}

	var results []nominatimResult
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return 0, 0, fmt.Errorf("%w: decode: %v", service.ErrGeocodingFailed, err)
	}
	if len(results) == 0 {
		return 0, 0, service.ErrGeocodingFailed
	}

	lat, err := strconv.ParseFloat(results[0].Lat, 64)
	if err != nil {
		return 0, 0, errors.Join(service.ErrGeocodingFailed, err)
	}
	lng, err := strconv.ParseFloat(results[0].Lon, 64)
	if err != nil {
		return 0, 0, errors.Join(service.ErrGeocodingFailed, err)
	}
	return lat, lng, nil
}

// buildRequest assembles the HTTP request with the required
// User-Agent and Accept headers. Extracted so Geocode stays under
// the 50-line cap and the error-path tests can target it
// independently if needed.
func (g *Geocoder) buildRequest(ctx context.Context, city, countryCode string) (*http.Request, error) {
	params := url.Values{}
	params.Set("format", "json")
	params.Set("limit", "1")
	params.Set("city", city)
	if countryCode != "" {
		params.Set("countrycodes", countryCode)
	}

	reqURL := g.endpoint + "?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: build request: %v", service.ErrGeocodingFailed, err)
	}
	req.Header.Set("User-Agent", g.userAgent)
	req.Header.Set("Accept", "application/json")
	return req, nil
}
