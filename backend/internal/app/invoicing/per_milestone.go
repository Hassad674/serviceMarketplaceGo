package invoicing

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/invoicing"
	"marketplace-backend/internal/domain/payment"
)

// IssueFromMilestoneInput groups the fields the per-milestone emission
// path needs from its callers. The proposal app service builds it from
// the milestone domain entity + the payment record + the provider's
// organization id (resolved from payment_records.provider_organization_id
// when present, falling back to users.organization_id).
//
// Premium gate semantics: the path SKIPS emission when the payment
// record's PlatformFeeAmount is 0 because that already encodes the
// "Premium at payment time" decision — the fee was waived at the moment
// the client funded the milestone (see app.payment.feeForRole). A user
// who becomes Premium AFTER funding still pays the fee for THIS
// milestone (their PlatformFeeAmount is non-zero on the record), which
// matches the brief's "A user who becomes Premium AFTER the milestone
// still gets billed for THIS milestone."
type IssueFromMilestoneInput struct {
	// PaymentRecord carries the platform_fee_amount + currency that the
	// invoice line bills for. The caller MUST pass a record whose
	// MilestoneID matches the milestone being approved.
	PaymentRecord *payment.PaymentRecord
	// ProviderOrganizationID is the org that owes the platform fee — the
	// invoice recipient. Resolved by the caller; we never look it up
	// here to keep this service narrow.
	ProviderOrganizationID uuid.UUID
	// ApprovedAt is the milestone approval timestamp — used as both the
	// service period start AND end (the platform fee is for ONE
	// approved milestone, not a date range). Defaults to time.Now() if
	// zero.
	ApprovedAt time.Time
}

// IssueFromMilestone emits a platform_fee invoice for the given milestone.
//
// Idempotent on TWO layers:
//   - App: FindPlatformFeeByMilestoneID short-circuits the second call.
//   - DB: partial UNIQUE index idx_invoice_milestone_platform_fee_unique
//     catches any race between the lookup and the INSERT.
//
// Returns (nil, nil) when emission is skipped (Premium waiver, missing
// fee, already-issued duplicate) — the caller treats this as a non-error
// outcome. Real I/O failures return (nil, err).
//
// LEGAL: the invoiced amount is platform_fee_amount ONLY. Stripe
// processing fees are charged to the client and NEVER re-billed to the
// provider — they are not legally refacturable. The line amount is
// EXACTLY paymentRecord.PlatformFeeAmount.
func (s *Service) IssueFromMilestone(ctx context.Context, in IssueFromMilestoneInput) (*invoicing.Invoice, error) {
	if err := in.validate(); err != nil {
		return nil, err
	}

	logger := slog.With(
		"flow", "invoicing.issue_from_milestone",
		"milestone_id", in.PaymentRecord.MilestoneID,
		"provider_org_id", in.ProviderOrganizationID,
		"payment_record_id", in.PaymentRecord.ID,
	)

	// Premium gate via the record itself: when the provider was Premium
	// at funding time, the record's PlatformFeeAmount was already set to
	// 0 by app.payment.feeForRole. No invoice is meaningful then.
	if in.PaymentRecord.PlatformFeeAmount <= 0 {
		logger.Info("invoicing: skipping platform_fee — record has zero fee (premium waiver)")
		return nil, nil
	}

	// Idempotence probe — short-circuit duplicate emissions.
	existing, err := s.invoices.FindPlatformFeeByMilestoneID(ctx, in.PaymentRecord.MilestoneID)
	if err != nil && !errors.Is(err, invoicing.ErrNotFound) {
		return nil, fmt.Errorf("invoicing: platform_fee dedup probe: %w", err)
	}
	if existing != nil {
		logger.Info("invoicing: platform_fee already issued, returning existing",
			"invoice_number", existing.Number)
		return existing, nil
	}

	// Billing profile resolution — same shape as the subscription /
	// monthly_commission flows so the recipient snapshot freeze stays
	// uniform.
	profile, err := s.profiles.FindByOrganization(ctx, in.ProviderOrganizationID)
	if err != nil {
		if errors.Is(err, invoicing.ErrNotFound) {
			return nil, fmt.Errorf("invoicing: platform_fee for org without billing profile %s: %w", in.ProviderOrganizationID, err)
		}
		return nil, fmt.Errorf("invoicing: load billing profile: %w", err)
	}
	recipient := buildRecipient(profile)

	approvedAt := in.ApprovedAt
	if approvedAt.IsZero() {
		approvedAt = time.Now().UTC()
	}

	draft, err := s.buildPerMilestoneDraft(in, recipient, approvedAt)
	if err != nil {
		return nil, err
	}

	logger.Info("invoicing: per-milestone draft built",
		"amount_excl_tax_cents", draft.AmountExclTaxCents,
		"currency", draft.Currency,
	)

	lang := pickLanguage(recipient.Country)
	return s.buildAndPersist(ctx, draft, lang)
}

// validate enforces the input invariants up-front so the caller cannot
// accidentally submit an inconsistent record (zero milestone id, missing
// payment record). Centralising these checks keeps the main flow lean
// and respects the < 50 lines per function rule.
func (in *IssueFromMilestoneInput) validate() error {
	if in.PaymentRecord == nil {
		return fmt.Errorf("invoicing: payment record required")
	}
	if in.PaymentRecord.MilestoneID == uuid.Nil {
		return fmt.Errorf("invoicing: payment record milestone id must be non-zero")
	}
	if in.ProviderOrganizationID == uuid.Nil {
		return fmt.Errorf("invoicing: provider organization id required")
	}
	return nil
}

// buildPerMilestoneDraft assembles the domain draft. Extracted from
// IssueFromMilestone to keep that function under 50 lines.
//
// Currency is normalised to upper-case "EUR" to match the NewInvoice
// V1 invariant. The payment record stores it lowercase ("eur") by
// historical convention; the invoice/domain expects "EUR".
func (s *Service) buildPerMilestoneDraft(in IssueFromMilestoneInput, recipient invoicing.RecipientInfo, approvedAt time.Time) (*invoicing.Invoice, error) {
	short := in.PaymentRecord.MilestoneID.String()
	if len(short) > 8 {
		short = short[:8]
	}

	milestoneID := in.PaymentRecord.MilestoneID
	draft, err := invoicing.NewInvoice(invoicing.NewInvoiceInput{
		RecipientOrganizationID: in.ProviderOrganizationID,
		Recipient:               recipient,
		Issuer:                  s.issuer,
		ServicePeriodStart:      approvedAt,
		ServicePeriodEnd:        approvedAt,
		SourceType:              invoicing.SourcePlatformFee,
		MilestoneID:             &milestoneID,
		Items: []invoicing.InvoiceItem{
			{
				Description:     fmt.Sprintf("Commission plateforme — milestone %s", short),
				Quantity:        1,
				UnitPriceCents:  in.PaymentRecord.PlatformFeeAmount,
				AmountCents:     in.PaymentRecord.PlatformFeeAmount,
				MilestoneID:     &milestoneID,
				PaymentRecordID: &in.PaymentRecord.ID,
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("invoicing: build platform_fee draft: %w", err)
	}
	return draft, nil
}
