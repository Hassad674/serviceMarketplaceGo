package gdpr

import (
	"errors"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
)

// HS256Signer signs + parses HS256 JWTs with a static secret. Used by
// the GDPR feature for the deletion-confirmation token.
//
// Why a dedicated signer: the access-token JWT and the deletion JWT
// must NOT share signing keys. A leaked access token MUST NOT let an
// attacker forge a deletion-confirmation link. The wire helper
// reuses cfg.JWTSecret + a salt to derive a separate signing key —
// see wire_gdpr.go.
type HS256Signer struct {
	secret []byte
}

// NewHS256Signer builds the signer. Returns an error if the secret
// is empty so the caller (boot path) can fail fast in production.
func NewHS256Signer(secret string) (*HS256Signer, error) {
	if secret == "" {
		return nil, errors.New("gdpr signer: empty secret")
	}
	return &HS256Signer{secret: []byte(secret)}, nil
}

// Sign signs the claims with HS256 and returns the compact form.
func (s *HS256Signer) Sign(claims jwt.Claims) (string, error) {
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := t.SignedString(s.secret)
	if err != nil {
		return "", fmt.Errorf("gdpr sign: %w", err)
	}
	return signed, nil
}

// Parse parses + verifies the token. Returns an error when the
// signature is invalid or the alg is anything other than HS256
// (defends against the alg=none + RS-as-HS confusion).
func (s *HS256Signer) Parse(token string, claims jwt.Claims) error {
	_, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.secret, nil
	})
	if err != nil {
		return fmt.Errorf("gdpr parse: %w", err)
	}
	return nil
}
