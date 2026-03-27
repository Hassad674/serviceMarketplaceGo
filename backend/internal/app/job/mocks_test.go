package job

import (
	"context"

	"github.com/google/uuid"

	domain "marketplace-backend/internal/domain/job"
	"marketplace-backend/internal/domain/user"
)

// --- mockJobRepo ---

type mockJobRepo struct {
	createFn        func(ctx context.Context, j *domain.Job) error
	getByIDFn       func(ctx context.Context, id uuid.UUID) (*domain.Job, error)
	updateFn        func(ctx context.Context, j *domain.Job) error
	listByCreatorFn func(ctx context.Context, creatorID uuid.UUID, cursor string, limit int) ([]*domain.Job, string, error)
}

func (m *mockJobRepo) Create(ctx context.Context, j *domain.Job) error {
	if m.createFn != nil {
		return m.createFn(ctx, j)
	}
	return nil
}

func (m *mockJobRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Job, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, domain.ErrJobNotFound
}

func (m *mockJobRepo) Update(ctx context.Context, j *domain.Job) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, j)
	}
	return nil
}

func (m *mockJobRepo) ListByCreator(ctx context.Context, creatorID uuid.UUID, cursor string, limit int) ([]*domain.Job, string, error) {
	if m.listByCreatorFn != nil {
		return m.listByCreatorFn(ctx, creatorID, cursor, limit)
	}
	return []*domain.Job{}, "", nil
}

// --- mockUserRepo ---

type mockUserRepo struct {
	getByIDFn func(ctx context.Context, id uuid.UUID) (*user.User, error)
}

func (m *mockUserRepo) Create(_ context.Context, _ *user.User) error { return nil }
func (m *mockUserRepo) Update(_ context.Context, _ *user.User) error { return nil }
func (m *mockUserRepo) Delete(_ context.Context, _ uuid.UUID) error  { return nil }
func (m *mockUserRepo) ExistsByEmail(_ context.Context, _ string) (bool, error) {
	return false, nil
}
func (m *mockUserRepo) GetByEmail(_ context.Context, _ string) (*user.User, error) {
	return nil, user.ErrUserNotFound
}

func (m *mockUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return &user.User{ID: id, Role: user.RoleEnterprise, DisplayName: "Test"}, nil
}
