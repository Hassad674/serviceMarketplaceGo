package user

import (
	"regexp"
	"strings"
	"unicode"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// Email is a validated email value object.
type Email struct {
	value string
}

func NewEmail(raw string) (Email, error) {
	trimmed := strings.TrimSpace(strings.ToLower(raw))
	if trimmed == "" || !emailRegex.MatchString(trimmed) {
		return Email{}, ErrInvalidEmail
	}
	return Email{value: trimmed}, nil
}

func (e Email) String() string {
	return e.value
}

// Password is a validated password value object.
type Password struct {
	value string
}

// MinPasswordLength is the minimum acceptable password length.
//
// Bumped from 8 to 10 in Phase 1 (SEC-20). Combined with the
// uppercase + lowercase + digit + special-character requirement
// this puts a brute-forced password well outside the offline
// dictionary attack horizon while still being typeable on a
// mobile keyboard. The web/mobile UI announce this rule to the
// user before they submit so a rejection is never a surprise.
const MinPasswordLength = 10

// isPasswordSpecial reports whether c counts as a special character
// for the password policy. We accept any unicode punctuation or
// symbol — that covers `!@#$%^&*()` and friends without forcing the
// user onto an ASCII-only keyboard.
func isPasswordSpecial(c rune) bool {
	return unicode.IsPunct(c) || unicode.IsSymbol(c)
}

func NewPassword(raw string) (Password, error) {
	if len(raw) < MinPasswordLength {
		return Password{}, ErrWeakPassword
	}

	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, c := range raw {
		switch {
		case unicode.IsUpper(c):
			hasUpper = true
		case unicode.IsLower(c):
			hasLower = true
		case unicode.IsDigit(c):
			hasDigit = true
		case isPasswordSpecial(c):
			hasSpecial = true
		}
	}

	if !hasUpper || !hasLower || !hasDigit || !hasSpecial {
		return Password{}, ErrWeakPassword
	}

	return Password{value: raw}, nil
}

func (p Password) String() string {
	return p.value
}
