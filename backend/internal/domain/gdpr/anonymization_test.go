package gdpr

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashEmail_DeterministicAndCaseInsensitive(t *testing.T) {
	h1, err := HashEmail("Jean@Example.com", "salt-1")
	require.NoError(t, err)

	h2, err := HashEmail("jean@example.com", "salt-1")
	require.NoError(t, err)

	h3, err := HashEmail("  jean@example.com  ", "salt-1")
	require.NoError(t, err)

	assert.Equal(t, h1, h2, "case should not change the digest")
	assert.Equal(t, h1, h3, "leading/trailing whitespace should be trimmed")
	assert.Len(t, h1, 64, "sha256 hex is 64 chars")
}

func TestHashEmail_DifferentSaltsProduceDifferentDigests(t *testing.T) {
	h1, err := HashEmail("alice@example.com", "salt-A")
	require.NoError(t, err)

	h2, err := HashEmail("alice@example.com", "salt-B")
	require.NoError(t, err)

	assert.NotEqual(t, h1, h2, "salt rotation must defeat dictionary attacks")
}

func TestHashEmail_RejectsEmptySalt(t *testing.T) {
	_, err := HashEmail("alice@example.com", "")
	assert.ErrorIs(t, err, ErrSaltRequired)
}

func TestHashEmail_EmptyEmailStillHashes(t *testing.T) {
	// An empty email is unusual but the audit row may have lost
	// the actor_email key — we still want a stable digest of just
	// the salt for the filler, never an error.
	h, err := HashEmail("", "salt-1")
	require.NoError(t, err)
	assert.Len(t, h, 64)
}

func TestTruncateIP_IPv4DropsLastTwoOctets(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"203.0.113.42", "203.0.x.x"},
		{"10.20.30.40", "10.20.x.x"},
		{"255.255.255.255", "255.255.x.x"},
		{"1.2.3.4", "1.2.x.x"},
		{"  192.168.0.1  ", "192.168.x.x"},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			got := TruncateIP(tc.in)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestTruncateIP_IPv6KeepsNetworkPrefix(t *testing.T) {
	got := TruncateIP("2001:db8::1234")
	// Should keep the /32 prefix and zero the rest.
	assert.True(t, strings.HasPrefix(got, "2001:"), "got %q", got)
	assert.NotContains(t, got, "1234", "device-specific bits must be dropped")
}

func TestTruncateIP_PassesThroughOnInvalidInput(t *testing.T) {
	assert.Equal(t, "not-an-ip", TruncateIP("not-an-ip"))
}

func TestTruncateIP_EmptyReturnsEmpty(t *testing.T) {
	assert.Equal(t, "", TruncateIP(""))
}
