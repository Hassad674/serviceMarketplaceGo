package organization

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validInvitationInput() NewInvitationInput {
	return NewInvitationInput{
		OrganizationID:  uuid.New(),
		Email:           "Marie.Dupont@Example.COM ",
		FirstName:       "Marie",
		LastName:        "Dupont",
		Title:           "Office Manager",
		Role:            RoleMember,
		InvitedByUserID: uuid.New(),
		Duration:        0, // default
	}
}

func TestNewInvitation_Valid(t *testing.T) {
	in := validInvitationInput()
	inv, err := NewInvitation(in)
	require.NoError(t, err)
	require.NotNil(t, inv)

	// Email normalized
	assert.Equal(t, "marie.dupont@example.com", inv.Email)
	assert.Equal(t, "Marie", inv.FirstName)
	assert.Equal(t, "Dupont", inv.LastName)
	assert.Equal(t, "Office Manager", inv.Title)
	assert.Equal(t, RoleMember, inv.Role)
	assert.Equal(t, InvitationStatusPending, inv.Status)
	assert.Len(t, inv.Token, 64, "token should be 64 hex chars (32 bytes)")

	// Default duration applied
	assert.WithinDuration(t, time.Now().Add(DefaultInvitationDuration), inv.ExpiresAt, time.Second)
}

func TestNewInvitation_CustomDuration(t *testing.T) {
	in := validInvitationInput()
	in.Duration = 48 * time.Hour
	inv, err := NewInvitation(in)
	require.NoError(t, err)
	assert.WithinDuration(t, time.Now().Add(48*time.Hour), inv.ExpiresAt, time.Second)
}

func TestNewInvitation_CannotInviteAsOwner(t *testing.T) {
	in := validInvitationInput()
	in.Role = RoleOwner
	inv, err := NewInvitation(in)
	assert.ErrorIs(t, err, ErrCannotInviteAsOwner)
	assert.Nil(t, inv)
}

func TestNewInvitation_InvalidRole(t *testing.T) {
	in := validInvitationInput()
	in.Role = Role("boss")
	inv, err := NewInvitation(in)
	assert.ErrorIs(t, err, ErrInvalidRole)
	assert.Nil(t, inv)
}

func TestNewInvitation_InvalidEmail(t *testing.T) {
	tests := []struct {
		name  string
		email string
	}{
		{"empty", ""},
		{"no at", "noatsign.com"},
		{"no domain dot", "user@localhost"},
		{"trailing at", "user@"},
		{"leading at", "@example.com"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := validInvitationInput()
			in.Email = tt.email
			_, err := NewInvitation(in)
			assert.ErrorIs(t, err, ErrInvalidEmail)
		})
	}
}

func TestNewInvitation_MissingNames(t *testing.T) {
	in := validInvitationInput()
	in.FirstName = ""
	_, err := NewInvitation(in)
	assert.ErrorIs(t, err, ErrNameRequired)

	in = validInvitationInput()
	in.LastName = "   "
	_, err = NewInvitation(in)
	assert.ErrorIs(t, err, ErrNameRequired)
}

func TestNewInvitation_NameTooLong(t *testing.T) {
	in := validInvitationInput()
	in.FirstName = strings.Repeat("x", maxNameLength+1)
	_, err := NewInvitation(in)
	assert.ErrorIs(t, err, ErrNameTooLong)
}

func TestNewInvitation_TitleTooLong(t *testing.T) {
	in := validInvitationInput()
	in.Title = strings.Repeat("t", maxTitleLength+1)
	_, err := NewInvitation(in)
	assert.ErrorIs(t, err, ErrTitleTooLong)
}

func TestNewInvitation_TokenUniqueness(t *testing.T) {
	// Not a statistical proof but a sanity check: 100 successive invitations
	// must all yield distinct tokens.
	seen := make(map[string]bool, 100)
	for i := 0; i < 100; i++ {
		inv, err := NewInvitation(validInvitationInput())
		require.NoError(t, err)
		assert.False(t, seen[inv.Token], "token collision at iteration %d", i)
		seen[inv.Token] = true
	}
}

func TestInvitation_IsExpired(t *testing.T) {
	inv, _ := NewInvitation(validInvitationInput())

	// Fresh invitation
	assert.False(t, inv.IsExpired())

	// Force-expire by rewinding ExpiresAt
	inv.ExpiresAt = time.Now().Add(-time.Minute)
	assert.True(t, inv.IsExpired())

	// Non-pending statuses are not "expired"
	inv.Status = InvitationStatusAccepted
	assert.False(t, inv.IsExpired())
}

func TestInvitation_Accept(t *testing.T) {
	inv, _ := NewInvitation(validInvitationInput())

	err := inv.Accept()
	require.NoError(t, err)
	assert.Equal(t, InvitationStatusAccepted, inv.Status)
	assert.NotNil(t, inv.AcceptedAt)

	// Second accept fails
	err = inv.Accept()
	assert.ErrorIs(t, err, ErrInvitationAlreadyUsed)
}

func TestInvitation_AcceptCancelled(t *testing.T) {
	inv, _ := NewInvitation(validInvitationInput())
	require.NoError(t, inv.Cancel())

	err := inv.Accept()
	assert.ErrorIs(t, err, ErrInvitationCancelled)
}

func TestInvitation_AcceptExpired(t *testing.T) {
	inv, _ := NewInvitation(validInvitationInput())
	inv.ExpiresAt = time.Now().Add(-time.Hour)

	err := inv.Accept()
	assert.ErrorIs(t, err, ErrInvitationExpired)
	assert.Equal(t, InvitationStatusExpired, inv.Status,
		"accept on expired should flip the status so subsequent reads are consistent")
}

func TestInvitation_Cancel(t *testing.T) {
	inv, _ := NewInvitation(validInvitationInput())

	err := inv.Cancel()
	require.NoError(t, err)
	assert.Equal(t, InvitationStatusCancelled, inv.Status)
	assert.NotNil(t, inv.CancelledAt)

	// Second cancel fails
	err = inv.Cancel()
	assert.ErrorIs(t, err, ErrInvalidInvitationStatus)
}

func TestInvitation_MarkExpired(t *testing.T) {
	inv, _ := NewInvitation(validInvitationInput())
	err := inv.MarkExpired()
	require.NoError(t, err)
	assert.Equal(t, InvitationStatusExpired, inv.Status)

	err = inv.MarkExpired()
	assert.ErrorIs(t, err, ErrInvalidInvitationStatus)
}

func TestInvitationStatus_IsValid(t *testing.T) {
	assert.True(t, InvitationStatusPending.IsValid())
	assert.True(t, InvitationStatusAccepted.IsValid())
	assert.True(t, InvitationStatusCancelled.IsValid())
	assert.True(t, InvitationStatusExpired.IsValid())
	assert.False(t, InvitationStatus("ghost").IsValid())
}
