package organization

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOrganization_ValidInputs(t *testing.T) {
	owner := uuid.New()

	tests := []struct {
		name    string
		orgType OrgType
	}{
		{"agency", OrgTypeAgency},
		{"enterprise", OrgTypeEnterprise},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			org, err := NewOrganization(owner, tt.orgType)
			require.NoError(t, err)
			require.NotNil(t, org)
			assert.NotEqual(t, uuid.Nil, org.ID)
			assert.Equal(t, owner, org.OwnerUserID)
			assert.Equal(t, tt.orgType, org.Type)
			assert.False(t, org.IsTransferPending())
			assert.WithinDuration(t, time.Now(), org.CreatedAt, time.Second)
			assert.Equal(t, org.CreatedAt, org.UpdatedAt)
		})
	}
}

func TestNewOrganization_InvalidInputs(t *testing.T) {
	tests := []struct {
		name    string
		owner   uuid.UUID
		orgType OrgType
		wantErr error
	}{
		{"nil owner", uuid.Nil, OrgTypeAgency, ErrNameRequired},
		{"invalid type", uuid.New(), OrgType("marketplace"), ErrInvalidOrgType},
		{"empty type", uuid.New(), OrgType(""), ErrInvalidOrgType},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			org, err := NewOrganization(tt.owner, tt.orgType)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.Nil(t, org)
		})
	}
}

func TestOrganization_InitiateTransfer(t *testing.T) {
	owner := uuid.New()
	org, err := NewOrganization(owner, OrgTypeAgency)
	require.NoError(t, err)

	target := uuid.New()
	err = org.InitiateTransfer(target, 7*24*time.Hour)
	require.NoError(t, err)

	assert.True(t, org.IsTransferPending())
	assert.False(t, org.IsTransferExpired())
	require.NotNil(t, org.PendingTransferToUserID)
	assert.Equal(t, target, *org.PendingTransferToUserID)
	require.NotNil(t, org.PendingTransferExpiresAt)
	assert.WithinDuration(t, time.Now().Add(7*24*time.Hour), *org.PendingTransferExpiresAt, time.Second)
}

func TestOrganization_InitiateTransfer_Invalid(t *testing.T) {
	owner := uuid.New()
	org, _ := NewOrganization(owner, OrgTypeAgency)

	// Cannot transfer to self
	err := org.InitiateTransfer(owner, time.Hour)
	assert.ErrorIs(t, err, ErrCannotTransferToSelf)

	// Cannot transfer to nil
	err = org.InitiateTransfer(uuid.Nil, time.Hour)
	assert.ErrorIs(t, err, ErrTransferTargetInvalid)

	// Cannot initiate two transfers at once
	require.NoError(t, org.InitiateTransfer(uuid.New(), time.Hour))
	err = org.InitiateTransfer(uuid.New(), time.Hour)
	assert.ErrorIs(t, err, ErrTransferAlreadyPending)
}

func TestOrganization_CancelTransfer(t *testing.T) {
	owner := uuid.New()
	org, _ := NewOrganization(owner, OrgTypeAgency)

	// Cancel on empty state is a no-op
	org.CancelTransfer()
	assert.False(t, org.IsTransferPending())

	// Cancel after initiation clears everything
	require.NoError(t, org.InitiateTransfer(uuid.New(), time.Hour))
	require.True(t, org.IsTransferPending())
	org.CancelTransfer()
	assert.False(t, org.IsTransferPending())
	assert.Nil(t, org.PendingTransferToUserID)
	assert.Nil(t, org.PendingTransferInitiatedAt)
	assert.Nil(t, org.PendingTransferExpiresAt)
}

func TestOrganization_CompleteTransfer(t *testing.T) {
	originalOwner := uuid.New()
	newOwner := uuid.New()
	org, _ := NewOrganization(originalOwner, OrgTypeAgency)
	require.NoError(t, org.InitiateTransfer(newOwner, time.Hour))

	err := org.CompleteTransfer(newOwner)
	require.NoError(t, err)

	assert.Equal(t, newOwner, org.OwnerUserID)
	assert.False(t, org.IsTransferPending())
}

func TestOrganization_CompleteTransfer_Errors(t *testing.T) {
	owner := uuid.New()
	target := uuid.New()

	t.Run("no pending transfer", func(t *testing.T) {
		org, _ := NewOrganization(owner, OrgTypeAgency)
		err := org.CompleteTransfer(target)
		assert.ErrorIs(t, err, ErrNoPendingTransfer)
	})

	t.Run("wrong accepter", func(t *testing.T) {
		org, _ := NewOrganization(owner, OrgTypeAgency)
		_ = org.InitiateTransfer(target, time.Hour)
		wrongUser := uuid.New()
		err := org.CompleteTransfer(wrongUser)
		assert.ErrorIs(t, err, ErrTransferTargetInvalid)
	})

	t.Run("expired", func(t *testing.T) {
		org, _ := NewOrganization(owner, OrgTypeAgency)
		_ = org.InitiateTransfer(target, -time.Hour) // already expired
		assert.True(t, org.IsTransferExpired())
		err := org.CompleteTransfer(target)
		assert.ErrorIs(t, err, ErrTransferExpired)
	})
}

func TestOrgType_IsValid(t *testing.T) {
	assert.True(t, OrgTypeAgency.IsValid())
	assert.True(t, OrgTypeEnterprise.IsValid())
	assert.False(t, OrgType("").IsValid())
	assert.False(t, OrgType("provider").IsValid())
}
