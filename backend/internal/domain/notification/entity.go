package notification

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
)

// NotificationType represents the kind of notification event.
type NotificationType string

const (
	TypeProposalReceived    NotificationType = "proposal_received"
	TypeProposalAccepted    NotificationType = "proposal_accepted"
	TypeProposalDeclined    NotificationType = "proposal_declined"
	TypeProposalModified    NotificationType = "proposal_modified"
	TypeProposalPaid        NotificationType = "proposal_paid"
	TypeCompletionRequested NotificationType = "completion_requested"
	TypeProposalCompleted   NotificationType = "proposal_completed"
	TypeReviewReceived      NotificationType = "review_received"
	TypeNewMessage          NotificationType = "new_message"
	TypeSystemAnnouncement  NotificationType = "system_announcement"
	TypeStripeRequirements  NotificationType = "stripe_requirements"
	TypeStripeAccountStatus NotificationType = "stripe_account_status"
	TypeKYCReminder         NotificationType = "kyc_reminder"
	TypeKYCRestriction      NotificationType = "kyc_restriction"
	TypeKYCUnlocked         NotificationType = "kyc_unlocked"

	TypeDisputeOpened                NotificationType = "dispute_opened"
	TypeDisputeCounterProposal       NotificationType = "dispute_counter_proposal"
	TypeDisputeCounterRejected       NotificationType = "dispute_counter_rejected"
	TypeDisputeEscalated             NotificationType = "dispute_escalated"
	TypeDisputeResolved              NotificationType = "dispute_resolved"
	TypeDisputeCancelled             NotificationType = "dispute_cancelled"
	TypeDisputeAutoResolved          NotificationType = "dispute_auto_resolved"
	TypeDisputeCancellationRequested NotificationType = "dispute_cancellation_requested"
	TypeDisputeCancellationRefused   NotificationType = "dispute_cancellation_refused"
)

var validTypes = map[NotificationType]bool{
	TypeProposalReceived:    true,
	TypeProposalAccepted:    true,
	TypeProposalDeclined:    true,
	TypeProposalModified:    true,
	TypeProposalPaid:        true,
	TypeCompletionRequested: true,
	TypeProposalCompleted:   true,
	TypeReviewReceived:      true,
	TypeNewMessage:          true,
	TypeSystemAnnouncement:  true,
	TypeStripeRequirements:  true,
	TypeStripeAccountStatus: true,
	TypeKYCReminder:         true,
	TypeKYCRestriction:      true,
	TypeKYCUnlocked:         true,

	TypeDisputeOpened:                true,
	TypeDisputeCounterProposal:       true,
	TypeDisputeCounterRejected:       true,
	TypeDisputeEscalated:             true,
	TypeDisputeResolved:              true,
	TypeDisputeCancelled:             true,
	TypeDisputeAutoResolved:          true,
	TypeDisputeCancellationRequested: true,
	TypeDisputeCancellationRefused:   true,
}

// IsValid checks if the notification type is recognised.
func (t NotificationType) IsValid() bool {
	return validTypes[t]
}

// Notification represents a persisted user notification.
type Notification struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Type      NotificationType
	Title     string
	Body      string
	Data      json.RawMessage
	ReadAt    *time.Time
	CreatedAt time.Time
}

// NewNotificationInput groups parameters for creating a Notification.
type NewNotificationInput struct {
	UserID uuid.UUID
	Type   NotificationType
	Title  string
	Body   string
	Data   json.RawMessage
}

// NewNotification creates a validated Notification.
func NewNotification(in NewNotificationInput) (*Notification, error) {
	if in.UserID == uuid.Nil {
		return nil, ErrMissingUser
	}
	if !in.Type.IsValid() {
		return nil, ErrInvalidType
	}
	title := strings.TrimSpace(in.Title)
	if title == "" {
		return nil, ErrEmptyTitle
	}
	data := in.Data
	if len(data) == 0 {
		data = json.RawMessage(`{}`)
	}

	return &Notification{
		ID:        uuid.New(),
		UserID:    in.UserID,
		Type:      in.Type,
		Title:     title,
		Body:      strings.TrimSpace(in.Body),
		Data:      data,
		CreatedAt: time.Now(),
	}, nil
}

// IsRead returns whether the notification has been read.
func (n *Notification) IsRead() bool {
	return n.ReadAt != nil
}

// Preferences stores per-type, per-channel notification preferences for a user.
type Preferences struct {
	UserID           uuid.UUID
	NotificationType NotificationType
	InApp            bool
	Push             bool
	Email            bool
}

// DefaultPreferences returns the default preferences for a given type.
func DefaultPreferences(userID uuid.UUID, nType NotificationType) *Preferences {
	emailDefault := false
	// Proposal-related types default to email ON
	switch nType {
	case TypeProposalReceived, TypeProposalAccepted, TypeProposalDeclined,
		TypeProposalPaid, TypeCompletionRequested, TypeProposalCompleted,
		TypeSystemAnnouncement, TypeStripeRequirements, TypeStripeAccountStatus,
		TypeKYCReminder, TypeKYCRestriction, TypeKYCUnlocked,
		TypeDisputeOpened, TypeDisputeCounterProposal, TypeDisputeCounterRejected,
		TypeDisputeEscalated, TypeDisputeResolved, TypeDisputeCancelled,
		TypeDisputeAutoResolved, TypeDisputeCancellationRequested,
		TypeDisputeCancellationRefused:
		emailDefault = true
	}
	return &Preferences{
		UserID:           userID,
		NotificationType: nType,
		InApp:            true,
		Push:             true,
		Email:            emailDefault,
	}
}

// DeviceToken represents a registered mobile device for push notifications.
type DeviceToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Token     string
	Platform  string
	CreatedAt time.Time
}
