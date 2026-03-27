package proposal

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validInput() NewProposalInput {
	return NewProposalInput{
		ConversationID: uuid.New(),
		SenderID:       uuid.New(),
		RecipientID:    uuid.New(),
		Title:          "Website redesign",
		Description:    "Full redesign of the corporate website",
		Amount:         150000,
		ClientID:       uuid.New(),
		ProviderID:     uuid.New(),
		Version:        1,
	}
}

func TestNewProposal(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*NewProposalInput)
		wantErr error
	}{
		{
			name:   "valid input",
			modify: func(_ *NewProposalInput) {},
		},
		{
			name:    "empty title",
			modify:  func(i *NewProposalInput) { i.Title = "" },
			wantErr: ErrEmptyTitle,
		},
		{
			name:    "empty description",
			modify:  func(i *NewProposalInput) { i.Description = "" },
			wantErr: ErrEmptyDescription,
		},
		{
			name:    "zero amount",
			modify:  func(i *NewProposalInput) { i.Amount = 0 },
			wantErr: ErrInvalidAmount,
		},
		{
			name:    "negative amount",
			modify:  func(i *NewProposalInput) { i.Amount = -500 },
			wantErr: ErrInvalidAmount,
		},
		{
			name: "same sender and recipient",
			modify: func(i *NewProposalInput) {
				shared := uuid.New()
				i.SenderID = shared
				i.RecipientID = shared
			},
			wantErr: ErrSameUser,
		},
		{
			name: "same client and provider",
			modify: func(i *NewProposalInput) {
				shared := uuid.New()
				i.ClientID = shared
				i.ProviderID = shared
			},
			wantErr: ErrSameUser,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := validInput()
			tt.modify(&input)

			p, err := NewProposal(input)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, p)
			} else {
				require.NoError(t, err)
				require.NotNil(t, p)
				assert.NotEqual(t, uuid.Nil, p.ID)
				assert.Equal(t, StatusPending, p.Status)
				assert.Equal(t, input.Title, p.Title)
				assert.Equal(t, input.Description, p.Description)
				assert.Equal(t, input.Amount, p.Amount)
				assert.Equal(t, 1, p.Version)
				assert.False(t, p.CreatedAt.IsZero())
				assert.False(t, p.UpdatedAt.IsZero())
				assert.Nil(t, p.AcceptedAt)
				assert.Nil(t, p.DeclinedAt)
				assert.Nil(t, p.PaidAt)
				assert.Nil(t, p.CompletedAt)
			}
		})
	}
}

func TestNewProposal_DefaultVersion(t *testing.T) {
	input := validInput()
	input.Version = 0

	p, err := NewProposal(input)

	require.NoError(t, err)
	assert.Equal(t, 1, p.Version)
}

func TestNewProposal_WithDeadline(t *testing.T) {
	input := validInput()
	deadline := time.Now().Add(30 * 24 * time.Hour)
	input.Deadline = &deadline

	p, err := NewProposal(input)

	require.NoError(t, err)
	require.NotNil(t, p.Deadline)
	assert.Equal(t, deadline, *p.Deadline)
}

func TestNewProposal_WithParentID(t *testing.T) {
	input := validInput()
	parentID := uuid.New()
	input.ParentID = &parentID
	input.Version = 2

	p, err := NewProposal(input)

	require.NoError(t, err)
	require.NotNil(t, p.ParentID)
	assert.Equal(t, parentID, *p.ParentID)
	assert.Equal(t, 2, p.Version)
}

