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

const platformFeePercent = 5

// NewPaymentRecord creates a payment record and computes all fee amounts.
// stripeFeeAmount is the estimated or actual Stripe processing fee.
//
// Phase 4: every payment is scoped to a specific milestone, not the
// whole proposal. The milestone_id column in payment_records is NOT
// NULL, so a zero milestoneID will trip the DB constraint and fail
// loudly at insert time — that is intentional.
func NewPaymentRecord(proposalID, milestoneID, clientID, providerID uuid.UUID, proposalAmount, stripeFeeAmount int64) *PaymentRecord {
	platformFee := proposalAmount * platformFeePercent / 100

	now := time.Now()
	return &PaymentRecord{
		ID:                uuid.New(),
		ProposalID:        proposalID,
		MilestoneID:       milestoneID,
		ClientID:          clientID,
		ProviderID:        providerID,
		ProposalAmount:    proposalAmount,
		StripeFeeAmount:   stripeFeeAmount,
		PlatformFeeAmount: platformFee,
		ClientTotalAmount: proposalAmount + stripeFeeAmount,
		ProviderPayout:    proposalAmount - platformFee,
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

func (r *PaymentRecord) MarkFailed() {
	r.Status = RecordStatusFailed
	r.UpdatedAt = time.Now()
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
func (r *PaymentRecord) ApplyDisputeResolution(providerAmount int64, transferID string) {
	r.ProviderPayout = providerAmount
	r.TransferStatus = TransferCompleted
	if transferID != "" {
		r.StripeTransferID = transferID
	}
	now := time.Now()
	r.TransferredAt = &now
	r.UpdatedAt = now
}

// MarkRefunded marks the entire payment as refunded (full refund to client).
func (r *PaymentRecord) MarkRefunded() {
	r.Status = RecordStatusRefunded
	r.UpdatedAt = time.Now()
}

// EstimateStripeFee estimates the Stripe processing fee for European cards.
// Stripe EU rate: 1.5% + 0.25€ (25 centimes).
func EstimateStripeFee(proposalAmount int64) int64 {
	// Fee = ceil((amount + 25) / (1 - 0.015)) - amount
	// Simplified: fee ≈ amount * 0.015 + 25, rounded up
	fee := (proposalAmount*15 + 999) / 1000 // ceil(amount * 1.5%)
	return fee + 25
}
