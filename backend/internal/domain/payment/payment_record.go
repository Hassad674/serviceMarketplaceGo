package payment

import (
	"time"

	"github.com/google/uuid"
)

type PaymentRecordStatus string

const (
	RecordStatusPending   PaymentRecordStatus = "pending"
	RecordStatusSucceeded PaymentRecordStatus = "succeeded"
	RecordStatusFailed    PaymentRecordStatus = "failed"
	RecordStatusRefunded  PaymentRecordStatus = "refunded"
)

type TransferStatus string

const (
	TransferPending   TransferStatus = "pending"
	TransferCompleted TransferStatus = "completed"
	TransferFailed    TransferStatus = "failed"
)

type PaymentRecord struct {
	ID                    uuid.UUID
	ProposalID            uuid.UUID
	MilestoneID           uuid.UUID // phase 4: one payment_record per milestone
	ClientID              uuid.UUID
	ProviderID            uuid.UUID
	StripePaymentIntentID string
	StripeTransferID      string

	ProposalAmount    int64 // original proposal amount in centimes
	StripeFeeAmount   int64 // Stripe processing fee (charged to client)
	PlatformFeeAmount int64 // 5% commission (deducted from provider)
	ClientTotalAmount int64 // what the client actually pays
	ProviderPayout    int64 // what the provider receives

	Currency       string
	Status         PaymentRecordStatus
	TransferStatus TransferStatus

	PaidAt        *time.Time
	TransferredAt *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// NewPaymentRecord creates a payment record with explicit fee amounts.
//
// platformFeeAmount is computed by the caller (app layer) using the billing
// package's fee schedule — the domain no longer hardcodes a commission rate,
// so the fee structure is free to evolve (tiered flat fees, premium waivers,
// etc.) without touching this constructor. The fee MUST be pre-computed by
// the caller; passing 0 means "no platform fee" (e.g. premium subscriber).
//
// stripeFeeAmount is the estimated or actual Stripe processing fee, charged
// on top of the proposal amount to the client.
//
// Phase 4: every payment is scoped to a specific milestone, not the
// whole proposal. The milestone_id column in payment_records is NOT
// NULL, so a zero milestoneID will trip the DB constraint and fail
// loudly at insert time — that is intentional.
func NewPaymentRecord(proposalID, milestoneID, clientID, providerID uuid.UUID, proposalAmount, stripeFeeAmount, platformFeeAmount int64) *PaymentRecord {
	now := time.Now()
	return &PaymentRecord{
		ID:                uuid.New(),
		ProposalID:        proposalID,
		MilestoneID:       milestoneID,
		ClientID:          clientID,
		ProviderID:        providerID,
		ProposalAmount:    proposalAmount,
		StripeFeeAmount:   stripeFeeAmount,
		PlatformFeeAmount: platformFeeAmount,
		ClientTotalAmount: proposalAmount + stripeFeeAmount,
		ProviderPayout:    proposalAmount - platformFeeAmount,
		Currency:          "eur",
		Status:            RecordStatusPending,
		TransferStatus:    TransferPending,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
}

func (r *PaymentRecord) MarkPaid() error {
	if r.Status != RecordStatusPending {
		return ErrPaymentNotPending
	}
	r.Status = RecordStatusSucceeded
	now := time.Now()
	r.PaidAt = &now
	r.UpdatedAt = now
	return nil
}

// MarkFailed transitions a payment record from Pending to Failed.
//
// State guard (BUG-02): only Pending records can transition to Failed.
// A second call (e.g. webhook replay, double-event from Stripe) on an
// already-Succeeded or already-Refunded record returns a wrapped
// ErrInvalidStateTransition — silently overwriting Status to Failed
// would corrupt the audit trail and could trip downstream branches
// that assume succeeded ⇒ funds in escrow.
//
// The status is mutated in-place ONLY when the guard passes; on
// rejection, the record is unchanged so the caller can safely
// short-circuit and log without rolling back.
func (r *PaymentRecord) MarkFailed() error {
	if r.Status != RecordStatusPending {
		return &StateTransitionError{
			Method:         "MarkFailed",
			ExpectedStatus: RecordStatusPending,
			ActualStatus:   r.Status,
			ActualTransfer: r.TransferStatus,
		}
	}
	r.Status = RecordStatusFailed
	r.UpdatedAt = time.Now()
	return nil
}

func (r *PaymentRecord) MarkTransferred(transferID string) error {
	if r.Status != RecordStatusSucceeded {
		return ErrPaymentNotSucceeded
	}
	if r.TransferStatus != TransferPending {
		return ErrTransferAlreadyDone
	}
	r.StripeTransferID = transferID
	r.TransferStatus = TransferCompleted
	now := time.Now()
	r.TransferredAt = &now
	r.UpdatedAt = now
	return nil
}

func (r *PaymentRecord) MarkTransferFailed() {
	r.TransferStatus = TransferFailed
	r.UpdatedAt = time.Now()
}

// ApplyDisputeResolution updates the record after a dispute resolution.
// Sets the actual provider payout amount and marks the transfer as completed.
// If amount is 0 (full refund to client), no Stripe transfer is recorded.
//
// State guard (BUG-02): the record MUST be Status=Succeeded (funds in
// escrow on the platform) AND TransferStatus != Completed (transfer not
// already executed). A replay on an already-transferred record could
// otherwise overwrite ProviderPayout to 0 and lose the provider's money;
// a replay on a still-Pending record is also a logical bug — escrow funds
// must be cleared before any resolution is applied.
//
// The mutation is atomic on the in-memory copy: if the guard rejects the
// call, none of ProviderPayout, TransferStatus, StripeTransferID, or the
// timestamps change, and the caller can surface the error without rolling
// back any state.
func (r *PaymentRecord) ApplyDisputeResolution(providerAmount int64, transferID string) error {
	if r.Status != RecordStatusSucceeded || r.TransferStatus == TransferCompleted {
		return &StateTransitionError{
			Method:           "ApplyDisputeResolution",
			ExpectedStatus:   RecordStatusSucceeded,
			ActualStatus:     r.Status,
			ExpectedTransfer: TransferCompleted, // "transfer must NOT be Completed" — surfaced via the guard's text
			ActualTransfer:   r.TransferStatus,
		}
	}
	r.ProviderPayout = providerAmount
	r.TransferStatus = TransferCompleted
	if transferID != "" {
		r.StripeTransferID = transferID
	}
	now := time.Now()
	r.TransferredAt = &now
	r.UpdatedAt = now
	return nil
}

// MarkRefunded marks the entire payment as refunded (full refund to client).
//
// State guard (BUG-02): only Succeeded records can transition to Refunded —
// you cannot refund a payment that never cleared (Pending), one that
// already failed (Failed), or one that already refunded (Refunded). A
// replay on a non-Succeeded record returns a wrapped
// ErrInvalidStateTransition; the in-memory state is untouched so the
// caller can short-circuit without rollback.
func (r *PaymentRecord) MarkRefunded() error {
	if r.Status != RecordStatusSucceeded {
		return &StateTransitionError{
			Method:         "MarkRefunded",
			ExpectedStatus: RecordStatusSucceeded,
			ActualStatus:   r.Status,
			ActualTransfer: r.TransferStatus,
		}
	}
	r.Status = RecordStatusRefunded
	r.UpdatedAt = time.Now()
	return nil
}

// EstimateStripeFee estimates the Stripe processing fee for European cards.
// Stripe EU rate: 1.5% + 0.25€ (25 centimes).
func EstimateStripeFee(proposalAmount int64) int64 {
	// Fee = ceil((amount + 25) / (1 - 0.015)) - amount
	// Simplified: fee ≈ amount * 0.015 + 25, rounded up
	fee := (proposalAmount*15 + 999) / 1000 // ceil(amount * 1.5%)
	return fee + 25
}
