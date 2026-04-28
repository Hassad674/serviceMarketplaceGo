package service

import (
	"context"

	"github.com/google/uuid"
)

// PaymentProcessor is implemented by the payment app service and consumed
// by the proposal service. This avoids cross-feature imports.
type PaymentProcessor interface {
	// CreatePaymentIntent creates a Stripe PaymentIntent for a proposal.
	// Returns the client secret for Stripe Elements on the frontend.
	CreatePaymentIntent(ctx context.Context, input PaymentIntentInput) (*PaymentIntentOutput, error)

	// TransferToProvider transfers funds to the provider's connected account
	// for every pending milestone of the proposal. Used at macro completion
	// and by the outbox worker where no specific milestone is known. For
	// milestone-scoped releases (mid-project approve / auto-approve) callers
	// MUST use TransferMilestone instead — TransferToProvider iterates all
	// pending records of the proposal.
	TransferToProvider(ctx context.Context, proposalID uuid.UUID) error

	// TransferMilestone transfers funds for a single milestone record.
	// This is the primary release path for multi-milestone proposals —
	// the per-milestone release events (CompleteProposal mid-project,
	// AutoApproveMilestone) must call this so only the just-released
	// record is transferred and the referral commission hook fires
	// against the correct milestone.
	TransferMilestone(ctx context.Context, milestoneID uuid.UUID) error

	// HandlePaymentSucceeded processes a successful payment webhook.
	// Returns the proposal ID so the caller can transition the proposal.
	HandlePaymentSucceeded(ctx context.Context, paymentIntentID string) (proposalID uuid.UUID, err error)

	// TransferPartialToProvider transfers a specific amount (in centimes) to the
	// provider's connected account. Used for dispute partial resolutions.
	TransferPartialToProvider(ctx context.Context, proposalID uuid.UUID, amount int64) error

	// RefundToClient creates a partial or full refund on the original PaymentIntent.
	// amount is in centimes. Used for dispute resolutions.
	RefundToClient(ctx context.Context, proposalID uuid.UUID, amount int64) error

	// CanProviderReceivePayouts reports whether the given provider
	// organization has a Stripe Connect account that is ready to receive
	// transfers (account exists AND PayoutsEnabled == true).
	//
	// Used as a pre-check in the milestone-release path: callers MUST
	// reject the release when this returns false, otherwise the local
	// state flips to "released" while the Stripe transfer silently fails
	// — giving the client a "milestone paid" notification while the
	// money never leaves the platform.
	CanProviderReceivePayouts(ctx context.Context, providerOrgID uuid.UUID) (bool, error)

	// HasAutoPayoutConsent reports whether the org has previously
	// completed a successful manual payout via the wallet. Once true,
	// milestone releases auto-transfer instead of waiting on another
	// explicit "Retirer" click. False (the default) keeps every record
	// in TransferPending until the provider clicks themselves — exactly
	// the right posture for a fresh provider whose Stripe account has
	// not been proven to work yet.
	HasAutoPayoutConsent(ctx context.Context, providerOrgID uuid.UUID) (bool, error)
}

type PaymentIntentInput struct {
	ProposalID     uuid.UUID
	MilestoneID    uuid.UUID // phase 4: every payment is scoped to a single milestone
	ClientID       uuid.UUID
	ProviderID     uuid.UUID
	ProposalAmount int64 // centimes — the milestone's amount, not the proposal's total
}

type PaymentIntentOutput struct {
	ClientSecret    string
	PaymentRecordID uuid.UUID
	ProposalAmount  int64
	StripeFee       int64
	PlatformFee     int64
	ClientTotal     int64
	ProviderPayout  int64
}
