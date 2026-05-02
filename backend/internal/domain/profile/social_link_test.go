package profile

import (
	"errors"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsValidPlatform(t *testing.T) {
	tests := []struct {
		name     string
		platform string
		want     bool
	}{
		{"linkedin lowercase", "linkedin", true},
		{"LinkedIn mixed case", "LinkedIn", true},
		{"instagram", "instagram", true},
		{"youtube", "youtube", true},
		{"twitter", "twitter", true},
		{"github", "github", true},
		{"website", "website", true},
		{"WEBSITE uppercase", "WEBSITE", true},
		{"empty string", "", false},
		{"unknown platform", "tiktok", false},
		{"facebook", "facebook", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidPlatform(tt.platform)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsValidPersona(t *testing.T) {
	tests := []struct {
		name    string
		persona SocialLinkPersona
		want    bool
	}{
		{"freelance", PersonaFreelance, true},
		{"referrer", PersonaReferrer, true},
		{"agency", PersonaAgency, true},
		{"empty string", SocialLinkPersona(""), false},
		{"unknown", SocialLinkPersona("admin"), false},
		{"case sensitive", SocialLinkPersona("FREELANCE"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsValidPersona(tt.persona))
		})
	}
}

// publicIPResolver is a hostResolver fake that returns a routable
// public IP for any host. Used by the "valid URL" cases so the test
// suite does not depend on real DNS.
func publicIPResolver(_ string) ([]net.IP, error) {
	return []net.IP{net.ParseIP("93.184.216.34")}, nil // example.com
}

// privateIPResolver mimics a hostile / mis-configured resolver that
// hands back an IP in the denylist. This is the DNS-rebinding case
// the validator must catch.
func privateIPResolverFor(ipStr string) hostResolver {
	return func(_ string) ([]net.IP, error) {
		return []net.IP{net.ParseIP(ipStr)}, nil
	}
}

// dnsErrorResolver simulates an NXDOMAIN-style failure. The validator
// must fail-closed.
func dnsErrorResolver(_ string) ([]net.IP, error) {
	return nil, errors.New("no such host")
}

func TestValidateSocialURL_BasicShape(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr error
	}{
		{"valid https url", "https://linkedin.com/in/user", nil},
		{"valid http url", "http://example.com", nil},
		{"empty string", "", ErrInvalidURL},
		{"no scheme", "linkedin.com/in/user", ErrInvalidURL},
		{"javascript scheme", "javascript:alert(1)", ErrInvalidURL},
		{"ftp scheme", "ftp://files.example.com", ErrInvalidURL},
		{"data uri", "data:text/html,<h1>hi</h1>", ErrInvalidURL},
		{"file scheme", "file:///etc/passwd", ErrInvalidURL},
		{"vbscript scheme", "vbscript:msgbox(1)", ErrInvalidURL},
		{"gopher scheme", "gopher://example.com/", ErrInvalidURL},
		{"no host", "https://", ErrInvalidURL},
		{"control chars in url", "https://example.com/\x00path", ErrInvalidURL},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSocialURLWith(tt.url, publicIPResolver)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidateSocialURL_RejectsSSRFVectors covers SEC-FINAL-04. Each
// case represents a real-world SSRF probe attempted via a profile
// social link. They MUST all return ErrInvalidURL.
//
// Stash demonstration: every sub-test FAILS on origin/main where
// ValidateSocialURL only checks scheme + non-empty host.
func TestValidateSocialURL_RejectsSSRFVectors(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{"loopback IPv4 literal", "http://127.0.0.1/admin"},
		{"loopback IPv4 alt /8", "http://127.5.5.5/admin"},
		{"loopback IPv6 literal", "http://[::1]:8080/admin"},
		{"private 10.0.0.0/8", "http://10.0.0.1/internal"},
		{"private 172.16.0.0/12", "http://172.18.0.1/internal"},
		{"private 192.168.0.0/16", "http://192.168.1.1/router"},
		{"AWS metadata 169.254.169.254", "http://169.254.169.254/latest/meta-data/iam/security-credentials"},
		{"GCP metadata via name (denied resolved IP)", "http://metadata.google.internal/computeMetadata/v1/"},
		{"link-local 169.254/16", "http://169.254.42.42/"},
		{"multicast 224.0.0.0/4", "http://224.0.0.1/"},
		{"unspecified 0.0.0.0/8", "http://0.0.0.0/"},
		{"IPv6 link-local fe80::/10", "http://[fe80::1]/"},
		{"IPv6 ULA fc00::/7", "http://[fc00::1]/"},
		{"IPv6 multicast ff00::/8", "http://[ff02::1]/"},
		{"IPv6 unspecified ::", "http://[::]/"},
		{"IPv4-mapped IPv6 to loopback", "http://[::ffff:127.0.0.1]/"},
		{"decimal-encoded loopback (2130706433)", "http://2130706433/"},
		{"octal-encoded loopback (0177.0.0.1)", "http://0177.0.0.1/"},
		{"hex-encoded loopback (0x7f000001)", "http://0x7f000001/"},
		{"CGNAT 100.64.0.0/10", "http://100.64.5.5/"},
	}
	// For test cases that require DNS resolution (named hosts or
	// non-literal IPs), pretend the resolver answers with the most
	// hostile possible IP — 127.0.0.1. The validator must still
	// reject every case because either the literal-IP check or the
	// resolved-IP check (or both) fires.
	hostile := privateIPResolverFor("127.0.0.1")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSocialURLWith(tt.url, hostile)
			assert.ErrorIs(t, err, ErrInvalidURL,
				"SSRF vector %q must be rejected", tt.url)
		})
	}
}

