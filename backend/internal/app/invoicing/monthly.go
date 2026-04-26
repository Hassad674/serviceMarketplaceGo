package invoicing

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/invoicing"
)

// IssueMonthlyConsolidatedInput groups the fields the monthly batch /
// CLI / scheduler hand to the service. The period is identified by
// (Year, Month) where Month is 1-12 and refers to the month being
// consolidated — e.g. month=4 in 2026 means the invoice is issued on
// 2026-05-01 for activity that took place in April.
type IssueMonthlyConsolidatedInput struct {
	OrganizationID uuid.UUID
	Year           int
	Month          int // 1-12
}

// CurrentMonthAggregate is the read-only projection used by the live
// "Mes factures → mois en cours" page. No DB writes, no side effects —
// just the totals an organisation will be billed for next month.
type CurrentMonthAggregate struct {
	PeriodStart    time.Time
	PeriodEnd      time.Time
	MilestoneCount int
	TotalFeeCents  int64
	Lines          []CurrentMonthLine
}

// CurrentMonthLine projects one released payment record into the live
// aggregate. The handler turns this into a row in the JSON response.
type CurrentMonthLine struct {
	MilestoneID         uuid.UUID
	PaymentRecordID     uuid.UUID
	ReleasedAt          time.Time
	PlatformFeeCents    int64
	ProposalAmountCents int64
}

// monthlyEventID returns the synthetic stripe_event_id we store on
// monthly consolidation rows. Reusing the existing UNIQUE column lets
// us keep idempotency without a dedicated migration — the value is
// stable per (org, year, month) so a second call with the same
// arguments returns the existing invoice via FindInvoiceByStripeEventID.
func monthlyEventID(orgID uuid.UUID, year, month int) string {
	return fmt.Sprintf("monthly_commission_%s_%04d-%02d", orgID, year, month)
}

// IssueMonthlyConsolidated produces a single customer-facing invoice
// covering every released milestone the org accumulated during the
// requested month. One invoice line per payment_record; the line
// amount is the platform commission (PlatformFeeCents), NOT the gross
// transaction.
//
// Return semantics mirror IssueFromSubscription:
//
//   - (inv, nil) — invoice issued (or replayed and returned as-is).
//   - (nil, nil) — empty period: org had no activity, no row to write.
//   - (nil, err) — caller should log and surface.
func (s *Service) IssueMonthlyConsolidated(ctx context.Context, in IssueMonthlyConsolidatedInput) (*invoicing.Invoice, error) {
	if in.Month < 1 || in.Month > 12 {
		return nil, fmt.Errorf("invoicing: invalid month %d (want 1..12)", in.Month)
	}
	if in.Year < 2000 {
		return nil, fmt.Errorf("invoicing: invalid year %d", in.Year)
	}
	if in.OrganizationID == uuid.Nil {
		return nil, fmt.Errorf("invoicing: organization id required")
	}

	periodStart := time.Date(in.Year, time.Month(in.Month), 1, 0, 0, 0, 0, time.UTC)
	periodEndExclusive := periodStart.AddDate(0, 1, 0)
	syntheticID := monthlyEventID(in.OrganizationID, in.Year, in.Month)

	logger := slog.With(
		"flow", "invoicing.issue_monthly_consolidated",
		"org_id", in.OrganizationID,
		"period", fmt.Sprintf("%04d-%02d", in.Year, in.Month),
		"synthetic_event_id", syntheticID,
	)

	// 1. Idempotency — application-level. We re-use the
	// stripe_event_id UNIQUE column with a synthetic id so a second
	// invocation for the same (org, year, month) returns the row
	// already issued instead of burning another counter value.
	if existing, err := s.invoices.FindInvoiceByStripeEventID(ctx, syntheticID); err == nil && existing != nil {
		logger.Info("invoicing: monthly consolidation already issued, returning existing row",
			"invoice_number", existing.Number)
		return existing, nil
	} else if err != nil && !errors.Is(err, invoicing.ErrNotFound) {
		return nil, fmt.Errorf("invoicing: monthly dedup probe failed: %w", err)
	}

	// 2. Billing profile. Defensive: a missing profile here is a
	// configuration bug at onboarding (Phase 6 gates the wallet
	// behind completeness) and we want to fail loud.
	profile, err := s.profiles.FindByOrganization(ctx, in.OrganizationID)
	if err != nil {
		if errors.Is(err, invoicing.ErrNotFound) {
			return nil, fmt.Errorf("invoicing: monthly consolidation for org without billing profile %s: %w", in.OrganizationID, err)
		}
		return nil, fmt.Errorf("invoicing: load billing profile: %w", err)
	}

	// 3. Released-and-uninvoiced payment records. Empty list →
	// nothing to consolidate; this is normal for orgs that did not
	// transact in the period.
	records, err := s.invoices.ListReleasedPaymentRecordsForOrg(ctx, in.OrganizationID, periodStart, periodEndExclusive)
	if err != nil {
		return nil, fmt.Errorf("invoicing: list released payment records: %w", err)
	}
	if len(records) == 0 {
		logger.Info("invoicing: no released milestones in period, nothing to consolidate")
		return nil, nil
	}

	// 4. Build the recipient snapshot — same helper as the
	// subscription path so the freeze guarantee is identical.
	recipient := buildRecipient(profile)

	// 5. Project payment_records into invoice lines. One line per
	// record; line amount is the commission (PlatformFeeCents).
	items := make([]invoicing.InvoiceItem, 0, len(records))
	for _, rec := range records {
		short := rec.MilestoneID.String()
		if len(short) > 8 {
			short = short[:8]
		}
		items = append(items, invoicing.InvoiceItem{
			Description:     fmt.Sprintf("Commission plateforme — milestone %s", short),
			Quantity:        1,
			UnitPriceCents:  rec.PlatformFeeCents,
			AmountCents:     rec.PlatformFeeCents,
			MilestoneID:     ptrUUID(rec.MilestoneID),
			PaymentRecordID: ptrUUID(rec.ID),
		})
	}

	// 6. Build the draft. Service period is the invoiced month —
	// inclusive end is one nanosecond before the next month starts.
	draft, err := invoicing.NewInvoice(invoicing.NewInvoiceInput{
		RecipientOrganizationID: in.OrganizationID,
		Recipient:               recipient,
		Issuer:                  s.issuer,
		ServicePeriodStart:      periodStart,
		ServicePeriodEnd:        periodEndExclusive.Add(-time.Nanosecond),
		SourceType:              invoicing.SourceMonthlyCommission,
		StripeEventID:           syntheticID,
		StripePaymentIntentID:   "",
		StripeInvoiceID:         "",
		Items:                   items,
	})
	if err != nil {
		return nil, fmt.Errorf("invoicing: build monthly draft: %w", err)
	}

	logger.Info("invoicing: monthly draft built",
		"line_count", len(items),
		"total_fee_cents", draft.AmountInclTaxCents,
	)

	// 7. Hand off to the shared post-NewInvoice pipeline.
	lang := pickLanguage(recipient.Country)
	return s.buildAndPersist(ctx, draft, lang)
}

