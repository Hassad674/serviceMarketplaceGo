package organization

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	notificationdomain "marketplace-backend/internal/domain/notification"
	"marketplace-backend/internal/domain/organization"
	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
)

// ---------------------------------------------------------------------------
// Shared mocks for notification trigger tests
// ---------------------------------------------------------------------------

// recordingSender is a NotificationSender that records every Send call
// in-memory so tests can assert on the type, recipient, and payload.
type recordingSender struct {
	mu    sync.Mutex
	calls []service.NotificationInput
	err   error // optional — simulate sender failures
}

func (r *recordingSender) Send(_ context.Context, in service.NotificationInput) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, in)
	return r.err
}

func (r *recordingSender) last() service.NotificationInput {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.calls) == 0 {
		return service.NotificationInput{}
	}
	return r.calls[len(r.calls)-1]
}

func (r *recordingSender) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.calls)
}

// mockUserRepoForMembership is a richer user repo mock that covers the
// methods MembershipService actually calls (GetByID, BumpSessionVersion,
// Delete). Other methods return zero values.
type mockUserRepoForMembership struct {
	users              map[uuid.UUID]*user.User
	bumpSessionErr     error
	bumpSessionCalls   []uuid.UUID
	deletedUsers       []uuid.UUID
	sessionVersion     map[uuid.UUID]int
	deleteErr          error
}

func newMockUserRepoForMembership() *mockUserRepoForMembership {
	return &mockUserRepoForMembership{
		users:          make(map[uuid.UUID]*user.User),
		sessionVersion: make(map[uuid.UUID]int),
	}
}

var _ repository.UserRepository = (*mockUserRepoForMembership)(nil)

func (m *mockUserRepoForMembership) Create(_ context.Context, _ *user.User) error { return nil }
func (m *mockUserRepoForMembership) GetByID(_ context.Context, id uuid.UUID) (*user.User, error) {
	if u, ok := m.users[id]; ok {
		return u, nil
	}
	return nil, user.ErrUserNotFound
}
func (m *mockUserRepoForMembership) GetByEmail(_ context.Context, _ string) (*user.User, error) {
	return nil, user.ErrUserNotFound
}
func (m *mockUserRepoForMembership) Update(_ context.Context, _ *user.User) error { return nil }
func (m *mockUserRepoForMembership) Delete(_ context.Context, id uuid.UUID) error {
	m.deletedUsers = append(m.deletedUsers, id)
	delete(m.users, id)
	return m.deleteErr
}
func (m *mockUserRepoForMembership) ExistsByEmail(_ context.Context, _ string) (bool, error) {
	return false, nil
}
func (m *mockUserRepoForMembership) ListAdmin(_ context.Context, _ repository.AdminUserFilters) ([]*user.User, string, error) {
	return nil, "", nil
}
func (m *mockUserRepoForMembership) CountAdmin(_ context.Context, _ repository.AdminUserFilters) (int, error) {
	return 0, nil
}
func (m *mockUserRepoForMembership) CountByRole(_ context.Context) (map[string]int, error) {
	return nil, nil
}
func (m *mockUserRepoForMembership) CountByStatus(_ context.Context) (map[string]int, error) {
	return nil, nil
}
func (m *mockUserRepoForMembership) RecentSignups(_ context.Context, _ int) ([]*user.User, error) {
	return nil, nil
}
func (m *mockUserRepoForMembership) GetStripeAccount(_ context.Context, _ uuid.UUID) (string, string, error) {
	return "", "", nil
}
func (m *mockUserRepoForMembership) FindUserIDByStripeAccount(_ context.Context, _ string) (uuid.UUID, error) {
	return uuid.Nil, nil
}
func (m *mockUserRepoForMembership) SetStripeAccount(_ context.Context, _ uuid.UUID, _, _ string) error {
	return nil
}
func (m *mockUserRepoForMembership) ClearStripeAccount(_ context.Context, _ uuid.UUID) error {
	return nil
}
func (m *mockUserRepoForMembership) GetStripeLastState(_ context.Context, _ uuid.UUID) ([]byte, error) {
	return nil, nil
}
func (m *mockUserRepoForMembership) SaveStripeLastState(_ context.Context, _ uuid.UUID, _ []byte) error {
	return nil
}
func (m *mockUserRepoForMembership) SetKYCFirstEarning(_ context.Context, _ uuid.UUID, _ time.Time) error {
	return nil
}
func (m *mockUserRepoForMembership) GetKYCPendingUsers(_ context.Context) ([]*user.User, error) {
	return nil, nil
}
func (m *mockUserRepoForMembership) SaveKYCNotificationState(_ context.Context, _ uuid.UUID, _ map[string]time.Time) error {
	return nil
}
func (m *mockUserRepoForMembership) UpdateEmailNotificationsEnabled(_ context.Context, _ uuid.UUID, _ bool) error {
	return nil
}
func (m *mockUserRepoForMembership) TouchLastActive(_ context.Context, _ uuid.UUID) error {
	return nil
}
func (m *mockUserRepoForMembership) BumpSessionVersion(_ context.Context, userID uuid.UUID) (int, error) {
	m.bumpSessionCalls = append(m.bumpSessionCalls, userID)
	m.sessionVersion[userID]++
	return m.sessionVersion[userID], m.bumpSessionErr
}
func (m *mockUserRepoForMembership) GetSessionVersion(_ context.Context, userID uuid.UUID) (int, error) {
	return m.sessionVersion[userID], nil
}

