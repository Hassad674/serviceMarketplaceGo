// Package nominatim provides a minimal HTTP adapter for the public
// OpenStreetMap Nominatim geocoding service, satisfying the
// port/service.Geocoder interface.
//
// The adapter is deliberately dependency-free (stdlib only) and
// single-purpose (forward geocoding by city + country code). It
// lives here rather than in a shared http-client package because
// the Nominatim usage policy requires a descriptive User-Agent
// and a bounded timeout — pulling in a generic retry/cache library
// would be premature complexity for the V1 cadence (profile saves
// per user are minutes apart and do not stress even the strictest
// rate limit).
//
// See port/service/geocoder.go for the contract. Callers MUST
// tolerate ErrGeocodingFailed and proceed without coordinates: the
// profile save flow NEVER fails because of a geocoding error.
package nominatim

import (
	"net/http"
	"time"

	"marketplace-backend/internal/observability"
)

// defaultEndpoint is the public Nominatim forward-geocoding URL.
// Overridable in tests (see geocoder_test.go) to point at an
// httptest.Server.
const defaultEndpoint = "https://nominatim.openstreetmap.org/search"

// defaultHTTPTimeout is the hard ceiling for a single geocoding
// call. Kept short because the profile save flow waits on this
// synchronously — a slow provider must not hold the user's save
// button for more than a couple of seconds.
const defaultHTTPTimeout = 2 * time.Second

// newHTTPClient builds the bounded-timeout HTTP client used by the
// geocoder. Separated from NewGeocoder so tests can override the
// transport via a custom http.Client if needed.
//
// The transport is wrapped with observability.HTTPClientTransport so
// each outbound geocoding request is captured as an OTel client span
// (no-op when tracing is disabled).
func newHTTPClient() *http.Client {
	return &http.Client{
		Timeout:   defaultHTTPTimeout,
		Transport: observability.HTTPClientTransport(http.DefaultTransport, "nominatim"),
	}
}
