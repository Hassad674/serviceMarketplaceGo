package organization


import (
	"context"
	"time"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/organization"
	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/port/repository"
)

// mockOrgRepo is a minimal mock of repository.OrganizationRepository.
type mockOrgRepo struct {
	createFn                    func(ctx context.Context, org *organization.Organization) error
	createWithOwnerMembershipFn func(ctx context.Context, org *organization.Organization, member *organization.Member) error
	findByIDFn                  func(ctx context.Context, id uuid.UUID) (*organization.Organization, error)
	findByOwnerUserIDFn         func(ctx context.Context, ownerUserID uuid.UUID) (*organization.Organization, error)
	updateFn                    func(ctx context.Context, org *organization.Organization) error
	deleteFn                    func(ctx context.Context, id uuid.UUID) error
}

var _ repository.OrganizationRepository = (*mockOrgRepo)(nil)

func (m *mockOrgRepo) Create(ctx context.Context, org *organization.Organization) error {
	if m.createFn != nil {
		return m.createFn(ctx, org)
	}
	return nil
}

func (m *mockOrgRepo) CreateWithOwnerMembership(ctx context.Context, org *organization.Organization, member *organization.Member) error {
	if m.createWithOwnerMembershipFn != nil {
		return m.createWithOwnerMembershipFn(ctx, org, member)
	}
	return nil
}

func (m *mockOrgRepo) FindByID(ctx context.Context, id uuid.UUID) (*organization.Organization, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(ctx, id)
	}
	return nil, organization.ErrOrgNotFound
}

func (m *mockOrgRepo) FindByOwnerUserID(ctx context.Context, ownerUserID uuid.UUID) (*organization.Organization, error) {
	if m.findByOwnerUserIDFn != nil {
		return m.findByOwnerUserIDFn(ctx, ownerUserID)
	}
	return nil, organization.ErrOrgNotFound
}

func (m *mockOrgRepo) Update(ctx context.Context, org *organization.Organization) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, org)
	}
	return nil
}

func (m *mockOrgRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

func (m *mockOrgRepo) CountAll(_ context.Context) (int, error) {
	return 0, nil
}

// --- stripe + kyc stubs (phase R5 additions) ---

func (m *mockOrgRepo) FindByUserID(_ context.Context, _ uuid.UUID) (*organization.Organization, error) {
	return nil, organization.ErrOrgNotFound
}
func (m *mockOrgRepo) FindByStripeAccountID(_ context.Context, _ string) (*organization.Organization, error) {
	return nil, organization.ErrOrgNotFound
}
func (m *mockOrgRepo) ListKYCPending(_ context.Context) ([]*organization.Organization, error) {
	return nil, nil
}
func (m *mockOrgRepo) GetStripeAccount(_ context.Context, _ uuid.UUID) (string, string, error) {
	return "", "", nil
}
func (m *mockOrgRepo) GetStripeAccountByUserID(_ context.Context, _ uuid.UUID) (string, string, error) {
	return "", "", nil
}
func (m *mockOrgRepo) SetStripeAccount(_ context.Context, _ uuid.UUID, _, _ string) error {
	return nil
}
func (m *mockOrgRepo) ClearStripeAccount(_ context.Context, _ uuid.UUID) error { return nil }
func (m *mockOrgRepo) GetStripeLastState(_ context.Context, _ uuid.UUID) ([]byte, error) {
	return nil, nil
}
func (m *mockOrgRepo) SaveStripeLastState(_ context.Context, _ uuid.UUID, _ []byte) error {
	return nil
}
func (m *mockOrgRepo) SetKYCFirstEarning(_ context.Context, _ uuid.UUID, _ time.Time) error {
	return nil
}
func (m *mockOrgRepo) SaveKYCNotificationState(_ context.Context, _ uuid.UUID, _ map[string]time.Time) error {
	return nil
}
func (m *mockOrgRepo) SaveRoleOverrides(_ context.Context, _ uuid.UUID, _ organization.RoleOverrides) error {
	return nil
}
func (m *mockOrgRepo) ListWithStripeAccount(_ context.Context) ([]uuid.UUID, error) {
	return nil, nil
}

// mockMemberRepo is a minimal mock of repository.OrganizationMemberRepository.
type mockMemberRepo struct {
	createFn              func(ctx context.Context, member *organization.Member) error
	findByIDFn            func(ctx context.Context, id uuid.UUID) (*organization.Member, error)
	findByOrgAndUserFn    func(ctx context.Context, orgID, userID uuid.UUID) (*organization.Member, error)
	findOwnerFn           func(ctx context.Context, orgID uuid.UUID) (*organization.Member, error)
	findUserPrimaryOrgFn  func(ctx context.Context, userID uuid.UUID) (*organization.Member, error)
	listFn                func(ctx context.Context, params repository.ListMembersParams) ([]*organization.Member, string, error)
	countByRoleFn         func(ctx context.Context, orgID uuid.UUID) (map[organization.Role]int, error)
	updateFn              func(ctx context.Context, member *organization.Member) error
	deleteFn              func(ctx context.Context, id uuid.UUID) error
}

var _ repository.OrganizationMemberRepository = (*mockMemberRepo)(nil)

func (m *mockMemberRepo) Create(ctx context.Context, member *organization.Member) error {
	if m.createFn != nil {
		return m.createFn(ctx, member)
	}
	return nil
}

func (m *mockMemberRepo) FindByID(ctx context.Context, id uuid.UUID) (*organization.Member, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(ctx, id)
	}
	return nil, organization.ErrMemberNotFound
}

