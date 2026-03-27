package call

import (
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusRinging  Status = "ringing"
	StatusActive   Status = "active"
	StatusDeclined Status = "declined"
	StatusMissed   Status = "missed"
	StatusEnded    Status = "ended"
)

func (s Status) IsValid() bool {
	switch s {
	case StatusRinging, StatusActive, StatusDeclined, StatusMissed, StatusEnded:
		return true
	}
	return false
}

type Type string

const (
	TypeAudio Type = "audio"
	TypeVideo Type = "video"
)

func (t Type) IsValid() bool {
	return t == TypeAudio || t == TypeVideo
}

type Call struct {
	ID             uuid.UUID
	ConversationID uuid.UUID
	InitiatorID    uuid.UUID
	RecipientID    uuid.UUID
	RoomName       string
	Status         Status
	Type           Type
	StartedAt      *time.Time
	Duration       int
	CreatedAt      time.Time
}

func New(conversationID, initiatorID, recipientID uuid.UUID, callType Type) (*Call, error) {
	if !callType.IsValid() {
		return nil, ErrInvalidCallType
	}
	if initiatorID == recipientID {
		return nil, ErrSelfCall
	}

	id := uuid.New()
	roomName := "call:" + id.String()

	return &Call{
		ID:             id,
		ConversationID: conversationID,
		InitiatorID:    initiatorID,
		RecipientID:    recipientID,
		RoomName:       roomName,
		Status:         StatusRinging,
		Type:           callType,
		CreatedAt:      time.Now(),
	}, nil
}

func (c *Call) Accept() error {
	if c.Status != StatusRinging {
		return ErrInvalidTransition
	}
	c.Status = StatusActive
	now := time.Now()
	c.StartedAt = &now
	return nil
}

func (c *Call) Decline() error {
	if c.Status != StatusRinging {
		return ErrInvalidTransition
	}
	c.Status = StatusDeclined
	return nil
}

func (c *Call) End(durationSec int) error {
	if c.Status != StatusActive && c.Status != StatusRinging {
		return ErrInvalidTransition
	}
	c.Status = StatusEnded
	c.Duration = durationSec
	return nil
}

func (c *Call) Miss() error {
	if c.Status != StatusRinging {
		return ErrInvalidTransition
	}
	c.Status = StatusMissed
	return nil
}