// ---------------------------------------------------------------------------
// Direct helper tests
// ---------------------------------------------------------------------------

func TestNotifier_Dispatch_NilSenderIsNoop(t *testing.T) {
	// Must not panic, must not do anything.
	dispatch(context.Background(), nil, uuid.New(),
		notificationdomain.TypeOrgMemberRoleChanged, "title", "body",
		json.RawMessage(`{}`))
}

func TestNotifier_Dispatch_NilUserIsNoop(t *testing.T) {
	sender := &recordingSender{}
	dispatch(context.Background(), sender, uuid.Nil,
		notificationdomain.TypeOrgMemberRoleChanged, "title", "body",
		json.RawMessage(`{}`))
	assert.Equal(t, 0, sender.count(), "nil user must not trigger a Send")
}

func TestNotifier_Dispatch_SwallowsSenderError(t *testing.T) {
	sender := &recordingSender{err: errors.New("downstream boom")}
	// This must not panic or propagate — dispatch is best-effort.
	dispatch(context.Background(), sender, uuid.New(),
		notificationdomain.TypeOrgMemberRoleChanged, "title", "body",
		json.RawMessage(`{}`))
	assert.Equal(t, 1, sender.count(), "Send is still invoked even if it errors")
}

func TestNotifier_OrgLabel(t *testing.T) {
	agencyOrg := &organization.Organization{Type: organization.OrgTypeAgency}
	entOrg := &organization.Organization{Type: organization.OrgTypeEnterprise}

	assert.Equal(t, "your agency", orgLabel(agencyOrg))
	assert.Equal(t, "your enterprise", orgLabel(entOrg))
	assert.Equal(t, "your organization", orgLabel(nil))
}

func TestNotifier_ActorDisplayName(t *testing.T) {
	tests := []struct {
		name string
		u    *user.User
		want string
	}{
		{"nil user", nil, "Someone"},
		{"display name wins",
			&user.User{DisplayName: "Sarah Connor", FirstName: "Sarah", LastName: "Connor"},
			"Sarah Connor"},
		{"first + last fallback",
			&user.User{FirstName: "Bob", LastName: "Builder"},
			"Bob Builder"},
		{"first only",
			&user.User{FirstName: "Cher"},
			"Cher"},
		{"empty everywhere",
			&user.User{},
			"Someone"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, actorDisplayName(tt.u))
		})
	}
}

func TestNotifier_MarshalData(t *testing.T) {
	raw := marshalData(map[string]any{"k": "v", "n": 42})
	var parsed map[string]any
	require.NoError(t, json.Unmarshal(raw, &parsed))
	assert.Equal(t, "v", parsed["k"])
	assert.EqualValues(t, 42, parsed["n"])

	// Nil map → empty object, not nil.
	assert.JSONEq(t, `{}`, string(marshalData(nil)))
}

