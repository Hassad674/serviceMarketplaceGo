package response

import (
	"marketplace-backend/internal/domain/user"
)

type UserResponse struct {
	ID              string `json:"id"`
	Email           string `json:"email"`
	FirstName       string `json:"first_name"`
	LastName        string `json:"last_name"`
	DisplayName     string `json:"display_name"`
	Role            string `json:"role"`
	ReferrerEnabled bool   `json:"referrer_enabled"`
	EmailVerified   bool   `json:"email_verified"`
	CreatedAt       string `json:"created_at"`
}

type AuthResponse struct {
	User         UserResponse `json:"user"`
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
}

func NewUserResponse(u *user.User) UserResponse {
	return UserResponse{
		ID:              u.ID.String(),
		Email:           u.Email,
		FirstName:       u.FirstName,
		LastName:        u.LastName,
		DisplayName:     u.DisplayName,
		Role:            u.Role.String(),
		ReferrerEnabled: u.ReferrerEnabled,
		EmailVerified:   u.EmailVerified,
		CreatedAt:       u.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

func NewAuthResponse(u *user.User, accessToken, refreshToken string) AuthResponse {
	return AuthResponse{
		User:         NewUserResponse(u),
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}
}
