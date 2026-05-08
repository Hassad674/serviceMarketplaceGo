// Package security exposes the read-only "security activity" use case
// for the account/security tab in the marketplace UI.
//
// The package owns three concerns: filtering audit_logs to authentication
// events attributable to a single user, parsing user-agent strings into a
// short display label ("Ordinateur de bureau (Chrome 120)"), and shaping
// the result into a paginated DTO for the handler.
//
// Removable: deleting this directory + the wiring lines in
// cmd/api/wire_security.go and the route registration in
// internal/handler/routes_security.go takes the feature off the API
// without affecting any other package — audit_logs is read-only here.
package security

import (
	"strings"
)

// AccessKind is a coarse classification of the device behind a user-agent
// string. It is intentionally narrow — three buckets cover the vast
// majority of real traffic and the page only needs an icon hint.
type AccessKind string

const (
	AccessKindDesktop AccessKind = "desktop"
	AccessKindMobile  AccessKind = "mobile"
	AccessKindTablet  AccessKind = "tablet"
	AccessKindUnknown AccessKind = "unknown"
)

// UserAgentSummary is the compact representation the handler returns.
// Display is the short label ("Mobile (Safari)"), Kind is the device
// bucket the UI uses to pick an icon. Both fall back to neutral
// values when parsing yields no signal.
type UserAgentSummary struct {
	Display string
	Kind    AccessKind
}

// ParseUserAgent extracts a short, locale-neutral label from a raw
// user-agent string. Returns ("", AccessKindUnknown) when the input is
// empty so the handler can render a "—" placeholder.
//
// The parser is intentionally a small allow-list rather than a full
// UA library — we only care about three device buckets and a handful
// of browsers, and a third-party UA parser is a steady source of CVEs
// and update churn we can avoid. The few branches below cover Chrome,
// Safari, Firefox, Edge, Opera, plus the matching device bucket.
//
// Returns the raw input prefixed with the kind label as a fallback so
// the audit row never disappears from the UI even on novel agents.
func ParseUserAgent(raw string) UserAgentSummary {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return UserAgentSummary{Display: "", Kind: AccessKindUnknown}
	}

	kind := classifyDevice(trimmed)
	browser := classifyBrowser(trimmed)
	deviceLabel := deviceLabelFor(kind)

	display := deviceLabel
	if browser != "" {
		display = deviceLabel + " (" + browser + ")"
	}
	return UserAgentSummary{Display: display, Kind: kind}
}

// classifyDevice maps the user-agent to one of the four AccessKind
// buckets. Order matters: tablets must be checked before "mobile"
// because iPads include "Macintosh" and Android tablets include
// "Mobile" only when they are phones.
func classifyDevice(ua string) AccessKind {
	lower := strings.ToLower(ua)
	switch {
	case strings.Contains(lower, "ipad"):
		return AccessKindTablet
	case strings.Contains(lower, "tablet"):
		return AccessKindTablet
	case strings.Contains(lower, "android") && !strings.Contains(lower, "mobile"):
		return AccessKindTablet
	case strings.Contains(lower, "mobile"),
		strings.Contains(lower, "iphone"),
		strings.Contains(lower, "android"):
		return AccessKindMobile
	case strings.Contains(lower, "windows"),
		strings.Contains(lower, "macintosh"),
		strings.Contains(lower, "linux"),
		strings.Contains(lower, "x11"):
		return AccessKindDesktop
	}
	return AccessKindUnknown
}

// classifyBrowser returns the human-readable browser label, optionally
// suffixed with the major version when the UA exposes one. The order
// is critical because most browsers identify as "Chrome" or "Safari"
// in their UA — Edge before Chrome, Chrome before Safari, etc.
func classifyBrowser(ua string) string {
	lower := strings.ToLower(ua)
	switch {
	case strings.Contains(lower, "edg/"):
		return labelWithVersion("Edge", ua, "Edg/")
	case strings.Contains(lower, "opr/"), strings.Contains(lower, "opera"):
		return labelWithVersion("Opera", ua, "OPR/")
	case strings.Contains(lower, "firefox/"):
		return labelWithVersion("Firefox", ua, "Firefox/")
	case strings.Contains(lower, "chrome/"):
		return labelWithVersion("Chrome", ua, "Chrome/")
	case strings.Contains(lower, "safari/") && strings.Contains(lower, "version/"):
		return labelWithVersion("Safari", ua, "Version/")
	}
	return ""
}

// labelWithVersion returns "Name 120" if the user-agent has a token
// "Prefix120.0.6099.71"; otherwise it returns just the name. We keep
// the major version only — minor/patch numbers add noise without
// value to a security activity log.
func labelWithVersion(name, ua, prefix string) string {
	idx := strings.Index(ua, prefix)
	if idx < 0 {
		return name
	}
	tail := ua[idx+len(prefix):]
	end := 0
	for end < len(tail) && (tail[end] >= '0' && tail[end] <= '9') {
		end++
	}
	if end == 0 {
		return name
	}
	return name + " " + tail[:end]
}

// deviceLabelFor returns the FR-leaning device bucket label used as
// the prefix in the summary string. Localisation happens client-side
// via the `kind` field; the display string here is a sensible default
// for the FR audience.
func deviceLabelFor(kind AccessKind) string {
	switch kind {
	case AccessKindDesktop:
		return "Ordinateur"
	case AccessKindMobile:
		return "Mobile"
	case AccessKindTablet:
		return "Tablette"
	}
	return "Appareil inconnu"
}
