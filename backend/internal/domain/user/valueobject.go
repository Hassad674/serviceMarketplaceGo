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

func NewPassword(raw string) (Password, error) {
	if len(raw) < 8 {
		return Password{}, ErrWeakPassword
	}

	var hasUpper, hasLower, hasDigit bool
	for _, c := range raw {
		switch {
		case unicode.IsUpper(c):
			hasUpper = true
		case unicode.IsLower(c):
			hasLower = true
		case unicode.IsDigit(c):
			hasDigit = true
		}
	}

	if !hasUpper || !hasLower || !hasDigit {
		return Password{}, ErrWeakPassword
	}

	return Password{value: raw}, nil
}

func (p Password) String() string {
	return p.value
}
