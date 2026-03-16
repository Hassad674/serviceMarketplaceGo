package auth

import (
	"context"
	"fmt"
	"strings"

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

type AuthOutput struct {
	User         *user.User
	AccessToken  string
	RefreshToken string
}

type Service struct {
	users  repository.UserRepository
	hasher service.HasherService
	tokens service.TokenService
}

func NewService(users repository.UserRepository, hasher service.HasherService, tokens service.TokenService) *Service {
	return &Service{
		users:  users,
		hasher: hasher,
		tokens: tokens,
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

	accessToken, err := s.tokens.GenerateAccessToken(u.ID, u.Role.String())
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

	accessToken, err := s.tokens.GenerateAccessToken(u.ID, u.Role.String())
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

	newAccessToken, err := s.tokens.GenerateAccessToken(u.ID, u.Role.String())
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
