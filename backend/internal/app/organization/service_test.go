package organization

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/domain/organization"
	"marketplace-backend/internal/domain/user"
)

func newAgencyUser() *user.User {
	u, _ := user.NewUser("acme@example.com", "hash", "Sarah", "Connor", "Acme Corp", user.RoleAgency)
	return u
}

func newEnterpriseUser() *user.User {
	u, _ := user.NewUser("buyer@example.com", "hash", "John", "Smith", "Buyer SA", user.RoleEnterprise)
	return u
}

func newProviderUser() *user.User {
	u, _ := user.NewUser("freelance@example.com", "hash", "Marie", "Durand", "Marie Durand", user.RoleProvider)
	return u
}

func newTestService(orgs *mockOrgRepo, members *mockMemberRepo) *Service {
	if orgs == nil {
		orgs = &mockOrgRepo{}
	}
	if members == nil {
		members = &mockMemberRepo{}
	}
	return NewService(orgs, members, &mockInvitationRepo{})
}

func TestService_CreateForOwner_Agency(t *testing.T) {
	var capturedOrg *organization.Organization
	var capturedMember *organization.Member

	orgs := &mockOrgRepo{
		createWithOwnerMembershipFn: func(_ context.Context, org *organization.Organization, member *organization.Member) error {
			capturedOrg = org
			capturedMember = member
			return nil
		},
	}

	svc := newTestService(orgs, nil)
	u := newAgencyUser()

	ctx, err := svc.CreateForOwner(context.Background(), u)
	require.NoError(t, err)
	require.NotNil(t, ctx)

	// Organization was persisted with correct fields
	require.NotNil(t, capturedOrg)
	assert.Equal(t, u.ID, capturedOrg.OwnerUserID)
	assert.Equal(t, organization.OrgTypeAgency, capturedOrg.Type)

	// Owner membership links to the same org
	require.NotNil(t, capturedMember)
	assert.Equal(t, capturedOrg.ID, capturedMember.OrganizationID)
	assert.Equal(t, u.ID, capturedMember.UserID)
	assert.Equal(t, organization.RoleOwner, capturedMember.Role)

	// Returned context carries everything the auth flow needs
	assert.Same(t, capturedOrg, ctx.Organization)
	assert.Same(t, capturedMember, ctx.Member)
	assert.NotEmpty(t, ctx.Permissions)
	// Owner should have at least withdraw permission
	found := false
	for _, p := range ctx.Permissions {
		if p == organization.PermWalletWithdraw {
			found = true
			break
		}
	}
	assert.True(t, found, "Owner should have wallet.withdraw permission")
}

func TestService_CreateForOwner_Enterprise(t *testing.T) {
	var capturedOrg *organization.Organization
	orgs := &mockOrgRepo{
		createWithOwnerMembershipFn: func(_ context.Context, org *organization.Organization, _ *organization.Member) error {
			capturedOrg = org
			return nil
		},
	}

	svc := newTestService(orgs, nil)
	u := newEnterpriseUser()

	ctx, err := svc.CreateForOwner(context.Background(), u)
	require.NoError(t, err)
	require.NotNil(t, ctx)
	assert.Equal(t, organization.OrgTypeEnterprise, capturedOrg.Type)
}

func TestService_CreateForOwner_ProviderPersonal(t *testing.T) {
	var capturedOrg *organization.Organization
	orgs := &mockOrgRepo{
		createWithOwnerMembershipFn: func(_ context.Context, org *organization.Organization, _ *organization.Member) error {
			capturedOrg = org
			return nil
		},
	}

	svc := newTestService(orgs, nil)
	u := newProviderUser()

	ctx, err := svc.CreateForOwner(context.Background(), u)
	require.NoError(t, err)
	require.NotNil(t, ctx)
	require.NotNil(t, capturedOrg)
	assert.Equal(t, organization.OrgTypeProviderPersonal, capturedOrg.Type)
	assert.NotEmpty(t, capturedOrg.Name)
}

