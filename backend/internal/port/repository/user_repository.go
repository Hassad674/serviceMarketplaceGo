package repository

import (
	"context"

	"github.com/google/uuid"
	"marketplace-backend/internal/domain/user"
)

type AdminUserFilters struct {
	Role   string
	Status string
	Search string
	Cursor string
	Limit  int
}

type UserRepository interface {
	Create(ctx context.Context, u *user.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*user.User, error)
	GetByEmail(ctx context.Context, email string) (*user.User, error)
	Update(ctx context.Context, u *user.User) error
	Delete(ctx context.Context, id uuid.UUID) error
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	ListAdmin(ctx context.Context, filters AdminUserFilters) ([]*user.User, string, error)
	CountAdmin(ctx context.Context, filters AdminUserFilters) (int, error)
}