// ---------------------------------------------------------------------------
// MembershipService trigger tests
// ---------------------------------------------------------------------------

// buildMembershipHarness wires a MembershipService with in-memory mocks
// for a single agency org containing {owner, admin, member}. Returns
// the service, the sender, and the key user IDs so each test can
// call any action with minimal boilerplate.
type membershipHarness struct {
	svc       *MembershipService
	sender    *recordingSender
	org       *organization.Organization
	ownerID   uuid.UUID
	adminID   uuid.UUID
	memberID  uuid.UUID
	memberMap map[uuid.UUID]*organization.Member
}

func buildMembershipHarness(t *testing.T) *membershipHarness {
	t.Helper()

	ownerID := uuid.New()
	adminID := uuid.New()
	memberID := uuid.New()

	org, err := organization.NewOrganization(ownerID, organization.OrgTypeAgency, "Acme")
	require.NoError(t, err)

	ownerMember, err := organization.NewMember(org.ID, ownerID, organization.RoleOwner, "Founder")
	require.NoError(t, err)
	adminMember, err := organization.NewMember(org.ID, adminID, organization.RoleAdmin, "Head of Ops")
	require.NoError(t, err)
	memberMember, err := organization.NewMember(org.ID, memberID, organization.RoleMember, "Engineer")
	require.NoError(t, err)

	memberMap := map[uuid.UUID]*organization.Member{
		ownerID:  ownerMember,
		adminID:  adminMember,
		memberID: memberMember,
	}

	orgs := &mockOrgRepo{
		findByIDFn: func(_ context.Context, _ uuid.UUID) (*organization.Organization, error) {
			return org, nil
		},
		updateFn: func(_ context.Context, _ *organization.Organization) error { return nil },
	}
	members := &mockMemberRepo{
		findByOrgAndUserFn: func(_ context.Context, _, userID uuid.UUID) (*organization.Member, error) {
			if m, ok := memberMap[userID]; ok {
				return m, nil
			}
			return nil, organization.ErrMemberNotFound
		},
		updateFn: func(_ context.Context, _ *organization.Member) error { return nil },
		deleteFn: func(_ context.Context, _ uuid.UUID) error { return nil },
	}

	users := newMockUserRepoForMembership()
	users.users[ownerID] = &user.User{
		ID: ownerID, FirstName: "Sarah", LastName: "Connor",
		DisplayName: "Sarah Connor", AccountType: user.AccountTypeMarketplaceOwner,
		Role: user.RoleAgency,
	}
	users.users[adminID] = &user.User{
		ID: adminID, FirstName: "Alice", LastName: "Admin",
		DisplayName: "Alice Admin", AccountType: user.AccountTypeOperator,
		Role: user.RoleAgency,
	}
	users.users[memberID] = &user.User{
		ID: memberID, FirstName: "Bob", LastName: "Member",
		DisplayName: "Bob Member", AccountType: user.AccountTypeOperator,
		Role: user.RoleAgency,
	}

	sender := &recordingSender{}
	svc := NewMembershipService(MembershipServiceDeps{
		Orgs:          orgs,
		Members:       members,
		Users:         users,
		Notifications: sender,
	})
	return &membershipHarness{
		svc:       svc,
		sender:    sender,
		org:       org,
		ownerID:   ownerID,
		adminID:   adminID,
		memberID:  memberID,
		memberMap: memberMap,
	}
}

func TestMembershipService_UpdateMemberRole_EmitsRoleChangedNotification(t *testing.T) {
	h := buildMembershipHarness(t)

	_, err := h.svc.UpdateMemberRole(context.Background(), h.ownerID, h.org.ID, h.memberID, organization.RoleAdmin)
	require.NoError(t, err)

	require.Equal(t, 1, h.sender.count(), "must emit exactly one notification")
	call := h.sender.last()
	assert.Equal(t, string(notificationdomain.TypeOrgMemberRoleChanged), call.Type)
	assert.Equal(t, h.memberID, call.UserID)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(call.Data, &payload))
	assert.Equal(t, h.org.ID.String(), payload["organization_id"])
	assert.Equal(t, "member", payload["old_role"])
	assert.Equal(t, "admin", payload["new_role"])
	assert.Equal(t, "Sarah Connor", payload["actor_name"])
}

