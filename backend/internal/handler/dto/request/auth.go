package request

// RegisterRequest is the body of POST /api/v1/auth/register. The role
// field accepts only `agency`, `enterprise`, `provider`. Names are
// capped at 100 chars; email max 254 (RFC 5321); password follows the
// domain `NewPassword` rule (validated server-side after decoding).
type RegisterRequest struct {
	Email       string `json:"email" validate:"required,email,max=254"`
	Password    string `json:"password" validate:"required,min=8,max=128"`
	FirstName   string `json:"first_name" validate:"omitempty,max=100"`
	LastName    string `json:"last_name" validate:"omitempty,max=100"`
	DisplayName string `json:"display_name" validate:"omitempty,max=100"`
	Role        string `json:"role" validate:"required,oneof=agency enterprise provider"`
}

// LoginRequest is the body of POST /api/v1/auth/login.
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email,max=254"`
	Password string `json:"password" validate:"required,min=1,max=128"`
}

// RefreshRequest is the body of POST /api/v1/auth/refresh. The
// refresh_token is opaque to the handler — its length cap protects
// against DoS via a 10MB JSON payload claiming to be a refresh token.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required,min=1,max=4096"`
}