func TestService_CreateForOwner_NilUser(t *testing.T) {
	svc := newTestService(nil, nil)
	ctx, err := svc.CreateForOwner(context.Background(), nil)
	assert.Error(t, err)
	assert.Nil(t, ctx)
}

func TestService_ResolveContext_UserIsMember(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	member, _ := organization.NewMember(orgID, userID, organization.RoleAdmin, "Lead Designer")
	org, _ := organization.NewOrganization(uuid.New(), organization.OrgTypeAgency, "Acme")
	org.ID = orgID

	orgs := &mockOrgRepo{
		findByIDFn: func(_ context.Context, id uuid.UUID) (*organization.Organization, error) {
			assert.Equal(t, orgID, id)
			return org, nil
		},
	}
	members := &mockMemberRepo{
		findUserPrimaryOrgFn: func(_ context.Context, id uuid.UUID) (*organization.Member, error) {
			assert.Equal(t, userID, id)
			return member, nil
		},
	}

	svc := newTestService(orgs, members)
	ctx, err := svc.ResolveContext(context.Background(), userID)
	require.NoError(t, err)
	require.NotNil(t, ctx)
	assert.Equal(t, org, ctx.Organization)
	assert.Equal(t, member, ctx.Member)
	assert.NotEmpty(t, ctx.Permissions)
}

func TestService_ResolveContext_SoloUserReturnsNil(t *testing.T) {
	members := &mockMemberRepo{
		findUserPrimaryOrgFn: func(_ context.Context, _ uuid.UUID) (*organization.Member, error) {
			return nil, organization.ErrMemberNotFound
		},
	}

	svc := newTestService(nil, members)
	ctx, err := svc.ResolveContext(context.Background(), uuid.New())
	require.NoError(t, err, "solo user is not an error")
	assert.Nil(t, ctx)
}

func TestService_HasPermission_Owner(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	member, _ := organization.NewMember(orgID, userID, organization.RoleOwner, "")
	org, _ := organization.NewOrganization(userID, organization.OrgTypeAgency, "Acme")
	org.ID = orgID

	svc := newTestService(
		&mockOrgRepo{
			findByIDFn: func(_ context.Context, _ uuid.UUID) (*organization.Organization, error) {
				return org, nil
			},
		},
		&mockMemberRepo{
			findUserPrimaryOrgFn: func(_ context.Context, _ uuid.UUID) (*organization.Member, error) {
				return member, nil
			},
		},
	)

	ok, err := svc.HasPermission(context.Background(), userID, organization.PermWalletWithdraw)
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestService_HasPermission_Viewer_ReadOnly(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	member, _ := organization.NewMember(orgID, userID, organization.RoleViewer, "")
	org, _ := organization.NewOrganization(uuid.New(), organization.OrgTypeAgency, "Acme")
	org.ID = orgID

	svc := newTestService(
		&mockOrgRepo{
			findByIDFn: func(_ context.Context, _ uuid.UUID) (*organization.Organization, error) {
				return org, nil
			},
		},
		&mockMemberRepo{
			findUserPrimaryOrgFn: func(_ context.Context, _ uuid.UUID) (*organization.Member, error) {
				return member, nil
			},
		},
	)

	// Viewer can see jobs
	ok, err := svc.HasPermission(context.Background(), userID, organization.PermJobsView)
	require.NoError(t, err)
	assert.True(t, ok)

	// Viewer cannot create jobs
	ok, err = svc.HasPermission(context.Background(), userID, organization.PermJobsCreate)
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestService_HasPermission_SoloUserAlwaysFalse(t *testing.T) {
	members := &mockMemberRepo{
		findUserPrimaryOrgFn: func(_ context.Context, _ uuid.UUID) (*organization.Member, error) {
			return nil, organization.ErrMemberNotFound
		},
	}

	svc := newTestService(nil, members)
	ok, err := svc.HasPermission(context.Background(), uuid.New(), organization.PermJobsView)
	require.NoError(t, err)
	assert.False(t, ok, "solo user must not hold any org permissions")
}
