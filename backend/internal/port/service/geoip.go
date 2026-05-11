package service

import "context"

// GeoIPLookup resolves an IPv4 / IPv6 address to a coarse city +
// country pair. It is a tiny interface on purpose (ISP): the Sécurité
// page only needs the city to render a Malt-style row — everything
// else (timezone, ASN, ISP) is out of scope.
//
// The returned GeoLocation is always safe to consume:
//   - On lookup failure (private IP, rate limit, timeout) the adapter
//     returns an empty GeoLocation + nil error. The caller persists
//     the empty fields and the UI falls back to "—".
//   - On adapter misconfiguration (network down, DNS failure) the
//     adapter logs at WARN and still returns an empty GeoLocation —
//     the audit row must NEVER block on a third-party hop.
type GeoIPLookup interface {
	Lookup(ctx context.Context, ip string) (GeoLocation, error)
}

// GeoLocation is the minimal city/country pair the Sécurité page
// renders. Empty strings are the documented "unknown" value — there
// is no other "missing" sentinel because the column defaults to ''
// at the SQL level (migration 150).
type GeoLocation struct {
	City        string
	CountryCode string // ISO 3166-1 alpha-2, uppercase ("FR", "US", ...)
}