func TestValidateSocialURL_DNSRebindingMitigation(t *testing.T) {
	// A perfectly innocent-looking hostname that resolves to a
	// denied IP must be rejected via the rebinding check.
	denied := []string{
		"127.0.0.1", "10.0.0.1", "192.168.1.1", "169.254.169.254",
		"::1", "fe80::1",
	}
	for _, target := range denied {
		t.Run("rebind_to_"+target, func(t *testing.T) {
			err := validateSocialURLWith(
				"https://harmless.example.com/", privateIPResolverFor(target),
			)
			assert.ErrorIs(t, err, ErrInvalidURL)
		})
	}
}

func TestValidateSocialURL_DNSErrorFailsClosed(t *testing.T) {
	err := validateSocialURLWith(
		"https://nxdomain.invalid/", dnsErrorResolver,
	)
	assert.ErrorIs(t, err, ErrInvalidURL)
}

func TestValidateSocialURL_AcceptsPublicIPs(t *testing.T) {
	resolver := func(_ string) ([]net.IP, error) {
		return []net.IP{
			net.ParseIP("8.8.8.8"),
			net.ParseIP("2001:4860:4860::8888"),
		}, nil
	}
	err := validateSocialURLWith("https://dns.google/", resolver)
	assert.NoError(t, err)
}

// TestValidateSocialURL_RebindingFirstPublicThenPrivate covers the
// classic rebinding scenario where the resolver hands back BOTH a
// public and a private IP. The validator must reject because we
// sweep every returned IP, not just the first.
func TestValidateSocialURL_RebindingFirstPublicThenPrivate(t *testing.T) {
	resolver := func(_ string) ([]net.IP, error) {
		return []net.IP{
			net.ParseIP("93.184.216.34"), // public
			net.ParseIP("127.0.0.1"),     // poisoned
		}, nil
	}
	err := validateSocialURLWith("https://example.com/", resolver)
	assert.ErrorIs(t, err, ErrInvalidURL)
}

func TestIsBlockedSocialIP(t *testing.T) {
	tests := []struct {
		name string
		ip   string
		want bool
	}{
		{"loopback v4", "127.0.0.1", true},
		{"loopback v6", "::1", true},
		{"private 10.x", "10.0.0.1", true},
		{"private 172.16.x", "172.16.0.1", true},
		{"private 192.168.x", "192.168.1.1", true},
		{"link-local v4", "169.254.169.254", true},
		{"multicast", "224.0.0.1", true},
		{"unspecified v4", "0.0.0.0", true},
		{"public v4", "8.8.8.8", false},
		{"public v6", "2001:4860:4860::8888", false},
		{"link-local v6", "fe80::1", true},
		{"ULA v6", "fc00::1", true},
		{"multicast v6", "ff02::1", true},
		{"4-in-6 loopback", "::ffff:127.0.0.1", true},
		{"4-in-6 public", "::ffff:8.8.8.8", false},
		{"CGNAT", "100.64.5.5", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			assert.Equal(t, tt.want, IsBlockedSocialIP(ip),
				"IsBlockedSocialIP(%s)", tt.ip)
		})
	}

	t.Run("nil IP is denied", func(t *testing.T) {
		assert.True(t, IsBlockedSocialIP(nil))
	})
}
