package organization

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/domain/audit"
	"marketplace-backend/internal/domain/organization"
)

// ---------------------------------------------------------------------------
// Test doubles
// ---------------------------------------------------------------------------

// stubRoleOverridesOrgRepo is a focused mock that captures persisted
// overrides and exposes a preset org for FindByID.
type stubRoleOverridesOrgRepo struct {
	mockOrgRepo
	orgByID     map[uuid.UUID]*organization.Organization
	savedCalled int
	savedWith   organization.RoleOverrides
}

func (s *stubRoleOverridesOrgRepo) FindByID(_ context.Context, id uuid.UUID) (*organization.Organization, error) {
	if org, ok := s.orgByID[id]; ok {
		// Return a shallow clone so the service can mutate its local copy
		// without racing with future reads.
		cp := *org
		cp.RoleOverrides = org.RoleOverrides.Clone()
		return &cp, nil
	}
	return nil, organization.ErrOrgNotFound
}

func (s *stubRoleOverridesOrgRepo) SaveRoleOverrides(_ context.Context, id uuid.UUID, overrides organization.RoleOverrides) error {
	s.savedCalled++
	s.savedWith = overrides.Clone()
	if org, ok := s.orgByID[id]; ok {
		org.RoleOverrides = overrides.Clone()
	}
	return nil
}

type stubRoleOverridesMemberRepo struct {
	mockMemberRepo
	memberByPair map[string]*organization.Member
	usersByRole  map[organization.Role][]uuid.UUID
}

func pairKey(orgID, userID uuid.UUID) string {
	return orgID.String() + ":" + userID.String()
}

func (s *stubRoleOverridesMemberRepo) FindByOrgAndUser(_ context.Context, orgID, userID uuid.UUID) (*organization.Member, error) {
	if m, ok := s.memberByPair[pairKey(orgID, userID)]; ok {
		return m, nil
	}
	return nil, organization.ErrMemberNotFound
}

func (s *stubRoleOverridesMemberRepo) ListUserIDsByRole(_ context.Context, _ uuid.UUID, role organization.Role) ([]uuid.UUID, error) {
	return s.usersByRole[role], nil
}

// stubRoleOverridesUserRepo wraps the already-complete
// mockUserRepoForInvites (defined in invitation_service_test.go) and
// overrides only the hooks this test file needs: BumpSessionVersion
// captures every bump so the test can assert on it.
type stubRoleOverridesUserRepo struct {
	mockUserRepoForInvites
	bumpedIDs []uuid.UUID
}

func (s *stubRoleOverridesUserRepo) BumpSessionVersion(_ context.Context, userID uuid.UUID) (int, error) {
	s.bumpedIDs = append(s.bumpedIDs, userID)
	return 2, nil
}

type stubAuditRepo struct {
	entries []*audit.Entry
}

func (s *stubAuditRepo) Log(_ context.Context, entry *audit.Entry) error {
	s.entries = append(s.entries, entry)
	return nil
}
func (s *stubAuditRepo) ListByResource(context.Context, audit.ResourceType, uuid.UUID, string, int) ([]*audit.Entry, string, error) {
	return nil, "", nil
}
func (s *stubAuditRepo) ListByUser(context.Context, uuid.UUID, string, int) ([]*audit.Entry, string, error) {
	return nil, "", nil
}

type stubRateLimiter struct {
	allow bool
	err   error
}

func (s *stubRateLimiter) Allow(context.Context, uuid.UUID) (bool, error) {
	return s.allow, s.err
}

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

func newTestOrg(ownerID uuid.UUID) *organization.Organization {
	org, _ := organization.NewOrganization(ownerID, organization.OrgTypeAgency, "Acme Corp")
	return org
}

