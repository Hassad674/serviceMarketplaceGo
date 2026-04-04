package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// CreditBonusLogEntry represents a credit bonus audit record.
type CreditBonusLogEntry struct {
	ID                    uuid.UUID
	ProviderID            uuid.UUID
	ClientID              uuid.UUID
	ProposalID            uuid.UUID
	ClientCardFingerprint string
	CreditsAwarded        int
	Status                string // "awarded", "blocked", "pending_review"
	BlockReason           string
	ProposalCreatedAt     time.Time
	ProposalPaidAt        time.Time
	CreatedAt             time.Time
}

// CreditBonusLogRepository manages credit bonus log entries for fraud detection.
type CreditBonusLogRepository interface {
	Insert(ctx context.Context, entry *CreditBonusLogEntry) error
	CountByProviderAndClient(ctx context.Context, providerID, clientID uuid.UUID, since time.Time) (int, error)
	ListPendingReview(ctx context.Context, cursor string, limit int) ([]*CreditBonusLogEntry, string, error)
	ListAll(ctx context.Context, cursor string, limit int) ([]*CreditBonusLogEntry, string, error)
	GetByID(ctx context.Context, id uuid.UUID) (*CreditBonusLogEntry, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string, creditsAwarded int) error
}
