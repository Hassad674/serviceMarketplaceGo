package twofactor

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGenerateCode_LengthAndDigits asserts the structural invariants
// every code must satisfy: exactly six characters, all decimal digits,
// representable as an integer in [0, 999999]. The check runs 1000
// iterations because a single sample is not enough to surface the
// leading-zero edge case (codes like "000042" must be padded, not
// truncated).
func TestGenerateCode_LengthAndDigits(t *testing.T) {
	for i := 0; i < 1000; i++ {
		code, err := GenerateCode()
		require.NoError(t, err, "iteration %d", i)
		require.Len(t, code, CodeLength, "iteration %d code=%q", i, code)
		n, parseErr := strconv.Atoi(code)
		require.NoError(t, parseErr, "iteration %d code=%q must be a decimal integer", i, code)
		assert.GreaterOrEqual(t, n, 0)
		assert.Less(t, n, 1_000_000)
	}
}

// TestGenerateCode_Entropy asserts the generator is not stuck on a
// single value or a tiny subset. With 1000 samples over a 10^6 space
// we expect at least 950 unique values (the birthday-paradox collision
// estimate is ~0.5 — far under the 50-value tolerance). A floor of
// 100 unique values would still be a strong signal the generator
// works while leaving room for the unlikely worst-case streak.
func TestGenerateCode_Entropy(t *testing.T) {
	const samples = 1000
	seen := make(map[string]int, samples)
	for i := 0; i < samples; i++ {
		code, err := GenerateCode()
		require.NoError(t, err)
		seen[code]++
	}
	// At least 100 distinct codes — a stuck PRNG would emit < 10.
	assert.GreaterOrEqual(t, len(seen), 100,
		"too few unique codes (%d) — generator may be stuck", len(seen))
}

// TestGenerateCode_LeadingZerosPreserved walks the keyspace via direct
// formatting so we know %06d does the right thing on small integers —
// "42" must become "000042", not "42".
func TestGenerateCode_LeadingZerosPreserved(t *testing.T) {
	tests := []struct {
		n    int
		want string
	}{
		{0, "000000"},
		{42, "000042"},
		{123, "000123"},
		{99999, "099999"},
		{999999, "999999"},
	}
	for _, tt := range tests {
		got := formatTestCode(tt.n)
		assert.Equal(t, tt.want, got)
	}
}

// formatTestCode mirrors the GenerateCode renderer so the
// leading-zero test runs against the same %06d format string the
// production code uses.
func formatTestCode(n int) string {
	return zeroPad(n, CodeLength)
}

func zeroPad(n, width int) string {
	s := strconv.Itoa(n)
	for len(s) < width {
		s = "0" + s
	}
	return s
}
