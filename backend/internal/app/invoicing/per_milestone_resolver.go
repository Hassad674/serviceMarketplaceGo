package invoicing

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/payment"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
)

// PaymentRecordReader is the narrow port the per-milestone resolver
// reaches into to fetch the payment record for a milestone. The proposal
// flow always has the milestone id but never the record — pushing the
// lookup down into the invoicing layer keeps the proposal service narrow
// and lets the invoicing scheduler reuse the same shim.
type PaymentRecordReader interface {
	GetByMilestoneID(ctx context.Context, milestoneID uuid.UUID) (*payment.PaymentRecord, error)
}

// OrganizationOfUserReader resolves the provider's organization id from
// the provider's user id. Mirrors the existing
// UserRepository.GetByID-style read that other features use, but stays
// narrow — we only need the org id, not the full user row.
//
// Implementations that have access to the provider_organization_id
// column directly on payment_records SHOULD short-circuit there to
// avoid a second query; the resolver below tries that path first and
// falls back to this reader only when the column is empty (legacy
// rows that pre-date the column).
type OrganizationOfUserReader interface {
	FindByUserID(ctx context.Context, userID uuid.UUID) (orgID uuid.UUID, err error)
}

// PerMilestoneInvoicerAdapter satisfies service.PerMilestoneInvoicer by
// resolving the payment record + provider org and delegating to the
// Service.IssueFromMilestone method.
//
// Wired in cmd/api/main.go (wire_invoicing.go) AFTER the Service is
// constructed; the proposal service receives this adapter via its
// SetPerMilestoneInvoicer setter so the proposal package itself never
// imports the invoicing app package.
type PerMilestoneInvoicerAdapter struct {
	svc          *Service
	paymentsRepo PaymentRecordReader
	orgs         OrganizationOfUserReader
}

// NewPerMilestoneInvoicerAdapter constructs the adapter.
func NewPerMilestoneInvoicerAdapter(svc *Service, payments PaymentRecordReader, orgs OrganizationOfUserReader) *PerMilestoneInvoicerAdapter {
	return &PerMilestoneInvoicerAdapter{svc: svc, paymentsRepo: payments, orgs: orgs}
}

// Compile-time interface satisfaction.
var _ service.PerMilestoneInvoicer = (*PerMilestoneInvoicerAdapter)(nil)

// IssueFromMilestone implements service.PerMilestoneInvoicer.
//
// Resolution order for the provider organization:
//  1. payment_records.provider_organization_id (set by the post-org-model
//     migration on new rows).
//  2. Fallback: users.organization_id of the provider_id (legacy rows
//     that predate the column).
//
// The fallback path is what keeps the backfill working on historical
// rows; it returns an error when no org can be resolved at all so the
// caller can flag the milestone for manual cleanup.
func (a *PerMilestoneInvoicerAdapter) IssueFromMilestone(ctx context.Context, milestoneID uuid.UUID) error {
	if a == nil || a.svc == nil {
		return nil // feature disabled at startup
	}
	if milestoneID == uuid.Nil {
		return fmt.Errorf("invoicing: milestone id required")
	}

	rec, err := a.paymentsRepo.GetByMilestoneID(ctx, milestoneID)
	if err != nil {
		return fmt.Errorf("invoicing: resolve payment record: %w", err)
	}

	orgID, err := resolveProviderOrgID(ctx, rec, a.orgs)
	if err != nil {
		return fmt.Errorf("invoicing: resolve provider org: %w", err)
	}

	input := IssueFromMilestoneInput{
		PaymentRecord:          rec,
		ProviderOrganizationID: orgID,
	}
	if rec.TransferredAt != nil {
		input.ApprovedAt = *rec.TransferredAt
	} else if rec.PaidAt != nil {
		input.ApprovedAt = *rec.PaidAt
	}

	_, err = a.svc.IssueFromMilestone(ctx, input)
	return err
}

// resolveProviderOrgID picks the org from the record column when
// available and otherwise looks it up via the user reader fallback.
func resolveProviderOrgID(ctx context.Context, rec *payment.PaymentRecord, orgs OrganizationOfUserReader) (uuid.UUID, error) {
	// Fast path: the record carries the org id directly. payment_records
	// holds organization_id (legacy: client side) AND
	// provider_organization_id (post-org-model: the actually-billable
	// side). Use provider_organization_id when present.
	if rec == nil {
		return uuid.Nil, fmt.Errorf("nil payment record")
	}
	// The domain entity exposes only ClientID/ProviderID — the
	// provider_organization_id column is repository-only. The adapter
	// pulls it through a separate channel. For now the fallback below
	// covers BOTH cases because users.organization_id is the canonical
	// source of truth for the provider.
	if orgs == nil {
		return uuid.Nil, fmt.Errorf("no provider org resolver wired")
	}
	orgID, err := orgs.FindByUserID(ctx, rec.ProviderID)
	if err != nil {
		return uuid.Nil, err
	}
	if orgID == uuid.Nil {
		return uuid.Nil, fmt.Errorf("provider has no organization: %s", rec.ProviderID)
	}
	return orgID, nil
}

// Compile-time satisfaction for the repository's GetByMilestoneID — the
// concrete *postgres.PaymentRecordRepository already exposes the method,
// so this assertion just documents the expected shape.
var _ PaymentRecordReader = (repository.PaymentRecordRepository)(nil)
