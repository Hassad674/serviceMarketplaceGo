package validator

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"marketplace-backend/internal/domain/user"
)

func DecodeJSON(r *http.Request, dst any) error {
	if r.Body == nil {
		return fmt.Errorf("request body is empty")
	}
	defer r.Body.Close()

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	return nil
}

func ValidateRequired(fields map[string]string) map[string]string {
	errs := make(map[string]string)
	for name, value := range fields {
		if strings.TrimSpace(value) == "" {
			errs[name] = fmt.Sprintf("%s is required", name)
		}
	}
	if len(errs) == 0 {
		return nil
	}
	return errs
}

func ValidateEmail(email string) error {
	_, err := user.NewEmail(email)
	return err
}

func ValidatePassword(password string) error {
	_, err := user.NewPassword(password)
	return err
}

func ValidateRole(role string) error {
	r := user.Role(role)
	if !r.IsValid() {
		return user.ErrInvalidRole
	}
	return nil
}
