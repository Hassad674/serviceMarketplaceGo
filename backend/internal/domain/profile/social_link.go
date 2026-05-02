package profile

import (
	"errors"
	"net"
	"net/netip"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
)

// SocialLink represents a social network link displayed on the
// organization's public profile. Phase R2 anchors social links on the
// org rather than on an individual user. A single organization can
// hold multiple independent sets of links — one per persona — to
// support users that operate as both freelance and referrer.
type SocialLink struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	Persona        SocialLinkPersona
	Platform       string
	URL            string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// SocialLinkPersona is the identity scope under which a social link
// is displayed. Agencies keep their legacy single set under the
// "agency" persona. Provider_personal users get two independent sets:
// "freelance" for their freelance marketplace identity and
// "referrer" for their apporteur d'affaires identity.
type SocialLinkPersona string

const (
	PersonaFreelance SocialLinkPersona = "freelance"
	PersonaReferrer  SocialLinkPersona = "referrer"
	PersonaAgency    SocialLinkPersona = "agency"
)

// ValidPlatforms lists all accepted social link platform identifiers.
var ValidPlatforms = []string{
	"linkedin",
	"instagram",
	"youtube",
	"twitter",
	"github",
	"website",
}

// ValidPersonas lists all accepted persona identifiers. Kept
// alongside the platform allowlist so both validators live in one
// place and reviews stay concise.
var ValidPersonas = []SocialLinkPersona{
	PersonaFreelance,
	PersonaReferrer,
	PersonaAgency,
}

var (
	ErrInvalidPlatform = errors.New("invalid social link platform")
	ErrInvalidURL      = errors.New("invalid social link URL")
	ErrInvalidPersona  = errors.New("invalid social link persona")
)

// IsValidPlatform checks whether the given platform string is supported.
func IsValidPlatform(platform string) bool {
	lower := strings.ToLower(platform)
	for _, p := range ValidPlatforms {
		if p == lower {
			return true
		}
	}
	return false
}

// IsValidPersona checks whether the given persona string is one of
// the recognised identity scopes.
func IsValidPersona(persona SocialLinkPersona) bool {
	for _, p := range ValidPersonas {
		if p == persona {
			return true
		}
	}
	return false
}

// ssrfDeniedRanges is the explicit denylist of CIDR blocks the
// SocialLink URL validator refuses to resolve into. The blast radius
// of letting a user-supplied URL hit one of these ranges is
// catastrophic on a hosted backend (cloud metadata, internal
// service discovery, the Postgres bind address):
//
//   - 10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16 — RFC1918 private.
//   - 127.0.0.0/8 — loopback IPv4.
//   - 169.254.0.0/16 — link-local IPv4 (AWS/GCP/Azure metadata is
//     169.254.169.254).
//   - 224.0.0.0/4 — IPv4 multicast.
//   - 0.0.0.0/8 — "this network" / unspecified.
//   - 100.64.0.0/10 — Carrier-grade NAT, also used by some clouds for
//     internal-only addressing.
//   - ::1/128 — IPv6 loopback.
//   - fe80::/10 — IPv6 link-local.
//   - fc00::/7 — IPv6 unique-local (RFC 4193 ULA).
//   - ff00::/8 — IPv6 multicast.
//   - ::/128 — IPv6 unspecified.
//
// We compile them once at package init so ValidateSocialURL stays
// allocation-free on the hot path.
var ssrfDeniedRanges []netip.Prefix

func init() {
	for _, cidr := range []string{
		"0.0.0.0/8",
		"10.0.0.0/8",
		"100.64.0.0/10",
		"127.0.0.0/8",
		"169.254.0.0/16",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"224.0.0.0/4",
		"::/128",
		"::1/128",
		"fc00::/7",
		"fe80::/10",
		"ff00::/8",
	} {
		p, err := netip.ParsePrefix(cidr)
		if err != nil {
			panic("profile: invalid SSRF CIDR " + cidr + ": " + err.Error())
		}
		ssrfDeniedRanges = append(ssrfDeniedRanges, p)
	}
}

// hostResolver is the indirection that lets tests inject a fake
// resolver. Production code uses the package-level `net.LookupIP`.
// A nil resolver disables DNS-rebinding mitigation but still rejects
// any host that PARSES as a literal IP in a denied range — useful
// for unit tests that don't want real DNS.
type hostResolver func(host string) ([]net.IP, error)

var defaultHostResolver hostResolver = net.LookupIP

// IsBlockedSocialIP reports whether the given net.IP falls inside any
// of the SSRF-denylist ranges. Exported so adjacent validators
// (referrer, freelance) can reuse the same denylist when they need
// their own URL guards without copying the CIDR table.
func IsBlockedSocialIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	addr, ok := netip.AddrFromSlice(ip.To16())
	if !ok {
		return true
	}
	// Convert IPv4-mapped IPv6 (::ffff:0.0.0.0/96) back to its IPv4
	// form so an IPv4 prefix actually matches. netip's prefix matching
	// is family-strict.
	if addr.Is4In6() {
		addr = addr.Unmap()
	}
	for _, p := range ssrfDeniedRanges {
		if p.Contains(addr) {
			return true
		}
	}
	return false
}

