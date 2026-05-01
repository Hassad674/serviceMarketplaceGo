package repository

import (
	"context"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/payment"
)

type PaymentRecordRepository interface {
	Create(ctx context.Context, record *payment.PaymentRecord) error
	// GetByID returns a single record by its primary key. Used by the
	// retry-transfer flow where the UI holds the stable record id —
	// GetByProposalID would be wrong because a proposal can own N records
	// (one per milestone) and only returns the most recent.
	GetByID(ctx context.Context, id uuid.UUID) (*payment.PaymentRecord, error)

	// GetByIDForOrg fetches a payment record under the caller's
	// organization tenant context. The adapter wraps the read in
	// RunInTxWithTenant so the RLS policy on payment_records
	// (USING organization_id = current_setting('app.current_org_id'))
	// admits only rows owned by the caller's org. Returns
	// ErrPaymentRecordNotFound when the row does not exist OR when
	// the caller's org is not the record's owning org.
	//
	// User-facing app callers (retry transfer, payout request) MUST
	// use this method.
	GetByIDForOrg(ctx context.Context, id, callerOrgID uuid.UUID) (*payment.PaymentRecord, error)
	GetByProposalID(ctx context.Context, proposalID uuid.UUID) (*payment.PaymentRecord, error)
	// ListByProposalID returns every payment record owned by the proposal,
	// ordered by created_at ascending (oldest milestone first). Used by
	// TransferToProvider's iterator path so a macro-completion transfer
	// releases EVERY pending milestone of the proposal, not just the
	// most recently created one (which is what GetByProposalID returns).
	ListByProposalID(ctx context.Context, proposalID uuid.UUID) ([]*payment.PaymentRecord, error)
	// GetByMilestoneID is the phase-4 idempotency key for
	// CreatePaymentIntent — every payment is scoped to one milestone.
	GetByMilestoneID(ctx context.Context, milestoneID uuid.UUID) (*payment.PaymentRecord, error)
	GetByPaymentIntentID(ctx context.Context, paymentIntentID string) (*payment.PaymentRecord, error)
	ListByOrganization(ctx context.Context, orgID uuid.UUID) ([]*payment.PaymentRecord, error)
	Update(ctx context.Context, record *payment.PaymentRecord) error
}
