package payment

import (
	"marketplace-backend/internal/domain/billing"
)

// FeePreviewResult bundles the pure fee calculation with two flags the
// UI acts on.
//
// ViewerIsProvider answers "would the authenticated user pay this fee
// on a proposal against the given recipient?" — the UI hides the
// preview entirely when this is false so a client never sees the
// prestataire's cost structure.
//
// ViewerIsSubscribed answers "does the user currently have Premium?" —
// when true, Billing.FeeCents is already zeroed by the service so the
// caller can render the summary as-is. The flag lets the UI show a
// Premium badge / highlight differently without recomputing the grid.
type FeePreviewResult struct {
	Billing            billing.Result
	ViewerIsProvider   bool
	ViewerIsSubscribed bool
}

// WalletOverview holds the provider's wallet state plus the apporteur's
// commission state when the viewer is a referrer. The two sides are
// independent — a user can have both a provider role (their own
// payouts) and be an apporteur (commissions on referrals they made).
// Frontend renders two sections when both are non-empty.
type WalletOverview struct {
	StripeAccountID   string         `json:"stripe_account_id"`
	ChargesEnabled    bool           `json:"charges_enabled"`
	PayoutsEnabled    bool           `json:"payouts_enabled"`
	EscrowAmount      int64          `json:"escrow_amount"`
	AvailableAmount   int64          `json:"available_amount"`
	TransferredAmount int64          `json:"transferred_amount"`
	Records           []WalletRecord `json:"records"`
	// Referral commission side — populated only when the viewer is an
	// apporteur with commissions. Zero-valued otherwise (UI hides the
	// section when pending+paid+clawed_back == 0).
	Commissions       CommissionWallet         `json:"commissions"`
	CommissionRecords []WalletCommissionRecord `json:"commission_records"`
}

// WalletRecord is one row of the provider's payment history: the
// money held in escrow / transferred / failed for a single milestone
// payment.
type WalletRecord struct {
	// ID is the payment_record row id — unique per (proposal, milestone)
	// pair. Exposed so the UI can use a stable React/Flutter key: a
	// proposal with N milestones produces N records that share the same
	// proposal_id, so proposal_id alone is NOT a valid key.
	ID             string `json:"id"`
	ProposalID     string `json:"proposal_id"`
	MilestoneID    string `json:"milestone_id,omitempty"`
	ProposalAmount int64  `json:"proposal_amount"`
	PlatformFee    int64  `json:"platform_fee"`
	ProviderPayout int64  `json:"provider_payout"`
	PaymentStatus  string `json:"payment_status"`
	TransferStatus string `json:"transfer_status"`
	MissionStatus  string `json:"mission_status"` // populated by wallet handler
	CreatedAt      string `json:"created_at"`
}

// CommissionWallet is the aggregate apporteur view: totals grouped by
// commission status. Mirrors the grammar of the provider-side cards
// (escrow / available / transferred) so the UI can reuse the same
// layout for both.
type CommissionWallet struct {
	PendingCents    int64  `json:"pending_cents"`
	PendingKYCCents int64  `json:"pending_kyc_cents"`
	PaidCents       int64  `json:"paid_cents"`
	ClawedBackCents int64  `json:"clawed_back_cents"`
	Currency        string `json:"currency"`
}

// WalletCommissionRecord is one row of the apporteur's commission
// history, ordered newest first by the service layer. Carries enough
// context (referral_id, proposal_id) for the UI to deep-link to the
// relevant referral / project.
type WalletCommissionRecord struct {
	ID               string `json:"id"`
	ReferralID       string `json:"referral_id,omitempty"`
	ProposalID       string `json:"proposal_id,omitempty"`
	MilestoneID      string `json:"milestone_id,omitempty"`
	GrossAmountCents int64  `json:"gross_amount_cents"`
	CommissionCents  int64  `json:"commission_cents"`
	Currency         string `json:"currency"`
	Status           string `json:"status"`
	StripeTransferID string `json:"stripe_transfer_id,omitempty"`
	PaidAt           string `json:"paid_at,omitempty"`
	ClawedBackAt     string `json:"clawed_back_at,omitempty"`
	CreatedAt        string `json:"created_at"`
}

// PayoutResult is returned by RequestPayout / RetryFailedTransfer so
// the wallet handler can render a clear status message:
//   - "transferred" — the bank-leg payout was issued.
//   - "transferred_pending_bank" — the platform→connected transfer
//     succeeded but the bank-leg payout failed; funds are safe on the
//     connected account and a retry is possible.
//   - "nothing_to_transfer" — no records were eligible.
type PayoutResult struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}
