package gdpr

import (
	"context"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/port/repository"
)

// stubMissingMethods makes stubUserRepo satisfy the full
// repository.UserRepository interface by panicking on every method
// the GDPR service does not call. Tests focus on GetByID; if a future
// change adds a new repo call, the panic surfaces immediately rather
// than silently passing on a no-op.
type stubMissingMethods struct{}

func (stubMissingMethods) Create(_ context.Context, _ *user.User) error { panic("stub: Create not used") }
func (stubMissingMethods) GetByEmail(_ context.Context, _ string) (*user.User, error) {
	panic("stub: GetByEmail not used")
}
func (stubMissingMethods) Update(_ context.Context, _ *user.User) error {
	panic("stub: Update not used")
}
func (stubMissingMethods) Delete(_ context.Context, _ uuid.UUID) error {
	panic("stub: Delete not used")
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
func (stubMissingMethods) BumpSessionVersion(_ context.Context, _ uuid.UUID) (int, error) {
	panic("stub: BumpSessionVersion not used")
}
func (stubMissingMethods) GetSessionVersion(_ context.Context, _ uuid.UUID) (int, error) {
	panic("stub: GetSessionVersion not used")
}
func (stubMissingMethods) UpdateEmailNotificationsEnabled(_ context.Context, _ uuid.UUID, _ bool) error {
	panic("stub: UpdateEmailNotificationsEnabled not used")
}
func (stubMissingMethods) TouchLastActive(_ context.Context, _ uuid.UUID) error {
	panic("stub: TouchLastActive not used")
}

// Type assertion: stubUserRepo MUST be a UserRepository at compile time.
var _ repository.UserRepository = (*stubUserRepo)(nil)

// stubGDPRRepo MUST be a GDPRRepository at compile time.
var _ repository.GDPRRepository = (*stubGDPRRepo)(nil)