// GetCurrentMonthAggregate aggregates the released-and-uninvoiced
// payment records of the current month for the given org. Read-only
// — no DB writes, no side effects. The Phase 6 handler turns this
// projection into the /me/invoicing/current-month JSON response.
func (s *Service) GetCurrentMonthAggregate(ctx context.Context, organizationID uuid.UUID) (CurrentMonthAggregate, error) {
	if organizationID == uuid.Nil {
		return CurrentMonthAggregate{}, fmt.Errorf("invoicing: organization id required")
	}

	now := time.Now().UTC()
	periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	periodEnd := periodStart.AddDate(0, 1, 0)

	records, err := s.invoices.ListReleasedPaymentRecordsForOrg(ctx, organizationID, periodStart, periodEnd)
	if err != nil {
		return CurrentMonthAggregate{}, fmt.Errorf("invoicing: list released payment records: %w", err)
	}

	out := CurrentMonthAggregate{
		PeriodStart: periodStart,
		PeriodEnd:   periodEnd,
		Lines:       make([]CurrentMonthLine, 0, len(records)),
	}
	for _, rec := range records {
		out.MilestoneCount++
		out.TotalFeeCents += rec.PlatformFeeCents
		out.Lines = append(out.Lines, CurrentMonthLine{
			MilestoneID:         rec.MilestoneID,
			PaymentRecordID:     rec.ID,
			ReleasedAt:          rec.TransferredAt,
			PlatformFeeCents:    rec.PlatformFeeCents,
			ProposalAmountCents: rec.ProposalAmountCents,
		})
	}
	return out, nil
}

// ptrUUID returns a pointer to the given uuid. Inlined helper kept
// here to avoid sprinkling &id throughout the loop body and to make
// the resulting items easier to read.
func ptrUUID(id uuid.UUID) *uuid.UUID { return &id }