// ValidateSocialURL checks that the URL is well-formed, uses an
// allowed scheme, and that the host does NOT resolve to any IP in
// the SSRF denylist (private/loopback/link-local/cloud-metadata).
//
// SECURITY (SEC-FINAL-04): a naive scheme+host check would let an
// attacker supply `http://169.254.169.254/latest/meta-data/...`
// (AWS metadata) or `http://10.0.0.1/admin` (private network probe)
// or even a hostname under their control whose DNS A record points
// at a denied IP (DNS rebinding). The fix is layered:
//
//  1. Reject schemes other than http/https up-front. This kills
//     `javascript:`, `data:`, `vbscript:`, `file:`, `gopher:` etc.
//  2. Reject any host that PARSES as a literal IP in a denied
//     range — including alternate encodings (decimal `2130706433`,
//     octal `0177.0.0.1`) which net.ParseIP rejects but Go's
//     net.LookupIP normalizes.
//  3. Resolve the host via DNS (production) and reject if ANY of the
//     returned IPs is in the denylist. This mitigates DNS rebinding:
//     even if the first lookup returns a public IP and the next one
//     swaps to 127.0.0.1, the fetch path will re-resolve and we
//     enforce the same check there. Belt + braces.
//
// The check is best-effort: a transient DNS error makes us reject
// (fail-closed) — better to refuse a borderline social URL than to
// risk an SSRF window. Callers that need a softer policy should
// catch the error.
func ValidateSocialURL(rawURL string) error {
	return validateSocialURLWith(rawURL, defaultHostResolver)
}

// validateSocialURLWith is the test seam for ValidateSocialURL. The
// resolver is injected so unit tests don't depend on real DNS. A nil
// resolver skips the rebinding-mitigation step but the literal-IP
// denylist check still runs.
func validateSocialURLWith(rawURL string, resolve hostResolver) error {
	if rawURL == "" {
		return ErrInvalidURL
	}
	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return ErrInvalidURL
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return ErrInvalidURL
	}
	host := parsed.Hostname()
	if host == "" {
		return ErrInvalidURL
	}

	// Empty path with control chars is also bogus — net/url tolerates
	// some of them but they have no place in a public social URL.
	for _, r := range rawURL {
		if r < 0x20 || r == 0x7f {
			return ErrInvalidURL
		}
	}

	// 1. Reject literal IPs in a denied range. We accept multiple
	//    encodings here that net.ParseIP rejects (decimal, octal,
	//    hex) by also trying net.LookupIP later — but we do not want
	//    to wait for DNS for a literal-IP attempt. The cheap check
	//    catches `127.0.0.1`, `[::1]`, `[::ffff:127.0.0.1]`, etc.
	if literal := net.ParseIP(host); literal != nil {
		if IsBlockedSocialIP(literal) {
			return ErrInvalidURL
		}
	}

	// 2. Reject suspicious encodings up-front. Go's `net.ParseIP`
	//    does NOT accept `0177.0.0.1` (octal) or `2130706433`
	//    (decimal-int), but `net.LookupIP` on most platforms does
	//    (it goes through getaddrinfo which still honors them).
	//    Normalize a few obvious cases so we never even ask DNS.
	if blockedEncoding(host) {
		return ErrInvalidURL
	}

	// 3. Production rebinding mitigation: resolve the hostname and
	//    reject if ANY returned IP is in the denylist.
	if resolve != nil {
		ips, err := resolve(host)
		if err != nil {
			// DNS error → fail-closed. Either the user typed a
			// non-existent host or the resolver is hostile; we
			// don't speculate.
			return ErrInvalidURL
		}
		if len(ips) == 0 {
			return ErrInvalidURL
		}
		for _, ip := range ips {
			if IsBlockedSocialIP(ip) {
				return ErrInvalidURL
			}
		}
	}

	return nil
}

// blockedEncoding catches alternate IP encodings that getaddrinfo
// tends to honour even though they have no place in a user-facing
// URL field. Two shapes are handled:
//
//   - Pure decimal integer like `2130706433` (= 127.0.0.1 packed
//     into a uint32). We reject any host that consists only of
//     digits and exceeds 255 (a port-like number could be valid
//     for a sub-octet, but a pure host of more than 3 digits is
//     suspicious enough to refuse outright).
//   - Octal-prefixed dotted quad like `0177.0.0.1` — every label
//     starts with `0` and has a non-zero suffix.
//
// Hex encodings (`0x7f000001`) are caught by the same digit-only
// rule because they include the alphabetic `x` which we reject.
func blockedEncoding(host string) bool {
	if host == "" {
		return true
	}
	// Single all-digit integer with magnitude beyond a 3-digit IP
	// label.
	allDigits := true
	for _, r := range host {
		if r < '0' || r > '9' {
			allDigits = false
			break
		}
	}
	if allDigits && len(host) > 3 {
		return true
	}
	// Hex-encoded IPv4 (`0x...`).
	if strings.HasPrefix(host, "0x") || strings.HasPrefix(host, "0X") {
		return true
	}
	// Octal-prefixed dotted quad. Every label that is two or more
	// chars and starts with `0` is suspicious.
	if strings.Contains(host, ".") {
		for _, label := range strings.Split(host, ".") {
			if len(label) >= 2 && label[0] == '0' {
				return true
			}
		}
	}
	return false
}