func newTestMember(orgID, userID uuid.UUID, role organization.Role) *organization.Member {
	m, _ := organization.NewMember(orgID, userID, role, "Test title")
	return m
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

// TestUpdateRoleOverrides_OwnerGrantsMemberJobsDelete verifies the happy
// path: an Owner grants a new permission to Members and the service
// returns the expected diff, persists the override, bumps sessions,
// and writes audit entries.
func TestUpdateRoleOverrides_OwnerGrantsMemberJobsDelete(t *testing.T) {
	ownerID := uuid.New()
	orgID := uuid.New()
	org := newTestOrg(ownerID)
	org.ID = orgID

	member1 := uuid.New()
	member2 := uuid.New()

	orgs := &stubRoleOverridesOrgRepo{
		orgByID: map[uuid.UUID]*organization.Organization{orgID: org},
	}
	members := &stubRoleOverridesMemberRepo{
		memberByPair: map[string]*organization.Member{
			pairKey(orgID, ownerID): newTestMember(orgID, ownerID, organization.RoleOwner),
		},
		usersByRole: map[organization.Role][]uuid.UUID{
			organization.RoleMember: {member1, member2},
		},
	}
	users := &stubRoleOverridesUserRepo{}
	audits := &stubAuditRepo{}

	svc := NewRoleOverridesService(RoleOverridesServiceDeps{
		Orgs:        orgs,
		Members:     members,
		Users:       users,
		Audits:      audits,
		Email:       nil, // nil is allowed — notifyOwner is a no-op without email
		RateLimiter: &stubRateLimiter{allow: true},
	})

	result, err := svc.UpdateRoleOverrides(context.Background(), UpdateRoleOverridesInput{
		ActorUserID:    ownerID,
		OrganizationID: orgID,
		Role:           organization.RoleMember,
		Overrides: map[organization.Permission]bool{
			organization.PermJobsDelete: true, // grant — not in defaults
		},
	})

	require.NoError(t, err)
	assert.Equal(t, organization.RoleMember, result.Role)
	assert.Equal(t, []organization.Permission{organization.PermJobsDelete}, result.GrantedKeys)
	assert.Empty(t, result.RevokedKeys)
	assert.Equal(t, 2, result.AffectedMembers)

	// Persistence
	assert.Equal(t, 1, orgs.savedCalled)
	assert.True(t, orgs.savedWith[organization.RoleMember][organization.PermJobsDelete])

	// Session bumps
	assert.ElementsMatch(t, []uuid.UUID{member1, member2}, users.bumpedIDs)

	// Audit
	require.Len(t, audits.entries, 1)
	assert.Equal(t, audit.ActionRolePermissionsChanged, audits.entries[0].Action)
	assert.Equal(t, audit.ResourceTypeOrganization, audits.entries[0].ResourceType)
}

// TestUpdateRoleOverrides_AdminIsRejected verifies the Owner-only
// enforcement at the service layer.
func TestUpdateRoleOverrides_AdminIsRejected(t *testing.T) {
	ownerID := uuid.New()
	adminID := uuid.New()
	orgID := uuid.New()
	org := newTestOrg(ownerID)
	org.ID = orgID

	orgs := &stubRoleOverridesOrgRepo{
		orgByID: map[uuid.UUID]*organization.Organization{orgID: org},
	}
	members := &stubRoleOverridesMemberRepo{
		memberByPair: map[string]*organization.Member{
			pairKey(orgID, adminID): newTestMember(orgID, adminID, organization.RoleAdmin),
		},
	}

	svc := NewRoleOverridesService(RoleOverridesServiceDeps{
		Orgs:        orgs,
		Members:     members,
		Users:       &stubRoleOverridesUserRepo{},
		Audits:      &stubAuditRepo{},
		RateLimiter: &stubRateLimiter{allow: true},
	})

	_, err := svc.UpdateRoleOverrides(context.Background(), UpdateRoleOverridesInput{
		ActorUserID:    adminID,
		OrganizationID: orgID,
		Role:           organization.RoleMember,
		Overrides:      map[organization.Permission]bool{organization.PermJobsDelete: true},
	})

	assert.ErrorIs(t, err, organization.ErrPermissionDenied)
	assert.Equal(t, 0, orgs.savedCalled)
}

// TestUpdateRoleOverrides_OwnerRoleRejected verifies that the Owner
// row itself cannot be customized.
func TestUpdateRoleOverrides_OwnerRoleRejected(t *testing.T) {
	ownerID := uuid.New()
	orgID := uuid.New()
	org := newTestOrg(ownerID)
	org.ID = orgID

	orgs := &stubRoleOverridesOrgRepo{
		orgByID: map[uuid.UUID]*organization.Organization{orgID: org},
	}
	members := &stubRoleOverridesMemberRepo{
		memberByPair: map[string]*organization.Member{
			pairKey(orgID, ownerID): newTestMember(orgID, ownerID, organization.RoleOwner),
		},
	}

	svc := NewRoleOverridesService(RoleOverridesServiceDeps{
		Orgs:        orgs,
		Members:     members,
		Users:       &stubRoleOverridesUserRepo{},
		Audits:      &stubAuditRepo{},
		RateLimiter: &stubRateLimiter{allow: true},
	})

	_, err := svc.UpdateRoleOverrides(context.Background(), UpdateRoleOverridesInput{
		ActorUserID:    ownerID,
		OrganizationID: orgID,
		Role:           organization.RoleOwner,
		Overrides:      map[organization.Permission]bool{organization.PermJobsView: false},
	})

	assert.ErrorIs(t, err, organization.ErrCannotOverrideOwner)
}

// TestUpdateRoleOverrides_LockedPermissionRejected verifies that an
// attempt to grant a non-overridable permission is refused.
func TestUpdateRoleOverrides_LockedPermissionRejected(t *testing.T) {
	ownerID := uuid.New()
	orgID := uuid.New()
	org := newTestOrg(ownerID)
	org.ID = orgID

	orgs := &stubRoleOverridesOrgRepo{
		orgByID: map[uuid.UUID]*organization.Organization{orgID: org},
	}
	members := &stubRoleOverridesMemberRepo{
		memberByPair: map[string]*organization.Member{
			pairKey(orgID, ownerID): newTestMember(orgID, ownerID, organization.RoleOwner),
		},
	}

	svc := NewRoleOverridesService(RoleOverridesServiceDeps{
		Orgs:        orgs,
		Members:     members,
		Users:       &stubRoleOverridesUserRepo{},
		Audits:      &stubAuditRepo{},
		RateLimiter: &stubRateLimiter{allow: true},
	})

	_, err := svc.UpdateRoleOverrides(context.Background(), UpdateRoleOverridesInput{
		ActorUserID:    ownerID,
		OrganizationID: orgID,
		Role:           organization.RoleAdmin,
		Overrides: map[organization.Permission]bool{
			organization.PermWalletWithdraw: true, // locked
		},
	})

	assert.ErrorIs(t, err, organization.ErrPermissionNotOverridable)
	assert.Equal(t, 0, orgs.savedCalled)
}

// TestUpdateRoleOverrides_RateLimited verifies that hitting the cap
// returns ErrRolePermChangesRateLimit and makes no DB writes.
func TestUpdateRoleOverrides_RateLimited(t *testing.T) {
	ownerID := uuid.New()
	orgID := uuid.New()
	org := newTestOrg(ownerID)
	org.ID = orgID

	orgs := &stubRoleOverridesOrgRepo{
		orgByID: map[uuid.UUID]*organization.Organization{orgID: org},
	}
	members := &stubRoleOverridesMemberRepo{
		memberByPair: map[string]*organization.Member{
			pairKey(orgID, ownerID): newTestMember(orgID, ownerID, organization.RoleOwner),
		},
	}

	svc := NewRoleOverridesService(RoleOverridesServiceDeps{
		Orgs:        orgs,
		Members:     members,
		Users:       &stubRoleOverridesUserRepo{},
		Audits:      &stubAuditRepo{},
		RateLimiter: &stubRateLimiter{allow: false},
	})

	_, err := svc.UpdateRoleOverrides(context.Background(), UpdateRoleOverridesInput{
		ActorUserID:    ownerID,
		OrganizationID: orgID,
		Role:           organization.RoleMember,
		Overrides:      map[organization.Permission]bool{organization.PermJobsDelete: true},
	})

	assert.ErrorIs(t, err, organization.ErrRolePermChangesRateLimit)
	assert.Equal(t, 0, orgs.savedCalled)
}

// TestUpdateRoleOverrides_NormalizeCollapsesRedundantCells verifies
// that sending a redundant override (same as default) does not
// persist an entry in the JSONB blob.
func TestUpdateRoleOverrides_NormalizeCollapsesRedundantCells(t *testing.T) {
	ownerID := uuid.New()
	orgID := uuid.New()
	org := newTestOrg(ownerID)
	org.ID = orgID

	orgs := &stubRoleOverridesOrgRepo{
		orgByID: map[uuid.UUID]*organization.Organization{orgID: org},
	}
	members := &stubRoleOverridesMemberRepo{
		memberByPair: map[string]*organization.Member{
			pairKey(orgID, ownerID): newTestMember(orgID, ownerID, organization.RoleOwner),
		},
	}

	svc := NewRoleOverridesService(RoleOverridesServiceDeps{
		Orgs:        orgs,
		Members:     members,
		Users:       &stubRoleOverridesUserRepo{},
		Audits:      &stubAuditRepo{},
		RateLimiter: &stubRateLimiter{allow: true},
	})

	_, err := svc.UpdateRoleOverrides(context.Background(), UpdateRoleOverridesInput{
		ActorUserID:    ownerID,
		OrganizationID: orgID,
		Role:           organization.RoleMember,
		Overrides: map[organization.Permission]bool{
			// Member already has PermJobsCreate by default —
			// this cell is redundant and must be discarded.
			organization.PermJobsCreate: true,
		},
	})

	require.NoError(t, err)
	// Persisted overrides should be empty / have no Member entry.
	_, has := orgs.savedWith[organization.RoleMember]
	assert.False(t, has, "redundant cells must not persist as overrides")
}

// TestGetMatrix_NonMemberRejected verifies that GetMatrix refuses to
// serve the matrix to users who are not part of the org.
func TestGetMatrix_NonMemberRejected(t *testing.T) {
	outsiderID := uuid.New()
	orgID := uuid.New()
	org := newTestOrg(uuid.New())
	org.ID = orgID

	orgs := &stubRoleOverridesOrgRepo{
		orgByID: map[uuid.UUID]*organization.Organization{orgID: org},
	}
	members := &stubRoleOverridesMemberRepo{
		memberByPair: map[string]*organization.Member{}, // no membership
	}

	svc := NewRoleOverridesService(RoleOverridesServiceDeps{
		Orgs:        orgs,
		Members:     members,
		Users:       &stubRoleOverridesUserRepo{},
		Audits:      &stubAuditRepo{},
		RateLimiter: &stubRateLimiter{allow: true},
	})

	_, err := svc.GetMatrix(context.Background(), outsiderID, orgID)
	assert.True(t, errors.Is(err, organization.ErrNotAMember))
}

// TestGetMatrix_ReturnsAllRoles verifies that GetMatrix always
// returns the four canonical roles in order.
func TestGetMatrix_ReturnsAllRoles(t *testing.T) {
	ownerID := uuid.New()
	orgID := uuid.New()
	org := newTestOrg(ownerID)
	org.ID = orgID

	orgs := &stubRoleOverridesOrgRepo{
		orgByID: map[uuid.UUID]*organization.Organization{orgID: org},
	}
	members := &stubRoleOverridesMemberRepo{
		memberByPair: map[string]*organization.Member{
			pairKey(orgID, ownerID): newTestMember(orgID, ownerID, organization.RoleOwner),
		},
	}

	svc := NewRoleOverridesService(RoleOverridesServiceDeps{
		Orgs:        orgs,
		Members:     members,
		Users:       &stubRoleOverridesUserRepo{},
		Audits:      &stubAuditRepo{},
		RateLimiter: &stubRateLimiter{allow: true},
	})

	matrix, err := svc.GetMatrix(context.Background(), ownerID, orgID)
	require.NoError(t, err)
	require.Len(t, matrix.Roles, 4)
	assert.Equal(t, organization.RoleOwner, matrix.Roles[0].Role)
	assert.Equal(t, organization.RoleAdmin, matrix.Roles[1].Role)
	assert.Equal(t, organization.RoleMember, matrix.Roles[2].Role)
	assert.Equal(t, organization.RoleViewer, matrix.Roles[3].Role)
}
