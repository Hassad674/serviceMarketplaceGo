package auth

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
)

type RegisterInput struct {
	Email       string
	Password    string
	FirstName   string
	LastName    string
	DisplayName string
	Role        user.Role
}

type LoginInput struct {
	Email    string
	Password string
}

type ForgotPasswordInput struct {
	Email string
}

type ResetPasswordInput struct {
	Token       string
	NewPassword string
}

type AuthOutput struct {
	User         *user.User
	AccessToken  string
	RefreshToken string
}

type Service struct {
	users       repository.UserRepository
	resets      repository.PasswordResetRepository
	hasher      service.HasherService
	tokens      service.TokenService
	email       service.EmailService
	frontendURL string
}

func NewService(
	users repository.UserRepository,
	resets repository.PasswordResetRepository,
	hasher service.HasherService,
	tokens service.TokenService,
	email service.EmailService,
	frontendURL string,
) *Service {
	return &Service{
		users:       users,
		resets:      resets,
		hasher:      hasher,
		tokens:      tokens,
		email:       email,
		frontendURL: frontendURL,
	}
}

func (s *Service) Register(ctx context.Context, input RegisterInput) (*AuthOutput, error) {
	email, err := user.NewEmail(input.Email)
	if err != nil {
		return nil, err
	}

	if _, err := user.NewPassword(input.Password); err != nil {
		return nil, err
	}

	exists, err := s.users.ExistsByEmail(ctx, email.String())
	if err != nil {
		return nil, fmt.Errorf("failed to check email: %w", err)
	}
	if exists {
		return nil, user.ErrEmailAlreadyExists
	}

	hashedPassword, err := s.hasher.Hash(input.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	u, err := user.NewUser(email.String(), hashedPassword, strings.TrimSpace(input.FirstName), strings.TrimSpace(input.LastName), strings.TrimSpace(input.DisplayName), input.Role)
	if err != nil {
		return nil, err
	}

	if err := s.users.Create(ctx, u); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	accessToken, err := s.tokens.GenerateAccessToken(u.ID, u.Role.String(), u.IsAdmin)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := s.tokens.GenerateRefreshToken(u.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return &AuthOutput{
		User:         u,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *Service) Login(ctx context.Context, input LoginInput) (*AuthOutput, error) {
	email, err := user.NewEmail(input.Email)
	if err != nil {
		return nil, user.ErrInvalidCredentials
	}

	u, err := s.users.GetByEmail(ctx, email.String())
	if err != nil {
		return nil, user.ErrInvalidCredentials
	}

	if err := s.hasher.Compare(u.HashedPassword, input.Password); err != nil {
		return nil, user.ErrInvalidCredentials
	}

	if u.IsSuspended() {
		return nil, user.NewSuspendedError(u.SuspensionReason)
	}
	if u.IsBanned() {
		return nil, user.NewBannedError(u.BanReason)
	}

	accessToken, err := s.tokens.GenerateAccessToken(u.ID, u.Role.String(), u.IsAdmin)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := s.tokens.GenerateRefreshToken(u.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return &AuthOutput{
		User:         u,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (*AuthOutput, error) {
	claims, err := s.tokens.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, user.ErrUnauthorized
	}

	u, err := s.users.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, user.ErrUnauthorized
	}

	if u.IsSuspended() {
		return nil, user.NewSuspendedError(u.SuspensionReason)
	}
	if u.IsBanned() {
		return nil, user.NewBannedError(u.BanReason)
	}

	newAccessToken, err := s.tokens.GenerateAccessToken(u.ID, u.Role.String(), u.IsAdmin)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	newRefreshToken, err := s.tokens.GenerateRefreshToken(u.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return &AuthOutput{
		User:         u,
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
	}, nil
}

func (s *Service) GetMe(ctx context.Context, userID uuid.UUID) (*user.User, error) {
	return s.users.GetByID(ctx, userID)
}

func (s *Service) EnableReferrer(ctx context.Context, userID uuid.UUID) (*user.User, error) {
	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("enable referrer: get user: %w", err)
	}

	if u.Role != user.RoleProvider {
		return nil, user.ErrInvalidRole
	}

	u.EnableReferrer()
	u.UpdatedAt = time.Now()

	if err := s.users.Update(ctx, u); err != nil {
		return nil, fmt.Errorf("enable referrer: update user: %w", err)
	}

	return u, nil
}

func (s *Service) ForgotPassword(ctx context.Context, input ForgotPasswordInput) error {
	email, err := user.NewEmail(input.Email)
	if err != nil {
		return nil // Don't reveal if email exists
	}

	u, err := s.users.GetByEmail(ctx, email.String())
	if err != nil {
		return nil // Don't reveal if email exists
	}

	token := uuid.New().String()
	pr := &repository.PasswordReset{
		ID:        uuid.New(),
		UserID:    u.ID,
		Token:     token,
		ExpiresAt: time.Now().Add(1 * time.Hour),
		CreatedAt: time.Now(),
	}

	if err := s.resets.Create(ctx, pr); err != nil {
		return fmt.Errorf("create reset token: %w", err)
	}

	resetURL := s.frontendURL + "/reset-password?token=" + token
	if err := s.email.SendPasswordReset(ctx, u.Email, resetURL); err != nil {
		return fmt.Errorf("send reset email: %w", err)
	}

	return nil
}

func (s *Service) ResetPassword(ctx context.Context, input ResetPasswordInput) error {
	if _, err := user.NewPassword(input.NewPassword); err != nil {
		return err
	}

	pr, err := s.resets.GetByToken(ctx, input.Token)
	if err != nil {
		return user.ErrUnauthorized
	}

	if pr.Used || pr.ExpiresAt.Before(time.Now()) {
		return user.ErrUnauthorized
	}

	hashedPassword, err := s.hasher.Hash(input.NewPassword)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	u, err := s.users.GetByID(ctx, pr.UserID)
	if err != nil {
		return fmt.Errorf("get user: %w", err)
	}

	u.HashedPassword = hashedPassword
	if err := s.users.Update(ctx, u); err != nil {
		return fmt.Errorf("update password: %w", err)
	}

	if err := s.resets.MarkUsed(ctx, pr.ID); err != nil {
		return fmt.Errorf("mark token used: %w", err)
	}

	return nil
}