func TestDetermineRoles(t *testing.T) {
	senderID := uuid.New()
	recipientID := uuid.New()

	tests := []struct {
		name           string
		senderRole     string
		recipientRole  string
		wantClientID   uuid.UUID
		wantProviderID uuid.UUID
		wantErr        error
	}{
		{
			name:           "enterprise sends to provider",
			senderRole:     "enterprise",
			recipientRole:  "provider",
			wantClientID:   senderID,
			wantProviderID: recipientID,
		},
		{
			name:           "provider sends to enterprise",
			senderRole:     "provider",
			recipientRole:  "enterprise",
			wantClientID:   recipientID,
			wantProviderID: senderID,
		},
		{
			name:           "enterprise sends to agency",
			senderRole:     "enterprise",
			recipientRole:  "agency",
			wantClientID:   senderID,
			wantProviderID: recipientID,
		},
		{
			name:           "agency sends to enterprise",
			senderRole:     "agency",
			recipientRole:  "enterprise",
			wantClientID:   recipientID,
			wantProviderID: senderID,
		},
		{
			name:           "agency sends to provider",
			senderRole:     "agency",
			recipientRole:  "provider",
			wantClientID:   senderID,
			wantProviderID: recipientID,
		},
		{
			name:           "provider sends to agency",
			senderRole:     "provider",
			recipientRole:  "agency",
			wantClientID:   recipientID,
			wantProviderID: senderID,
		},
		{
			name:          "provider to provider is invalid",
			senderRole:    "provider",
			recipientRole: "provider",
			wantErr:       ErrInvalidRoleCombination,
		},
		{
			name:          "enterprise to enterprise is invalid",
			senderRole:    "enterprise",
			recipientRole: "enterprise",
			wantErr:       ErrInvalidRoleCombination,
		},
		{
			name:          "agency to agency is invalid",
			senderRole:    "agency",
			recipientRole: "agency",
			wantErr:       ErrInvalidRoleCombination,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clientID, providerID, err := DetermineRoles(
				senderID, tt.senderRole,
				recipientID, tt.recipientRole,
			)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Equal(t, uuid.Nil, clientID)
				assert.Equal(t, uuid.Nil, providerID)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantClientID, clientID)
				assert.Equal(t, tt.wantProviderID, providerID)
			}
		})
	}
}

func newPendingProposal() *Proposal {
	sender := uuid.New()
	recipient := uuid.New()
	now := time.Now()
	return &Proposal{
		ID:          uuid.New(),
		SenderID:    sender,
		RecipientID: recipient,
		Status:      StatusPending,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

func TestProposal_Accept(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*Proposal) uuid.UUID
		wantErr error
	}{
		{
			name: "pending proposal accepted by recipient",
			setup: func(p *Proposal) uuid.UUID {
				return p.RecipientID
			},
		},
		{
			name: "sender cannot accept own proposal",
			setup: func(p *Proposal) uuid.UUID {
				return p.SenderID
			},
			wantErr: ErrNotAuthorized,
		},
		{
			name: "third party cannot accept",
			setup: func(p *Proposal) uuid.UUID {
				return uuid.New()
			},
			wantErr: ErrNotAuthorized,
		},
		{
			name: "already accepted proposal",
			setup: func(p *Proposal) uuid.UUID {
				p.Status = StatusAccepted
				return p.RecipientID
			},
			wantErr: ErrInvalidStatus,
		},
		{
			name: "already declined proposal",
			setup: func(p *Proposal) uuid.UUID {
				p.Status = StatusDeclined
				return p.RecipientID
			},
			wantErr: ErrInvalidStatus,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newPendingProposal()
			userID := tt.setup(p)

			err := p.Accept(userID)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, StatusAccepted, p.Status)
				assert.NotNil(t, p.AcceptedAt)
			}
		})
	}
}

func TestProposal_Decline(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*Proposal) uuid.UUID
		wantErr error
	}{
		{
			name: "pending proposal declined by recipient",
			setup: func(p *Proposal) uuid.UUID {
				return p.RecipientID
			},
		},
		{
			name: "sender cannot decline",
			setup: func(p *Proposal) uuid.UUID {
				return p.SenderID
			},
			wantErr: ErrNotAuthorized,
		},
		{
			name: "not pending",
			setup: func(p *Proposal) uuid.UUID {
				p.Status = StatusAccepted
				return p.RecipientID
			},
			wantErr: ErrInvalidStatus,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newPendingProposal()
			userID := tt.setup(p)

			err := p.Decline(userID)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, StatusDeclined, p.Status)
				assert.NotNil(t, p.DeclinedAt)
			}
		})
	}
}

