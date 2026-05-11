package handler

import (
	"strings"
)

// ParsedUserAgent is the result of running a raw User-Agent header
// through the inline parser. It is intentionally tiny — we only need
// what the Malt-style "Sécurité" page renders:
//
//   - Label   : "Ordinateur de bureau (Chrome)" / "iPhone (Safari)" / ...
//   - Browser : "Chrome" / "Safari" / "Firefox" / "Edge" / "Opera"
//   - OS      : "Windows" / "macOS" / "Linux" / "iOS" / "Android"
//
// Unparseable / bot strings fall back to the neutral
// "Appareil inconnu" label with empty Browser + OS, so callers can
// distinguish "we tried and failed" from "we did not look".
//
// We do NOT add a third-party dependency for this — the parser is
// 60 lines of allow-listed substring matches, easy to audit, and
// covers the realistic browsers that hit a B2B marketplace
// (Chrome / Safari / Firefox / Edge / Opera on Windows / macOS / Linux /
// iOS / Android). Anything outside that set is correctly bucketed as
// "Appareil inconnu" which is the same fallback Malt itself uses.
type ParsedUserAgent struct {
	Label   string
	Browser string
	OS      string
}

// UnknownDeviceLabel is the fallback shown on the Sécurité row when
// the User-Agent string is empty / unparseable / clearly a bot. Kept
// public so tests and the web layer can assert against it without
// duplicating the literal.
const UnknownDeviceLabel = "Appareil inconnu"

// ParseUserAgent runs the raw header through the inline matcher and
// returns a ParsedUserAgent. Empty / whitespace-only input returns the
// fallback. The function is allocation-light (a single ToLower + a
// handful of Contains checks) and safe to call on every login.
func ParseUserAgent(raw string) ParsedUserAgent {
	ua := strings.TrimSpace(raw)
	if ua == "" {
		return ParsedUserAgent{Label: UnknownDeviceLabel}
	}
	lc := strings.ToLower(ua)

	// Bots get the neutral label — they are a tiny fraction of real
	// logins (mostly health checks and accidental crawlers) so we do
	// NOT try to render a bot-specific row.
	if looksLikeBot(lc) {
		return ParsedUserAgent{Label: UnknownDeviceLabel}
	}

	browser := detectBrowser(lc)
	os, family := detectOSAndFamily(lc)

	// Build the Malt-style "<family> (<browser>)" label. If either part
	// is missing fall back to the neutral label so we never render
	// half-baked strings like "(Safari)" or "iPhone ()".
	if family == "" || browser == "" {
		return ParsedUserAgent{Label: UnknownDeviceLabel, Browser: browser, OS: os}
	}
	return ParsedUserAgent{
		Label:   family + " (" + browser + ")",
		Browser: browser,
		OS:      os,
	}
}

// looksLikeBot keeps the bot guard in one place so the test surface is
// trivial. The list is deliberately short — false positives here would
// drop legitimate users into the unknown bucket.
func looksLikeBot(lc string) bool {
	for _, needle := range []string{
		"bot",
		"crawler",
		"spider",
		"curl/",
		"wget/",
		"python-requests",
		"go-http-client",
	} {
		if strings.Contains(lc, needle) {
			return true
		}
	}
	return false
}

// detectBrowser returns the canonical browser name. Order matters:
// Edge / Opera / Chrome / Safari overlap in tokens so the most
// specific match must come first.
func detectBrowser(lc string) string {
	switch {
	case strings.Contains(lc, "edg/"), strings.Contains(lc, "edge/"):
		return "Edge"
	case strings.Contains(lc, "opr/"), strings.Contains(lc, "opera/"):
		return "Opera"
	case strings.Contains(lc, "firefox/"):
		return "Firefox"
	case strings.Contains(lc, "chrome/"):
		// Chrome on iOS reports as "CriOS" but also still contains
		// "chrome/" most of the time — the Edge / Opera branches above
		// already ate the false positives.
		return "Chrome"
	case strings.Contains(lc, "crios/"):
		return "Chrome"
	case strings.Contains(lc, "fxios/"):
		return "Firefox"
	case strings.Contains(lc, "safari/"):
		// Safari token appears in lots of UAs (it is the base WebKit
		// string), so this branch is intentionally last.
		return "Safari"
	default:
		return ""
	}
}

// detectOSAndFamily resolves the OS name AND the user-facing device
// family ("iPhone", "iPad", "Android", "Ordinateur de bureau") in one
// pass so the two stay in lock-step.
func detectOSAndFamily(lc string) (osName string, family string) {
	switch {
	case strings.Contains(lc, "iphone"):
		return "iOS", "iPhone"
	case strings.Contains(lc, "ipad"):
		return "iOS", "iPad"
	case strings.Contains(lc, "android"):
		// Android tablets vs phones are hard to disambiguate from the
		// UA alone (the "Mobile" token is sometimes absent on tablets,
		// sometimes present on phones). Bucket all of Android under
		// the generic "Android" family — the icon picker on the web
		// already picks "smartphone" for it.
		return "Android", "Android"
	case strings.Contains(lc, "windows"):
		return "Windows", "Ordinateur de bureau"
	case strings.Contains(lc, "mac os x"), strings.Contains(lc, "macintosh"):
		return "macOS", "Ordinateur de bureau"
	case strings.Contains(lc, "linux"):
		// Most Linux desktop UAs include "x11" too — keep the match on
		// "linux" alone so headless desktop browsers still resolve.
		return "Linux", "Ordinateur de bureau"
	default:
		return "", ""
	}
}
