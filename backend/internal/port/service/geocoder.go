package service

import (
	"context"
	"errors"
)

// Geocoder resolves a city + ISO-3166 country pair to decimal
// coordinates. It is an OPTIONAL dependency of the profile service:
// the profile save flow must never fail because of a geocoding
// error. Implementations live in adapter/nominatim (public OSM
// service) and can be swapped for a paid provider without touching
// the caller — the Geocoder contract is intentionally minimal.
//
// Callers must tolerate ErrGeocodingFailed and proceed without the
// coordinates. A missing lat/lng on the profile is interpreted by
// the UI as "render the location without map features" — it is
// never a fatal error.
type Geocoder interface {
	// Geocode returns the decimal latitude / longitude of the first
	// match for (city, countryCode). Implementations MUST apply
	// their own hard timeout so a slow provider cannot block the
	// caller's request beyond a few seconds. On any failure
	// (network error, timeout, no match, invalid response)
	// implementations MUST return ErrGeocodingFailed wrapped in the
	// original error so the caller can match with errors.Is.
	Geocode(ctx context.Context, city, countryCode string) (lat, lng float64, err error)
}

// ErrGeocodingFailed is the sentinel returned by Geocoder
// implementations when the coordinates could not be resolved.
// Callers convert it to a WARN log and fall back to NULL
// lat/lng — NEVER to an HTTP error for the end user.
var ErrGeocodingFailed = errors.New("geocoding failed")
