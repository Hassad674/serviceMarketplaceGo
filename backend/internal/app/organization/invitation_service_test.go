package organization

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/domain/organization"
	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
)

// ---------------------------------------------------------------------------
// Local mocks (not shared with service_test.go's mocks)
// ---------------------------------------------------------------------------

type mockUserRepoForInvites struct {
	getByEmailFn    func(ctx context.Context, email string) (*user.User, error)
	getByIDFn       func(ctx context.Context, id uuid.UUID) (*user.User, error)
	existsByEmailFn func(ctx context.Context, email string) (bool, error)
	deleteFn        func(ctx context.Context, id uuid.UUID) error
	deleteCalls     []uuid.UUID
}

var _ repository.UserRepository = (*mockUserRepoForInvites)(nil)

func (m *mockUserRepoForInvites) Create(_ context.Context, _ *user.User) error { return nil }
func (m *mockUserRepoForInvites) GetByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, user.ErrUserNotFound
}
func (m *mockUserRepoForInvites) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	if m.getByEmailFn != nil {
		return m.getByEmailFn(ctx, email)
	}
	return nil, user.ErrUserNotFound
}
func (m *mockUserRepoForInvites) Update(_ context.Context, _ *user.User) error { return nil }
func (m *mockUserRepoForInvites) Delete(ctx context.Context, id uuid.UUID) error {
	m.deleteCalls = append(m.deleteCalls, id)
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}
func (m *mockUserRepoForInvites) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	if m.existsByEmailFn != nil {
		return m.existsByEmailFn(ctx, email)
	}
	return false, nil
}
func (m *mockUserRepoForInvites) ListAdmin(_ context.Context, _ repository.AdminUserFilters) ([]*user.User, string, error) {
	return nil, "", nil
}
func (m *mockUserRepoForInvites) CountAdmin(_ context.Context, _ repository.AdminUserFilters) (int, error) {
	return 0, nil
}
func (m *mockUserRepoForInvites) CountByRole(_ context.Context) (map[string]int, error) {
	return nil, nil
}
func (m *mockUserRepoForInvites) CountByStatus(_ context.Context) (map[string]int, error) {
	return nil, nil
}
func (m *mockUserRepoForInvites) RecentSignups(_ context.Context, _ int) ([]*user.User, error) {
	return nil, nil
}
func (m *mockUserRepoForInvites) GetStripeAccount(_ context.Context, _ uuid.UUID) (string, string, error) {
	return "", "", nil
}
func (m *mockUserRepoForInvites) FindUserIDByStripeAccount(_ context.Context, _ string) (uuid.UUID, error) {
	return uuid.Nil, nil
}
func (m *mockUserRepoForInvites) SetStripeAccount(_ context.Context, _ uuid.UUID, _, _ string) error {
	return nil
}
func (m *mockUserRepoForInvites) ClearStripeAccount(_ context.Context, _ uuid.UUID) error {
	return nil
}
func (m *mockUserRepoForInvites) GetStripeLastState(_ context.Context, _ uuid.UUID) ([]byte, error) {
	return nil, nil
}
func (m *mockUserRepoForInvites) SaveStripeLastState(_ context.Context, _ uuid.UUID, _ []byte) error {
	return nil
}
func (m *mockUserRepoForInvites) SetKYCFirstEarning(_ context.Context, _ uuid.UUID, _ time.Time) error {
	return nil
}
func (m *mockUserRepoForInvites) GetKYCPendingUsers(_ context.Context) ([]*user.User, error) {
	return nil, nil
}
func (m *mockUserRepoForInvites) SaveKYCNotificationState(_ context.Context, _ uuid.UUID, _ map[string]time.Time) error {
	return nil
}

type mockHasher struct{}

func (m *mockHasher) Hash(password string) (string, error)        { return "hashed_" + password, nil }
func (m *mockHasher) Compare(hashed, password string) error       { return nil }

