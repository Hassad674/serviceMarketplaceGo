package notification

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNewNotification_Valid(t *testing.T) {
	userID := uuid.New()
	data := json.RawMessage(`{"proposal_id":"abc"}`)

	n, err := NewNotification(NewNotificationInput{
		UserID: userID,
		Type:   TypeProposalReceived,
		Title:  "New proposal",
		Body:   "You received a proposal",
		Data:   data,
	})

	assert.NoError(t, err)
	assert.NotNil(t, n)
	assert.Equal(t, userID, n.UserID)
	assert.Equal(t, TypeProposalReceived, n.Type)
	assert.Equal(t, "New proposal", n.Title)
	assert.Equal(t, "You received a proposal", n.Body)
	assert.JSONEq(t, `{"proposal_id":"abc"}`, string(n.Data))
	assert.Nil(t, n.ReadAt)
	assert.NotEqual(t, uuid.Nil, n.ID)
}

func TestNewNotification_DefaultData(t *testing.T) {
	n, err := NewNotification(NewNotificationInput{
		UserID: uuid.New(),
		Type:   TypeNewMessage,
		Title:  "New message",
	})

	assert.NoError(t, err)
	assert.NotNil(t, n)
	assert.JSONEq(t, `{}`, string(n.Data))
}

func TestNewNotification_Validation(t *testing.T) {
	validUserID := uuid.New()

	tests := []struct {
		name    string
		input   NewNotificationInput
		wantErr error
	}{
		{
			name: "missing user ID",
			input: NewNotificationInput{
				Type:  TypeProposalReceived,
				Title: "Title",
			},
			wantErr: ErrMissingUser,
		},
		{
			name: "invalid type",
			input: NewNotificationInput{
				UserID: validUserID,
				Type:   NotificationType("unknown_type"),
				Title:  "Title",
			},
			wantErr: ErrInvalidType,
		},
		{
			name: "empty title",
			input: NewNotificationInput{
				UserID: validUserID,
				Type:   TypeNewMessage,
				Title:  "",
			},
			wantErr: ErrEmptyTitle,
		},
		{
			name: "whitespace-only title",
			input: NewNotificationInput{
				UserID: validUserID,
				Type:   TypeNewMessage,
				Title:  "   ",
			},
			wantErr: ErrEmptyTitle,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n, err := NewNotification(tt.input)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.Nil(t, n)
		})
	}
}

func TestNotification_IsRead(t *testing.T) {
	t.Run("unread notification", func(t *testing.T) {
		n := &Notification{ReadAt: nil}
		assert.False(t, n.IsRead())
	})

	t.Run("read notification", func(t *testing.T) {
		now := time.Now()
		n := &Notification{ReadAt: &now}
		assert.True(t, n.IsRead())
	})
}

func TestDefaultPreferences_EmailOn(t *testing.T) {
	userID := uuid.New()
	emailOnTypes := []NotificationType{
		TypeProposalReceived,
		TypeProposalAccepted,
		TypeProposalDeclined,
		TypeProposalPaid,
		TypeCompletionRequested,
		TypeProposalCompleted,
		TypeSystemAnnouncement,
	}

	for _, nType := range emailOnTypes {
		t.Run(string(nType), func(t *testing.T) {
			prefs := DefaultPreferences(userID, nType)
			assert.Equal(t, userID, prefs.UserID)
			assert.Equal(t, nType, prefs.NotificationType)
			assert.True(t, prefs.InApp)
			assert.True(t, prefs.Push)
			assert.True(t, prefs.Email, "email should default to true for %s", nType)
		})
	}
}

func TestStripeRequirementsType_IsValid(t *testing.T) {
	assert.True(t, TypeStripeRequirements.IsValid(), "stripe_requirements should be a valid notification type")
}

func TestStripeAccountStatusType_IsValid(t *testing.T) {
	assert.True(t, TypeStripeAccountStatus.IsValid(), "stripe_account_status should be a valid notification type")
}

func TestStripeTypes_DefaultPreferences_EmailOn(t *testing.T) {
	userID := uuid.New()

	stripeTypes := []NotificationType{
		TypeStripeRequirements,
		TypeStripeAccountStatus,
	}

	for _, nType := range stripeTypes {
		t.Run(string(nType), func(t *testing.T) {
			prefs := DefaultPreferences(userID, nType)
			assert.Equal(t, userID, prefs.UserID)
			assert.Equal(t, nType, prefs.NotificationType)
			assert.True(t, prefs.InApp, "in_app should default to true for %s", nType)
			assert.True(t, prefs.Push, "push should default to true for %s", nType)
			assert.True(t, prefs.Email, "email should default to true for %s", nType)
		})
	}
}

func TestDefaultPreferences_EmailOff(t *testing.T) {
	userID := uuid.New()
	emailOffTypes := []NotificationType{
		TypeNewMessage,
		TypeProposalModified,
		TypeReviewReceived,
	}

	for _, nType := range emailOffTypes {
		t.Run(string(nType), func(t *testing.T) {
			prefs := DefaultPreferences(userID, nType)
			assert.Equal(t, userID, prefs.UserID)
			assert.Equal(t, nType, prefs.NotificationType)
			assert.True(t, prefs.InApp)
			assert.True(t, prefs.Push)
			assert.False(t, prefs.Email, "email should default to false for %s", nType)
		})
	}
}
