package security

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseUserAgent_Empty(t *testing.T) {
	got := ParseUserAgent("")
	assert.Equal(t, "", got.Display)
	assert.Equal(t, AccessKindUnknown, got.Kind)
}

func TestParseUserAgent_DesktopChrome(t *testing.T) {
	ua := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.6099.71 Safari/537.36"
	got := ParseUserAgent(ua)
	assert.Equal(t, AccessKindDesktop, got.Kind)
	assert.Equal(t, "Ordinateur (Chrome 120)", got.Display)
}

func TestParseUserAgent_MobileSafari(t *testing.T) {
	ua := "Mozilla/5.0 (iPhone; CPU iPhone OS 16_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.5 Mobile/15E148 Safari/604.1"
	got := ParseUserAgent(ua)
	assert.Equal(t, AccessKindMobile, got.Kind)
	assert.Equal(t, "Mobile (Safari 16)", got.Display)
}

func TestParseUserAgent_TabletIPad(t *testing.T) {
	ua := "Mozilla/5.0 (iPad; CPU OS 16_5 like Mac OS X) AppleWebKit/605.1.15 Version/16.5 Safari/604.1"
	got := ParseUserAgent(ua)
	assert.Equal(t, AccessKindTablet, got.Kind)
	assert.Contains(t, got.Display, "Tablette")
}

func TestParseUserAgent_Edge(t *testing.T) {
	ua := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 Edg/120.0.2210.61"
	got := ParseUserAgent(ua)
	assert.Equal(t, AccessKindDesktop, got.Kind)
	assert.Equal(t, "Ordinateur (Edge 120)", got.Display)
}

func TestParseUserAgent_Firefox(t *testing.T) {
	ua := "Mozilla/5.0 (X11; Linux x86_64; rv:121.0) Gecko/20100101 Firefox/121.0"
	got := ParseUserAgent(ua)
	assert.Equal(t, AccessKindDesktop, got.Kind)
	assert.Equal(t, "Ordinateur (Firefox 121)", got.Display)
}

func TestParseUserAgent_AndroidMobile(t *testing.T) {
	ua := "Mozilla/5.0 (Linux; Android 14; Pixel 8) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.6099.43 Mobile Safari/537.36"
	got := ParseUserAgent(ua)
	assert.Equal(t, AccessKindMobile, got.Kind)
	assert.Equal(t, "Mobile (Chrome 120)", got.Display)
}

func TestParseUserAgent_AndroidTablet(t *testing.T) {
	// Android UA without the "Mobile" token → tablet form factor.
	ua := "Mozilla/5.0 (Linux; Android 13; Tab S9) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	got := ParseUserAgent(ua)
	assert.Equal(t, AccessKindTablet, got.Kind)
}

func TestParseUserAgent_Unknown(t *testing.T) {
	got := ParseUserAgent("curl/8.4.0")
	assert.Equal(t, AccessKindUnknown, got.Kind)
	assert.Equal(t, "Appareil inconnu", got.Display)
}

func TestParseUserAgent_NoVersionFallsBackToName(t *testing.T) {
	// Chrome marker without trailing digits — version parser should
	// return the bare browser name instead of a malformed label.
	ua := "Mozilla/5.0 Linux Chrome/abc Safari/537.36"
	got := ParseUserAgent(ua)
	assert.Contains(t, got.Display, "Chrome")
	assert.NotContains(t, got.Display, "Chrome ")
}
