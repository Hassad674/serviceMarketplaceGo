package referral

import (
	"time"

	"github.com/google/uuid"
)

// Attribution links a proposal (signed contract between the provider and the
// client of an active referral) back to that referral, so commissions on its
// milestones can be routed to the apporteur.
//
// Attribution is created lazily when a proposal is created and a matching
// active referral is found on the (provider, client) couple. Once the referral
// expires, NEW proposals between the same couple no longer get attributed —
// but EXISTING attributions remain valid and continue paying out commissions
// as their milestones are released. This is intentional: the apporteur did the
// work of introducing them, the contract was signed inside the exclusivity
// window, the deal is theirs.
//
// proposal_id is stored as a bare uuid.UUID without a foreign key reference
// because the project rule forbids cross-feature foreign keys (see CLAUDE.md
// "Modularity above all"). The attribution feature has no compile-time link
// to the proposal feature.
type Attribution struct {
	ID              uuid.UUID
	ReferralID      uuid.UUID
	ProposalID      uuid.UUID
	ProviderID      uuid.UUID
	ClientID        uuid.UUID
	RatePctSnapshot float64
	AttributedAt    time.Time
	// EndedAt is non-nil when the apporteur has explicitly terminated
	// the intro for this proposal. Once set, NEW milestones approved on
	// or after EndedAt MUST NOT generate commissions (gate enforced in
	// commission_distributor). Milestones approved BEFORE EndedAt keep
	// their commission rows — fair to the apporteur for work delivered
	// during the active window. Nil = active, attribution still earns.
	EndedAt *time.Time
}

// IsEnded reports whether the attribution has been explicitly
// terminated by the apporteur. Equivalent to a non-nil EndedAt; named
// method exists so callers (e.g. the commission gate) read as the
// business intent rather than poking at the pointer.
func (a *Attribution) IsEnded() bool {
	return a != nil && a.EndedAt != nil
}

// NewAttributionInput is the validated input for NewAttribution.
type NewAttributionInput struct {
	ReferralID      uuid.UUID
	ProposalID      uuid.UUID
	ProviderID      uuid.UUID
	ClientID        uuid.UUID
	RatePctSnapshot float64
}

// NewAttribution builds a validated Attribution row, snapshotting the rate
// at attribution time so subsequent edits to the parent referral cannot
// retroactively change the commission for a proposal already in flight.
func NewAttribution(input NewAttributionInput) (*Attribution, error) {
	if input.ReferralID == uuid.Nil || input.ProposalID == uuid.Nil ||
		input.ProviderID == uuid.Nil || input.ClientID == uuid.Nil {
		return nil, ErrNotAuthorized
	}
	if input.ProviderID == input.ClientID {
		return nil, ErrSelfReferral
	}
	if input.RatePctSnapshot < MinRatePct || input.RatePctSnapshot > MaxRatePct {
		return nil, ErrRateOutOfRange
	}
	return &Attribution{
		ID:              uuid.New(),
		ReferralID:      input.ReferralID,
		ProposalID:      input.ProposalID,
		ProviderID:      input.ProviderID,
		ClientID:        input.ClientID,
		RatePctSnapshot: input.RatePctSnapshot,
		AttributedAt:    time.Now().UTC(),
	}, nil
}
