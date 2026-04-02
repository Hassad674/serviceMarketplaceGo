package admin

import (
	"context"
	"fmt"
	"time"

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

func (s *Service) SuspendUser(ctx context.Context, userID uuid.UUID, reason string, expiresAt *time.Time) error {
	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("suspend user: %w", err)
	}

	u.Suspend(reason, expiresAt)

	if err := s.users.Update(ctx, u); err != nil {
		return fmt.Errorf("suspend user: save: %w", err)
	}
	return nil
}

func (s *Service) UnsuspendUser(ctx context.Context, userID uuid.UUID) error {
	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("unsuspend user: %w", err)
	}

	u.Unsuspend()

	if err := s.users.Update(ctx, u); err != nil {
		return fmt.Errorf("unsuspend user: save: %w", err)
	}
	return nil
}

func (s *Service) BanUser(ctx context.Context, userID uuid.UUID, reason string) error {
	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("ban user: %w", err)
	}

	u.Ban(reason)

	if err := s.users.Update(ctx, u); err != nil {
		return fmt.Errorf("ban user: save: %w", err)
	}
	return nil
}

func (s *Service) UnbanUser(ctx context.Context, userID uuid.UUID) error {
	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("unban user: %w", err)
	}

	u.Unban()

	if err := s.users.Update(ctx, u); err != nil {
		return fmt.Errorf("unban user: save: %w", err)
	}
	return nil
}
