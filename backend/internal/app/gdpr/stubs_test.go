package gdpr

import (
	"context"

	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/port/repository"
)

// stubMissingMethods makes stubUserRepo satisfy the narrowed
// repository.UserReader interface (the GDPR service's actual
// dependency) by panicking on every reader method other than
// GetByID. Tests focus on GetByID; if a future change adds a new
// repo call, the panic surfaces immediately rather than silently
// passing on a no-op. The legacy 14-method panic stub was shrunk to
// the 7 reader methods after the GDPR service narrowed.
type stubMissingMethods struct{}

func (stubMissingMethods) GetByEmail(_ context.Context, _ string) (*user.User, error) {
	panic("stub: GetByEmail not used")
}
func (stubMissingMethods) ExistsByEmail(_ context.Context, _ string) (bool, error) {
	panic("stub: ExistsByEmail not used")
}
func (stubMissingMethods) ListAdmin(_ context.Context, _ repository.AdminUserFilters) ([]*user.User, string, error) {
	panic("stub: ListAdmin not used")
}
func (stubMissingMethods) CountAdmin(_ context.Context, _ repository.AdminUserFilters) (int, error) {
	panic("stub: CountAdmin not used")
}
func (stubMissingMethods) CountByRole(_ context.Context) (map[string]int, error) {
	panic("stub: CountByRole not used")
}
func (stubMissingMethods) CountByStatus(_ context.Context) (map[string]int, error) {
	panic("stub: CountByStatus not used")
}
func (stubMissingMethods) RecentSignups(_ context.Context, _ int) ([]*user.User, error) {
	panic("stub: RecentSignups not used")
}

// Type assertion: stubUserRepo MUST be a UserReader at compile time.
var _ repository.UserReader = (*stubUserRepo)(nil)

// stubGDPRRepo MUST be a GDPRRepository at compile time.
var _ repository.GDPRRepository = (*stubGDPRRepo)(nil)