func (m *mockMemberRepo) FindByOrgAndUser(ctx context.Context, orgID, userID uuid.UUID) (*organization.Member, error) {
	if m.findByOrgAndUserFn != nil {
		return m.findByOrgAndUserFn(ctx, orgID, userID)
	}
	return nil, organization.ErrMemberNotFound
}

func (m *mockMemberRepo) FindOwner(ctx context.Context, orgID uuid.UUID) (*organization.Member, error) {
	if m.findOwnerFn != nil {
		return m.findOwnerFn(ctx, orgID)
	}
	return nil, organization.ErrMemberNotFound
}

func (m *mockMemberRepo) FindUserPrimaryOrg(ctx context.Context, userID uuid.UUID) (*organization.Member, error) {
	if m.findUserPrimaryOrgFn != nil {
		return m.findUserPrimaryOrgFn(ctx, userID)
	}
	return nil, organization.ErrMemberNotFound
}

func (m *mockMemberRepo) List(ctx context.Context, params repository.ListMembersParams) ([]*organization.Member, string, error) {
	if m.listFn != nil {
		return m.listFn(ctx, params)
	}
	return nil, "", nil
}

func (m *mockMemberRepo) CountByRole(ctx context.Context, orgID uuid.UUID) (map[organization.Role]int, error) {
	if m.countByRoleFn != nil {
		return m.countByRoleFn(ctx, orgID)
	}
	return nil, nil
}

func (m *mockMemberRepo) Update(ctx context.Context, member *organization.Member) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, member)
	}
	return nil
}

func (m *mockMemberRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

func (m *mockMemberRepo) ListMemberUserIDsByOrgIDs(_ context.Context, orgIDs []uuid.UUID) (map[uuid.UUID][]uuid.UUID, error) {
	return map[uuid.UUID][]uuid.UUID{}, nil
}

func (m *mockMemberRepo) ListUserIDsByRole(_ context.Context, _ uuid.UUID, _ organization.Role) ([]uuid.UUID, error) {
	return nil, nil
}

// mockInvitationRepo is a minimal stub; invitation logic lands in Phase 2.
type mockInvitationRepo struct{}

var _ repository.OrganizationInvitationRepository = (*mockInvitationRepo)(nil)

func (m *mockInvitationRepo) Create(_ context.Context, _ *organization.Invitation) error {
	return nil
}

func (m *mockInvitationRepo) FindByID(_ context.Context, _ uuid.UUID) (*organization.Invitation, error) {
	return nil, organization.ErrInvitationNotFound
}

func (m *mockInvitationRepo) FindByToken(_ context.Context, _ string) (*organization.Invitation, error) {
	return nil, organization.ErrInvitationNotFound
}

func (m *mockInvitationRepo) FindPendingByOrgAndEmail(_ context.Context, _ uuid.UUID, _ string) (*organization.Invitation, error) {
	return nil, organization.ErrInvitationNotFound
}

func (m *mockInvitationRepo) List(_ context.Context, _ repository.ListInvitationsParams) ([]*organization.Invitation, string, error) {
	return nil, "", nil
}

func (m *mockInvitationRepo) Update(_ context.Context, _ *organization.Invitation) error {
	return nil
}

func (m *mockInvitationRepo) Delete(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (m *mockInvitationRepo) ExpireStale(_ context.Context) (int, error) {
	return 0, nil
}

func (m *mockInvitationRepo) CountPending(_ context.Context) (int, error) {
	return 0, nil
}

func (m *mockInvitationRepo) AcceptInvitationTx(
	_ context.Context,
	_ *organization.Invitation,
	_ *user.User,
	_ *organization.Member,
) error {
	return nil
}