type mockEmailForInvites struct {
	sendTeamInvitationFn func(ctx context.Context, input service.TeamInvitationEmailInput) error
	lastInput            *service.TeamInvitationEmailInput
}

var _ service.EmailService = (*mockEmailForInvites)(nil)

func (m *mockEmailForInvites) SendPasswordReset(_ context.Context, _, _ string) error { return nil }
func (m *mockEmailForInvites) SendNotification(_ context.Context, _, _, _ string) error {
	return nil
}
func (m *mockEmailForInvites) SendTeamInvitation(ctx context.Context, input service.TeamInvitationEmailInput) error {
	m.lastInput = &input
	if m.sendTeamInvitationFn != nil {
		return m.sendTeamInvitationFn(ctx, input)
	}
	return nil
}

type mockInvitationRateLimiter struct {
	allowed bool
	err     error
}

func (m *mockInvitationRateLimiter) Allow(_ context.Context, _ uuid.UUID) (bool, error) {
	return m.allowed, m.err
}

type trackingInvitationRepo struct {
	mockInvitationRepo
	createdInvitations []*organization.Invitation
	storedInvitations  map[uuid.UUID]*organization.Invitation
	findPendingReturn  *organization.Invitation
	acceptTxCalled     bool
	acceptTxInvitation *organization.Invitation
	acceptTxUser       *user.User
	acceptTxMember     *organization.Member
}

func newTrackingInvitationRepo() *trackingInvitationRepo {
	return &trackingInvitationRepo{
		storedInvitations: make(map[uuid.UUID]*organization.Invitation),
	}
}

