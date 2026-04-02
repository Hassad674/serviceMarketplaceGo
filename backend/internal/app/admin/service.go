package admin

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/port/repository"
)

type Service struct {
	users repository.UserRepository
}

func NewService(users repository.UserRepository) *Service {
	return &Service{users: users}
}

func (s *Service) ListUsers(ctx context.Context, filters repository.AdminUserFilters) ([]*user.User, string, int, error) {
	users, nextCursor, err := s.users.ListAdmin(ctx, filters)
	if err != nil {
		return nil, "", 0, fmt.Errorf("list admin users: %w", err)
	}

	count, err := s.users.CountAdmin(ctx, filters)
	if err != nil {
		return nil, "", 0, fmt.Errorf("count admin users: %w", err)
	}

	return users, nextCursor, count, nil
}

func (s *Service) GetUser(ctx context.Context, id uuid.UUID) (*user.User, error) {
	u, err := s.users.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get admin user: %w", err)
	}
	return u, nil
}