func TestMembershipService_UpdateMemberTitle_EmitsTitleOnlyNotification(t *testing.T) {
	h := buildMembershipHarness(t)

	_, err := h.svc.UpdateMemberTitle(context.Background(), h.ownerID, h.org.ID, h.memberID, "Staff Engineer")
	require.NoError(t, err)

	require.Equal(t, 1, h.sender.count())
	call := h.sender.last()
	assert.Equal(t, string(notificationdomain.TypeOrgMemberRoleChanged), call.Type)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(call.Data, &payload))
	assert.Equal(t, true, payload["title_only"])
	assert.Equal(t, "Staff Engineer", payload["new_title"])
}

func TestMembershipService_UpdateMemberTitle_SelfUpdateSkipsNotification(t *testing.T) {
	h := buildMembershipHarness(t)

	_, err := h.svc.UpdateMemberTitle(context.Background(), h.memberID, h.org.ID, h.memberID, "Senior Dev")
	require.NoError(t, err)

	assert.Equal(t, 0, h.sender.count(), "self title update must not fire a notification")
}

func TestMembershipService_RemoveMember_EmitsRemovedNotificationBeforeDelete(t *testing.T) {
	h := buildMembershipHarness(t)

	err := h.svc.RemoveMember(context.Background(), h.ownerID, h.org.ID, h.memberID)
	require.NoError(t, err)

	require.Equal(t, 1, h.sender.count())
	call := h.sender.last()
	assert.Equal(t, string(notificationdomain.TypeOrgMemberRemoved), call.Type)
	assert.Equal(t, h.memberID, call.UserID)
	// Operator account must also have been deleted in the same flow.
	users := h.svc.users.(*mockUserRepoForMembership)
	assert.Contains(t, users.deletedUsers, h.memberID, "operator account must be purged")
}

func TestMembershipService_LeaveOrganization_EmitsLeftNotificationToOwner(t *testing.T) {
	h := buildMembershipHarness(t)

	err := h.svc.LeaveOrganization(context.Background(), h.memberID, h.org.ID)
	require.NoError(t, err)

	require.Equal(t, 1, h.sender.count())
	call := h.sender.last()
	assert.Equal(t, string(notificationdomain.TypeOrgMemberLeft), call.Type)
	assert.Equal(t, h.ownerID, call.UserID, "owner must be the recipient")

	var payload map[string]any
	require.NoError(t, json.Unmarshal(call.Data, &payload))
	assert.Equal(t, "Bob Member", payload["leaver_name"])
}

// ---------------------------------------------------------------------------
// TransferService trigger tests (methods hang off MembershipService)
// ---------------------------------------------------------------------------

func TestTransferService_InitiateTransferOwnership_EmitsInitiatedNotification(t *testing.T) {
	h := buildMembershipHarness(t)

	_, err := h.svc.InitiateTransferOwnership(context.Background(), h.ownerID, h.org.ID, h.adminID)
	require.NoError(t, err)

	require.Equal(t, 1, h.sender.count())
	call := h.sender.last()
	assert.Equal(t, string(notificationdomain.TypeOrgTransferInitiated), call.Type)
	assert.Equal(t, h.adminID, call.UserID, "pending new owner receives the notif")
}

