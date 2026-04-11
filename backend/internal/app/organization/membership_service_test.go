package organization

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/domain/organization"
	"marketplace-backend/internal/domain/user"
)

// ---------------------------------------------------------------------------
// membership service — defensive delete path (R18)
// ---------------------------------------------------------------------------
//
// These tests cover the "swallow delete error" behaviour introduced by
// migration 076: when the downstream users.Delete call fails (e.g. a
// legacy FK constraint rejects the cascade), MembershipService must NOT
// return an error to the caller — doing so would leave the user in an
// orphan state (members row gone, user row stuck) and the HTTP client
// would see a 5xx for what is, from their perspective, a successful
// leave/remove. Instead, the service logs a greppable warning and
// returns nil. The orphan is later reclaimed by InvitationService on
// the next re-invite attempt. See invitation_service_test.go for the
// reclaim side.

// orphanDeleteUserRepo is a UserRepository mock that always fails the
// Delete call but behaves normally on every other method. Used to
// simulate the pre-migration FK-constraint failure.
type orphanDeleteUserRepo struct {
	mockUserRepoForInvites
	deleteErr    error
	deleteCalled int
}

func (m *orphanDeleteUserRepo) Delete(_ context.Context, _ uuid.UUID) error {
	m.deleteCalled++
	return m.deleteErr
}

func newOperatorUser(id uuid.UUID, email string) *user.User {
	return &user.User{
		ID:          id,
		Email:       email,
		AccountType: user.AccountTypeOperator,
	}
}

func TestMembershipService_LeaveOrganization_SwallowsDeleteError(t *testing.T) {
	ownerID := uuid.New()
	operatorID := uuid.New()
	org, _ := buildOrgAndOwnerMember(t, ownerID)

	membership, err := organization.NewMember(org.ID, operatorID, organization.RoleMember, "")
	require.NoError(t, err)

	orgs := &mockOrgRepo{
		findByIDFn: func(_ context.Context, _ uuid.UUID) (*organization.Organization, error) {
			return org, nil
		},
	}
	members := &mockMemberRepo{
		findByOrgAndUserFn: func(_ context.Context, _, userID uuid.UUID) (*organization.Member, error) {
			if userID == operatorID {
				return membership, nil
			}
			return nil, organization.ErrMemberNotFound
		},
		deleteFn: func(_ context.Context, _ uuid.UUID) error {
			return nil
		},
	}

	operator := newOperatorUser(operatorID, "op@example.test")
	users := &orphanDeleteUserRepo{
		mockUserRepoForInvites: mockUserRepoForInvites{
			getByIDFn: func(_ context.Context, id uuid.UUID) (*user.User, error) {
				if id == operatorID {
					return operator, nil
				}
				return nil, user.ErrUserNotFound
			},
		},
		deleteErr: errors.New("fk: messages_sender_id_fkey rejects delete"),
	}

	svc := NewMembershipService(MembershipServiceDeps{
		Orgs:    orgs,
		Members: members,
		Users:   users,
	})

	// Despite the users.Delete failing, LeaveOrganization must return nil.
	// The orphan is tolerated — it will be cleaned up on the next re-invite.
	err = svc.LeaveOrganization(context.Background(), operatorID, org.ID)
	assert.NoError(t, err)
	assert.Equal(t, 1, users.deleteCalled, "delete was attempted exactly once")
}

func TestMembershipService_LeaveOrganization_OwnerCannotLeave(t *testing.T) {
	ownerID := uuid.New()
	org, owner := buildOrgAndOwnerMember(t, ownerID)

	orgs := &mockOrgRepo{
		findByIDFn: func(_ context.Context, _ uuid.UUID) (*organization.Organization, error) {
			return org, nil
		},
	}
	members := &mockMemberRepo{
		findByOrgAndUserFn: func(_ context.Context, _, _ uuid.UUID) (*organization.Member, error) {
			return owner, nil
		},
	}
	svc := NewMembershipService(MembershipServiceDeps{
		Orgs:    orgs,
		Members: members,
		Users:   &mockUserRepoForInvites{},
	})

	err := svc.LeaveOrganization(context.Background(), ownerID, org.ID)
	assert.ErrorIs(t, err, organization.ErrLastOwnerCannotLeave)
}

func TestMembershipService_RemoveMember_SwallowsDeleteError(t *testing.T) {
	ownerID := uuid.New()
	operatorID := uuid.New()
	org, owner := buildOrgAndOwnerMember(t, ownerID)

	target, err := organization.NewMember(org.ID, operatorID, organization.RoleMember, "")
	require.NoError(t, err)

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
			if userID == operatorID {
				return target, nil
			}
			return nil, organization.ErrMemberNotFound
		},
		deleteFn: func(_ context.Context, _ uuid.UUID) error {
			return nil
		},
	}

	operator := newOperatorUser(operatorID, "op@example.test")
	users := &orphanDeleteUserRepo{
		mockUserRepoForInvites: mockUserRepoForInvites{
			getByIDFn: func(_ context.Context, id uuid.UUID) (*user.User, error) {
				if id == operatorID {
					return operator, nil
				}
				return &user.User{ID: id}, nil // actor lookup returns a stub
			},
		},
		deleteErr: errors.New("fk: messages_sender_id_fkey rejects delete"),
	}

	svc := NewMembershipService(MembershipServiceDeps{
		Orgs:    orgs,
		Members: members,
		Users:   users,
	})

	// The handler must see nil so the HTTP response is 204, not 500.
	err = svc.RemoveMember(context.Background(), ownerID, org.ID, operatorID)
	assert.NoError(t, err)
	assert.Equal(t, 1, users.deleteCalled, "delete was attempted exactly once")
}

func TestMembershipService_RemoveMember_CannotRemoveSelf(t *testing.T) {
	ownerID := uuid.New()
	org, _ := buildOrgAndOwnerMember(t, ownerID)

	svc := NewMembershipService(MembershipServiceDeps{
		Orgs:    &mockOrgRepo{},
		Members: &mockMemberRepo{},
		Users:   &mockUserRepoForInvites{},
	})

	err := svc.RemoveMember(context.Background(), ownerID, org.ID, ownerID)
	assert.ErrorIs(t, err, organization.ErrCannotRemoveSelf)
}