func TestProposal_Withdraw(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*Proposal) uuid.UUID
		wantErr error
	}{
		{
			name: "sender withdraws pending proposal",
			setup: func(p *Proposal) uuid.UUID {
				return p.SenderID
			},
		},
		{
			name: "recipient cannot withdraw",
			setup: func(p *Proposal) uuid.UUID {
				return p.RecipientID
			},
			wantErr: ErrNotAuthorized,
		},
		{
			name: "third party cannot withdraw",
			setup: func(p *Proposal) uuid.UUID {
				return uuid.New()
			},
			wantErr: ErrNotAuthorized,
		},
		{
			name: "not pending",
			setup: func(p *Proposal) uuid.UUID {
				p.Status = StatusAccepted
				return p.SenderID
			},
			wantErr: ErrInvalidStatus,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newPendingProposal()
			userID := tt.setup(p)

			err := p.Withdraw(userID)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, StatusWithdrawn, p.Status)
			}
		})
	}
}

func TestProposal_MarkPaid(t *testing.T) {
	tests := []struct {
		name    string
		status  ProposalStatus
		wantErr error
	}{
		{name: "accepted to paid", status: StatusAccepted},
		{name: "pending cannot be paid", status: StatusPending, wantErr: ErrInvalidStatus},
		{name: "declined cannot be paid", status: StatusDeclined, wantErr: ErrInvalidStatus},
		{name: "already paid", status: StatusPaid, wantErr: ErrInvalidStatus},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newPendingProposal()
			p.Status = tt.status

			err := p.MarkPaid()

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, StatusPaid, p.Status)
				assert.NotNil(t, p.PaidAt)
			}
		})
	}
}

func TestProposal_MarkActive(t *testing.T) {
	tests := []struct {
		name    string
		status  ProposalStatus
		wantErr error
	}{
		{name: "paid to active", status: StatusPaid},
		{name: "accepted cannot activate", status: StatusAccepted, wantErr: ErrInvalidStatus},
		{name: "pending cannot activate", status: StatusPending, wantErr: ErrInvalidStatus},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newPendingProposal()
			p.Status = tt.status

			err := p.MarkActive()

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, StatusActive, p.Status)
			}
		})
	}
}

func TestProposal_MarkCompleted(t *testing.T) {
	tests := []struct {
		name    string
		status  ProposalStatus
		wantErr error
	}{
		{name: "active to completed", status: StatusActive},
		{name: "paid cannot complete", status: StatusPaid, wantErr: ErrInvalidStatus},
		{name: "pending cannot complete", status: StatusPending, wantErr: ErrInvalidStatus},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newPendingProposal()
			p.Status = tt.status

			err := p.MarkCompleted()

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, StatusCompleted, p.Status)
				assert.NotNil(t, p.CompletedAt)
			}
		})
	}
}

func TestProposal_CanBeModifiedBy(t *testing.T) {
	tests := []struct {
		name   string
		setup  func(*Proposal) uuid.UUID
		expect bool
	}{
		{
			name: "pending proposal recipient can modify",
			setup: func(p *Proposal) uuid.UUID {
				return p.RecipientID
			},
			expect: true,
		},
		{
			name: "pending proposal sender cannot modify",
			setup: func(p *Proposal) uuid.UUID {
				return p.SenderID
			},
			expect: false,
		},
		{
			name: "third party cannot modify",
			setup: func(p *Proposal) uuid.UUID {
				return uuid.New()
			},
			expect: false,
		},
		{
			name: "accepted proposal cannot be modified",
			setup: func(p *Proposal) uuid.UUID {
				p.Status = StatusAccepted
				return p.RecipientID
			},
			expect: false,
		},
		{
			name: "declined proposal cannot be modified",
			setup: func(p *Proposal) uuid.UUID {
				p.Status = StatusDeclined
				return p.RecipientID
			},
			expect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newPendingProposal()
			userID := tt.setup(p)

			result := p.CanBeModifiedBy(userID)

			assert.Equal(t, tt.expect, result)
		})
	}
}

func TestProposalStatus_IsValid(t *testing.T) {
	tests := []struct {
		name    string
		status  ProposalStatus
		isValid bool
	}{
		{"pending is valid", StatusPending, true},
		{"accepted is valid", StatusAccepted, true},
		{"declined is valid", StatusDeclined, true},
		{"withdrawn is valid", StatusWithdrawn, true},
		{"paid is valid", StatusPaid, true},
		{"active is valid", StatusActive, true},
		{"completed is valid", StatusCompleted, true},
		{"empty is invalid", ProposalStatus(""), false},
		{"random is invalid", ProposalStatus("draft"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.isValid, tt.status.IsValid())
		})
	}
}