func (t *trackingInvitationRepo) Create(_ context.Context, inv *organization.Invitation) error {
	t.createdInvitations = append(t.createdInvitations, inv)
	t.storedInvitations[inv.ID] = inv
	return nil
}
func (t *trackingInvitationRepo) FindByID(_ context.Context, id uuid.UUID) (*organization.Invitation, error) {
	if inv, ok := t.storedInvitations[id]; ok {
		return inv, nil
	}
	return nil, organization.ErrInvitationNotFound
}
func (t *trackingInvitationRepo) FindByToken(_ context.Context, token string) (*organization.Invitation, error) {
	for _, inv := range t.storedInvitations {
		if inv.Token == token {
			return inv, nil
		}
	}
	return nil, organization.ErrInvitationNotFound
}
func (t *trackingInvitationRepo) FindPendingByOrgAndEmail(_ context.Context, _ uuid.UUID, _ string) (*organization.Invitation, error) {
	if t.findPendingReturn != nil {
		return t.findPendingReturn, nil
	}
	return nil, organization.ErrInvitationNotFound
}
func (t *trackingInvitationRepo) Update(_ context.Context, inv *organization.Invitation) error {
	t.storedInvitations[inv.ID] = inv
	return nil
}
func (t *trackingInvitationRepo) AcceptInvitationTx(_ context.Context, inv *organization.Invitation, u *user.User, m *organization.Member) error {
	t.acceptTxCalled = true
	t.acceptTxInvitation = inv
	t.acceptTxUser = u
	t.acceptTxMember = m
	t.storedInvitations[inv.ID] = inv
	return nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newTestInvitationService(
	orgs *mockOrgRepo,
	members *mockMemberRepo,
	invitations *trackingInvitationRepo,
	users *mockUserRepoForInvites,
	email *mockEmailForInvites,
	rateLimiter InvitationRateLimiter,
) *InvitationService {
	if orgs == nil {
		orgs = &mockOrgRepo{}
	}
	if members == nil {
		members = &mockMemberRepo{}
	}
	if invitations == nil {
		invitations = newTrackingInvitationRepo()
	}
	if users == nil {
		users = &mockUserRepoForInvites{}
	}
	if email == nil {
		email = &mockEmailForInvites{}
	}
	if rateLimiter == nil {
		rateLimiter = &mockInvitationRateLimiter{allowed: true}
	}
	return NewInvitationService(InvitationServiceDeps{
		Orgs:        orgs,
		Members:     members,
		Invitations: invitations,
		Users:       users,
		Hasher:      &mockHasher{},
		Email:       email,
		RateLimiter: rateLimiter,
		FrontendURL: "https://app.example.test",
	})
}

func buildOrgAndOwnerMember(t *testing.T, ownerID uuid.UUID) (*organization.Organization, *organization.Member) {
	t.Helper()
	org, err := organization.NewOrganization(ownerID, organization.OrgTypeAgency, "Acme")
	require.NoError(t, err)
	owner, err := organization.NewMember(org.ID, ownerID, organization.RoleOwner, "")
	require.NoError(t, err)
	return org, owner
}

// ---------------------------------------------------------------------------
// SendInvitation tests
// ---------------------------------------------------------------------------

func TestInvitationService_SendInvitation_Success(t *testing.T) {
	ownerID := uuid.New()
	org, owner := buildOrgAndOwnerMember(t, ownerID)

	orgs := &mockOrgRepo{
		findByIDFn: func(_ context.Context, _ uuid.UUID) (*organization.Organization, error) {
			return org, nil
		},
	}
	members := &mockMemberRepo{
		findByOrgAndUserFn: func(_ context.Context, _, userID uuid.UUID) (*organization.Member, error) {
			if userID == ownerID {
				return owner, nil
			}
			return nil, organization.ErrMemberNotFound
		},
	}
	invRepo := newTrackingInvitationRepo()
	email := &mockEmailForInvites{}
	svc := newTestInvitationService(orgs, members, invRepo, nil, email, nil)

	inv, err := svc.SendInvitation(context.Background(), SendInvitationInput{
		InviterUserID:  ownerID,
		OrganizationID: org.ID,
		Email:          "Marie@Example.Com",
		FirstName:      "Marie",
		LastName:       "Dupont",
		Title:          "Office Manager",
		Role:           organization.RoleMember,
	})
	require.NoError(t, err)
	require.NotNil(t, inv)
	assert.Equal(t, "marie@example.com", inv.Email) // normalized
	assert.Equal(t, organization.RoleMember, inv.Role)
	assert.Equal(t, organization.InvitationStatusPending, inv.Status)
	assert.Len(t, invRepo.createdInvitations, 1)

	// Email delivered with the right shape
	require.NotNil(t, email.lastInput)
	assert.Equal(t, "marie@example.com", email.lastInput.To)
	assert.Equal(t, "agency", email.lastInput.OrgType)
	assert.Equal(t, "member", email.lastInput.Role)
	assert.Contains(t, email.lastInput.AcceptURL, "https://app.example.test/invitation/")
	assert.Contains(t, email.lastInput.AcceptURL, inv.Token)
}

func TestInvitationService_SendInvitation_RejectsNonPermittedActor(t *testing.T) {
	ownerID := uuid.New()
	memberID := uuid.New()
	org, _ := buildOrgAndOwnerMember(t, ownerID)

	// Actor is a Viewer → no team.invite permission
	viewer, _ := organization.NewMember(org.ID, memberID, organization.RoleViewer, "")

	orgs := &mockOrgRepo{
		findByIDFn: func(_ context.Context, _ uuid.UUID) (*organization.Organization, error) { return org, nil },
	}
	members := &mockMemberRepo{
		findByOrgAndUserFn: func(_ context.Context, _, id uuid.UUID) (*organization.Member, error) {
			if id == memberID {
				return viewer, nil
			}
			return nil, organization.ErrMemberNotFound
		},
	}
	svc := newTestInvitationService(orgs, members, nil, nil, nil, nil)

	_, err := svc.SendInvitation(context.Background(), SendInvitationInput{
		InviterUserID:  memberID,
		OrganizationID: org.ID,
		Email:          "marie@example.test",
		FirstName:      "Marie",
		LastName:       "D",
		Role:           organization.RoleMember,
	})
	assert.ErrorIs(t, err, organization.ErrPermissionDenied)
}

func TestInvitationService_SendInvitation_RejectsOwnerRole(t *testing.T) {
	ownerID := uuid.New()
	org, owner := buildOrgAndOwnerMember(t, ownerID)

	orgs := &mockOrgRepo{
		findByIDFn: func(_ context.Context, _ uuid.UUID) (*organization.Organization, error) { return org, nil },
	}
	members := &mockMemberRepo{
		findByOrgAndUserFn: func(_ context.Context, _, _ uuid.UUID) (*organization.Member, error) {
			return owner, nil
		},
	}
	svc := newTestInvitationService(orgs, members, nil, nil, nil, nil)

	_, err := svc.SendInvitation(context.Background(), SendInvitationInput{
		InviterUserID:  ownerID,
		OrganizationID: org.ID,
		Email:          "another@example.test",
		FirstName:      "X",
		LastName:       "Y",
		Role:           organization.RoleOwner, // forbidden
	})
	assert.ErrorIs(t, err, organization.ErrCannotInviteAsOwner)
}

func TestInvitationService_SendInvitation_RejectsExistingEmail(t *testing.T) {
	ownerID := uuid.New()
	org, owner := buildOrgAndOwnerMember(t, ownerID)

	existingUser := &user.User{ID: uuid.New(), Email: "used@example.test"}
	orgs := &mockOrgRepo{
		findByIDFn: func(_ context.Context, _ uuid.UUID) (*organization.Organization, error) { return org, nil },
	}
	members := &mockMemberRepo{
		findByOrgAndUserFn: func(_ context.Context, _, _ uuid.UUID) (*organization.Member, error) {
			return owner, nil
		},
	}
	users := &mockUserRepoForInvites{
		getByEmailFn: func(_ context.Context, _ string) (*user.User, error) {
			return existingUser, nil
		},
	}
	svc := newTestInvitationService(orgs, members, nil, users, nil, nil)

	_, err := svc.SendInvitation(context.Background(), SendInvitationInput{
		InviterUserID:  ownerID,
		OrganizationID: org.ID,
		Email:          "used@example.test",
		FirstName:      "X",
		LastName:       "Y",
		Role:           organization.RoleMember,
	})
	assert.ErrorIs(t, err, organization.ErrAlreadyMember)
}

// TestInvitationService_SendInvitation_ReclaimsOrphanOperator verifies
// that an orphan operator (account_type=operator, no org, zero active
// memberships) is auto-deleted before the invitation flow proceeds,
// allowing the email to be re-invited in one call instead of requiring
// a manual DB cleanup. This is the self-healing path for the R18 bug.
func TestInvitationService_SendInvitation_ReclaimsOrphanOperator(t *testing.T) {
	ownerID := uuid.New()
	org, owner := buildOrgAndOwnerMember(t, ownerID)

	orphanID := uuid.New()
	orphan := &user.User{
		ID:             orphanID,
		Email:          "orphan@example.test",
		AccountType:    user.AccountTypeOperator,
		OrganizationID: nil, // no org — zombie state
	}

	orgs := &mockOrgRepo{
		findByIDFn: func(_ context.Context, _ uuid.UUID) (*organization.Organization, error) { return org, nil },
	}
	members := &mockMemberRepo{
		findByOrgAndUserFn: func(_ context.Context, _, userID uuid.UUID) (*organization.Member, error) {
			if userID == ownerID {
				return owner, nil
			}
			return nil, organization.ErrMemberNotFound
		},
		countByUserFn: func(_ context.Context, userID uuid.UUID) (int, error) {
			if userID == orphanID {
				return 0, nil // orphan — zero memberships
			}
			return 0, nil
		},
	}
	users := &mockUserRepoForInvites{
		getByEmailFn: func(_ context.Context, _ string) (*user.User, error) {
			return orphan, nil
		},
	}
	invRepo := newTrackingInvitationRepo()
	svc := newTestInvitationService(orgs, members, invRepo, users, nil, nil)

	inv, err := svc.SendInvitation(context.Background(), SendInvitationInput{
		InviterUserID:  ownerID,
		OrganizationID: org.ID,
		Email:          "orphan@example.test",
		FirstName:      "Re",
		LastName:       "Invited",
		Role:           organization.RoleMember,
	})
	require.NoError(t, err)
	require.NotNil(t, inv)

	// Orphan delete happened before the new invitation was created.
	require.Len(t, users.deleteCalls, 1)
	assert.Equal(t, orphanID, users.deleteCalls[0])
	// The new invitation was persisted.
	assert.Len(t, invRepo.createdInvitations, 1)
}

// TestInvitationService_SendInvitation_KeepsBlockingNonOrphanOperator
// verifies that when an existing operator still has active memberships,
// we do NOT delete them — we fall through to ErrAlreadyMember. This
// protects users who are legitimately operators in another org.
func TestInvitationService_SendInvitation_KeepsBlockingNonOrphanOperator(t *testing.T) {
	ownerID := uuid.New()
	org, owner := buildOrgAndOwnerMember(t, ownerID)

	activeID := uuid.New()
	active := &user.User{
		ID:          activeID,
		Email:       "active@example.test",
		AccountType: user.AccountTypeOperator,
	}

	orgs := &mockOrgRepo{
		findByIDFn: func(_ context.Context, _ uuid.UUID) (*organization.Organization, error) { return org, nil },
	}
	members := &mockMemberRepo{
		findByOrgAndUserFn: func(_ context.Context, _, userID uuid.UUID) (*organization.Member, error) {
			if userID == ownerID {
				return owner, nil
			}
			return nil, organization.ErrMemberNotFound
		},
		countByUserFn: func(_ context.Context, _ uuid.UUID) (int, error) {
			return 1, nil // still a member somewhere
		},
	}
	users := &mockUserRepoForInvites{
		getByEmailFn: func(_ context.Context, _ string) (*user.User, error) {
			return active, nil
		},
	}
	svc := newTestInvitationService(orgs, members, nil, users, nil, nil)

	_, err := svc.SendInvitation(context.Background(), SendInvitationInput{
		InviterUserID:  ownerID,
		OrganizationID: org.ID,
		Email:          "active@example.test",
		FirstName:      "X",
		LastName:       "Y",
		Role:           organization.RoleMember,
	})
	assert.ErrorIs(t, err, organization.ErrAlreadyMember)
	// Must NOT delete an active operator.
	assert.Empty(t, users.deleteCalls)
}

// TestInvitationService_SendInvitation_OrphanDeleteFailureStillBlocks
// verifies that when the orphan cleanup delete fails, the frontend
// still sees ErrAlreadyMember (the consistent error message stays).
func TestInvitationService_SendInvitation_OrphanDeleteFailureStillBlocks(t *testing.T) {
	ownerID := uuid.New()
	org, owner := buildOrgAndOwnerMember(t, ownerID)

	orphanID := uuid.New()
	orphan := &user.User{
		ID:          orphanID,
		Email:       "orphan@example.test",
		AccountType: user.AccountTypeOperator,
	}

	orgs := &mockOrgRepo{
		findByIDFn: func(_ context.Context, _ uuid.UUID) (*organization.Organization, error) { return org, nil },
	}
	members := &mockMemberRepo{
		findByOrgAndUserFn: func(_ context.Context, _, userID uuid.UUID) (*organization.Member, error) {
			if userID == ownerID {
				return owner, nil
			}
			return nil, organization.ErrMemberNotFound
		},
		countByUserFn: func(_ context.Context, _ uuid.UUID) (int, error) { return 0, nil },
	}
	users := &mockUserRepoForInvites{
		getByEmailFn: func(_ context.Context, _ string) (*user.User, error) {
			return orphan, nil
		},
		deleteFn: func(_ context.Context, _ uuid.UUID) error {
			return errors.New("fk constraint: messages_sender_fkey")
		},
	}
	svc := newTestInvitationService(orgs, members, nil, users, nil, nil)

	_, err := svc.SendInvitation(context.Background(), SendInvitationInput{
		InviterUserID:  ownerID,
		OrganizationID: org.ID,
		Email:          "orphan@example.test",
		FirstName:      "X",
		LastName:       "Y",
		Role:           organization.RoleMember,
	})
	assert.ErrorIs(t, err, organization.ErrAlreadyMember)
	// We tried to delete once (and failed).
	assert.Len(t, users.deleteCalls, 1)
}

func TestInvitationService_SendInvitation_RateLimited(t *testing.T) {
	ownerID := uuid.New()
	org, owner := buildOrgAndOwnerMember(t, ownerID)

	orgs := &mockOrgRepo{
		findByIDFn: func(_ context.Context, _ uuid.UUID) (*organization.Organization, error) { return org, nil },
	}
	members := &mockMemberRepo{
		findByOrgAndUserFn: func(_ context.Context, _, _ uuid.UUID) (*organization.Member, error) {
			return owner, nil
		},
	}
	svc := newTestInvitationService(orgs, members, nil, nil, nil, &mockInvitationRateLimiter{allowed: false})

	_, err := svc.SendInvitation(context.Background(), SendInvitationInput{
		InviterUserID:  ownerID,
		OrganizationID: org.ID,
		Email:          "new@example.test",
		FirstName:      "X",
		LastName:       "Y",
		Role:           organization.RoleMember,
	})
	assert.ErrorIs(t, err, ErrInvitationRateLimited)
}

// ---------------------------------------------------------------------------
// AcceptInvitation tests
// ---------------------------------------------------------------------------

func TestInvitationService_AcceptInvitation_Success(t *testing.T) {
	ownerID := uuid.New()
	org, _ := buildOrgAndOwnerMember(t, ownerID)

	// Seed a pending invitation
	inv, err := organization.NewInvitation(organization.NewInvitationInput{
		OrganizationID:  org.ID,
		Email:           "marie@example.test",
		FirstName:       "Marie",
		LastName:        "Dupont",
		Title:           "Lead",
		Role:            organization.RoleMember,
		InvitedByUserID: ownerID,
	})
	require.NoError(t, err)

	invRepo := newTrackingInvitationRepo()
	invRepo.storedInvitations[inv.ID] = inv

	orgs := &mockOrgRepo{
		findByIDFn: func(_ context.Context, _ uuid.UUID) (*organization.Organization, error) { return org, nil },
	}
	svc := newTestInvitationService(orgs, nil, invRepo, nil, nil, nil)

	result, err := svc.AcceptInvitation(context.Background(), AcceptInvitationInput{
		Token:    inv.Token,
		Password: "StrongPass1!",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.User)

	assert.Equal(t, "marie@example.test", result.User.Email)
	assert.Equal(t, user.AccountTypeOperator, result.User.AccountType)
	assert.Equal(t, user.RoleAgency, result.User.Role)
	assert.Equal(t, "hashed_StrongPass1!", result.User.HashedPassword)

	require.NotNil(t, result.Member)
	assert.Equal(t, organization.RoleMember, result.Member.Role)
	assert.Equal(t, "Lead", result.Member.Title)
	assert.Equal(t, result.User.ID, result.Member.UserID)
	assert.Equal(t, org.ID, result.Member.OrganizationID)

	// Transactional write was invoked
	assert.True(t, invRepo.acceptTxCalled)
	assert.Equal(t, organization.InvitationStatusAccepted, invRepo.acceptTxInvitation.Status)
}

func TestInvitationService_AcceptInvitation_RejectsExpired(t *testing.T) {
	ownerID := uuid.New()
	org, _ := buildOrgAndOwnerMember(t, ownerID)

	inv, _ := organization.NewInvitation(organization.NewInvitationInput{
		OrganizationID:  org.ID,
		Email:           "marie@example.test",
		FirstName:       "Marie",
		LastName:        "Dupont",
		Role:            organization.RoleMember,
		InvitedByUserID: ownerID,
	})
	inv.ExpiresAt = time.Now().Add(-time.Hour) // force expired

	invRepo := newTrackingInvitationRepo()
	invRepo.storedInvitations[inv.ID] = inv
	svc := newTestInvitationService(nil, nil, invRepo, nil, nil, nil)

	_, err := svc.AcceptInvitation(context.Background(), AcceptInvitationInput{
		Token:    inv.Token,
		Password: "StrongPass1!",
	})
	assert.ErrorIs(t, err, organization.ErrInvitationExpired)
	assert.False(t, invRepo.acceptTxCalled)
}

func TestInvitationService_AcceptInvitation_RejectsWeakPassword(t *testing.T) {
	ownerID := uuid.New()
	org, _ := buildOrgAndOwnerMember(t, ownerID)

	inv, _ := organization.NewInvitation(organization.NewInvitationInput{
		OrganizationID:  org.ID,
		Email:           "marie@example.test",
		FirstName:       "Marie",
		LastName:        "Dupont",
		Role:            organization.RoleMember,
		InvitedByUserID: ownerID,
	})
	invRepo := newTrackingInvitationRepo()
	invRepo.storedInvitations[inv.ID] = inv
	orgs := &mockOrgRepo{
		findByIDFn: func(_ context.Context, _ uuid.UUID) (*organization.Organization, error) { return org, nil },
	}
	svc := newTestInvitationService(orgs, nil, invRepo, nil, nil, nil)

	_, err := svc.AcceptInvitation(context.Background(), AcceptInvitationInput{
		Token:    inv.Token,
		Password: "weak", // fails user.NewPassword validation
	})
	assert.Error(t, err)
	assert.True(t, errors.Is(err, user.ErrWeakPassword))
	assert.False(t, invRepo.acceptTxCalled)
}

// ---------------------------------------------------------------------------
// CancelInvitation tests
// ---------------------------------------------------------------------------

func TestInvitationService_CancelInvitation_Success(t *testing.T) {
	ownerID := uuid.New()
	org, owner := buildOrgAndOwnerMember(t, ownerID)

	inv, _ := organization.NewInvitation(organization.NewInvitationInput{
		OrganizationID:  org.ID,
		Email:           "marie@example.test",
		FirstName:       "Marie",
		LastName:        "Dupont",
		Role:            organization.RoleMember,
		InvitedByUserID: ownerID,
	})
	invRepo := newTrackingInvitationRepo()
	invRepo.storedInvitations[inv.ID] = inv

	orgs := &mockOrgRepo{
		findByIDFn: func(_ context.Context, _ uuid.UUID) (*organization.Organization, error) { return org, nil },
	}
	members := &mockMemberRepo{
		findByOrgAndUserFn: func(_ context.Context, _, _ uuid.UUID) (*organization.Member, error) {
			return owner, nil
		},
	}
	svc := newTestInvitationService(orgs, members, invRepo, nil, nil, nil)

	err := svc.CancelInvitation(context.Background(), ownerID, org.ID, inv.ID)
	require.NoError(t, err)
	assert.Equal(t, organization.InvitationStatusCancelled, invRepo.storedInvitations[inv.ID].Status)
}

// --- Session version stubs (migration 056, Phase 3) ---
func (m *mockUserRepoForInvites) BumpSessionVersion(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}
func (m *mockUserRepoForInvites) GetSessionVersion(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}
