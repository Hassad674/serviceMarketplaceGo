package twofactor

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

// CodeLength is the number of decimal digits in a 2FA email code.
// Six is the industry default (matches every TOTP authenticator app)
// and gives a 10^6 keyspace — combined with 5 attempts and a 10-min
// expiry, the per-challenge guess probability is 5e-6, well under
// the SOC's "one in a million" threshold.
const CodeLength = 6

// codeKeyspace is the upper bound for crypto/rand.Int — the smallest
// integer NOT representable as a 6-digit decimal code. We sample in
// [0, 10^6) and zero-pad on render so every code is exactly six
// digits, including leading-zero codes like "000042". Without the
// pad, the keyspace would silently shrink because a code like "42"
// would be unenterable in a six-character input field.
var codeKeyspace = big.NewInt(1_000_000)

// GenerateCode returns a cryptographically-secure 6-digit decimal
// string suitable for email 2FA. Internally:
//
//  1. crypto/rand.Int over [0, 10^6) — uniform sampling, no modulo
//     bias. Using math/rand here would be a critical bug: the
//     attacker would learn the seed from the timestamp and predict
//     every subsequent code.
//  2. Format with %06d so leading-zero codes are emitted at full
//     length. Six digits gives the user a memorable token without
//     widening the keyspace beyond what email-delivery delays make
//     useful.
//
// The function returns an error only when crypto/rand itself fails
// (which Go's runtime treats as fatal anyway — every modern OS makes
// /dev/urandom always-available). Callers can treat a non-nil error
// as a hard failure and surface a generic 500.
func GenerateCode() (string, error) {
	n, err := rand.Int(rand.Reader, codeKeyspace)
	if err != nil {
		return "", fmt.Errorf("twofactor: generate code: %w", err)
	}
	return fmt.Sprintf("%0*d", CodeLength, n.Int64()), nil
}