func TestTransferService_CancelTransferOwnership_EmitsCancelledNotification(t *testing.T) {
	h := buildMembershipHarness(t)

	// Put the org in a "pending transfer" state first.
	_, err := h.svc.InitiateTransferOwnership(context.Background(), h.ownerID, h.org.ID, h.adminID)
	require.NoError(t, err)
	require.Equal(t, 1, h.sender.count())

	err = h.svc.CancelTransferOwnership(context.Background(), h.ownerID, h.org.ID)
	require.NoError(t, err)

	require.Equal(t, 2, h.sender.count(), "cancel must emit a second notification")
	call := h.sender.last()
	assert.Equal(t, string(notificationdomain.TypeOrgTransferCancelled), call.Type)
	assert.Equal(t, h.adminID, call.UserID, "the former target must be notified of the cancel")
}

func TestTransferService_DeclineTransferOwnership_EmitsDeclinedNotification(t *testing.T) {
	h := buildMembershipHarness(t)

	_, err := h.svc.InitiateTransferOwnership(context.Background(), h.ownerID, h.org.ID, h.adminID)
	require.NoError(t, err)
	require.Equal(t, 1, h.sender.count())

	err = h.svc.DeclineTransferOwnership(context.Background(), h.adminID, h.org.ID)
	require.NoError(t, err)

	require.Equal(t, 2, h.sender.count())
	call := h.sender.last()
	assert.Equal(t, string(notificationdomain.TypeOrgTransferDeclined), call.Type)
	assert.Equal(t, h.ownerID, call.UserID, "current owner receives the decline")
}

func TestTransferService_AcceptTransferOwnership_EmitsAcceptedNotification(t *testing.T) {
	h := buildMembershipHarness(t)

	_, err := h.svc.InitiateTransferOwnership(context.Background(), h.ownerID, h.org.ID, h.adminID)
	require.NoError(t, err)

	_, err = h.svc.AcceptTransferOwnership(context.Background(), h.adminID, h.org.ID)
	require.NoError(t, err)

	require.Equal(t, 2, h.sender.count())
	call := h.sender.last()
	assert.Equal(t, string(notificationdomain.TypeOrgTransferAccepted), call.Type)
	assert.Equal(t, h.ownerID, call.UserID, "old owner (now Admin) is notified of the handover")
}

// ---------------------------------------------------------------------------
// InvitationService trigger test (AcceptInvitation)
// ---------------------------------------------------------------------------

func TestInvitationService_AcceptInvitation_EmitsAcceptedNotification(t *testing.T) {
	ownerID := uuid.New()
	org, _ := buildOrgAndOwnerMember(t, ownerID)

	inv, err := organization.NewInvitation(organization.NewInvitationInput{
		OrganizationID:  org.ID,
		Email:           "newbie@example.test",
		FirstName:       "Newbie",
		LastName:        "Joiner",
		Title:           "Junior",
		Role:            organization.RoleMember,
		InvitedByUserID: ownerID,
	})
	require.NoError(t, err)

	invRepo := newTrackingInvitationRepo()
	invRepo.storedInvitations[inv.ID] = inv

	orgs := &mockOrgRepo{
		findByIDFn: func(_ context.Context, _ uuid.UUID) (*organization.Organization, error) {
			return org, nil
		},
	}

	sender := &recordingSender{}
	svc := NewInvitationService(InvitationServiceDeps{
		Orgs:          orgs,
		Members:       &mockMemberRepo{},
		Invitations:   invRepo,
		Users:         &mockUserRepoForInvites{},
		Hasher:        &mockHasher{},
		Email:         &mockEmailForInvites{},
		RateLimiter:   &mockInvitationRateLimiter{allowed: true},
		Notifications: sender,
		FrontendURL:   "https://app.example.test",
	})

	result, err := svc.AcceptInvitation(context.Background(), AcceptInvitationInput{
		Token:    inv.Token,
		Password: "StrongPass1!",
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	require.Equal(t, 1, sender.count())
	call := sender.last()
	assert.Equal(t, string(notificationdomain.TypeOrgInvitationAccepted), call.Type)
	assert.Equal(t, ownerID, call.UserID, "inviter receives the acceptance notification")

	var payload map[string]any
	require.NoError(t, json.Unmarshal(call.Data, &payload))
	assert.Equal(t, "Newbie Joiner", payload["new_member_name"])
	assert.Equal(t, inv.ID.String(), payload["invitation_id"])
}
