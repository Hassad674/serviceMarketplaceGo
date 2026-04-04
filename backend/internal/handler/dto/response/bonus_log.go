package response

import (
	"time"

	"marketplace-backend/internal/port/repository"
)

// BonusLogResponse represents a credit bonus log entry for the admin panel.
type BonusLogResponse struct {
	ID                    string    `json:"id"`
	ProviderID            string    `json:"provider_id"`
	ClientID              string    `json:"client_id"`
	ProposalID            string    `json:"proposal_id"`
	ClientCardFingerprint string    `json:"client_card_fingerprint,omitempty"`
	CreditsAwarded        int       `json:"credits_awarded"`
	Status                string    `json:"status"`
	BlockReason           string    `json:"block_reason,omitempty"`
	ProposalCreatedAt     *string   `json:"proposal_created_at,omitempty"`
	ProposalPaidAt        time.Time `json:"proposal_paid_at"`
	CreatedAt             time.Time `json:"created_at"`
}

func NewBonusLogResponse(e *repository.CreditBonusLogEntry) BonusLogResponse {
	r := BonusLogResponse{
		ID:                    e.ID.String(),
		ProviderID:            e.ProviderID.String(),
		ClientID:              e.ClientID.String(),
		ProposalID:            e.ProposalID.String(),
		ClientCardFingerprint: e.ClientCardFingerprint,
		CreditsAwarded:        e.CreditsAwarded,
		Status:                e.Status,
		BlockReason:           e.BlockReason,
		ProposalPaidAt:        e.ProposalPaidAt,
		CreatedAt:             e.CreatedAt,
	}
	if !e.ProposalCreatedAt.IsZero() {
		s := e.ProposalCreatedAt.Format(time.RFC3339)
		r.ProposalCreatedAt = &s
	}
	return r
}
