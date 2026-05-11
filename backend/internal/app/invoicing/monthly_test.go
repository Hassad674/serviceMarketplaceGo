package invoicing_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	invoicingapp "marketplace-backend/internal/app/invoicing"
	"marketplace-backend/internal/domain/invoicing"
	"marketplace-backend/internal/port/repository"
)

// monthlyInput is a small test helper: most cases share the same
// (year, month) so we centralise the boilerplate.
func monthlyInput(orgID uuid.UUID) invoicingapp.IssueMonthlyConsolidatedInput {
	return invoicingapp.IssueMonthlyConsolidatedInput{
		OrganizationID: orgID,
		Year:           2026,
		Month:          4,
	}
}

func threeReleasedRecords(periodStart time.Time) []repository.ReleasedPaymentRecord {
	return []repository.ReleasedPaymentRecord{
		{
			ID:                  uuid.New(),
			MilestoneID:         uuid.New(),
			ProposalID:          uuid.New(),
			ProposalAmountCents: 100_00,
			PlatformFeeCents:    10_00,
			Currency:            "EUR",
			TransferredAt:       periodStart.Add(2 * 24 * time.Hour),
		},
		{
			ID:                  uuid.New(),
			MilestoneID:         uuid.New(),
			ProposalID:          uuid.New(),
			ProposalAmountCents: 200_00,
			PlatformFeeCents:    20_00,
			Currency:            "EUR",
			TransferredAt:       periodStart.Add(7 * 24 * time.Hour),
		},
		{
			ID:                  uuid.New(),
			MilestoneID:         uuid.New(),
			ProposalID:          uuid.New(),
			ProposalAmountCents: 50_00,
			PlatformFeeCents:    5_00,
			Currency:            "EUR",
			TransferredAt:       periodStart.Add(15 * 24 * time.Hour),
		},
	}
}

func TestIssueMonthlyConsolidated_HappyPath_FRDomestic(t *testing.T) {
	svc, invRepo, profileRepo, pdf, storage, deliverer, _ := newSvc(t)
	orgID := uuid.New()
	profileRepo.findByOrgFn = func(_ context.Context, _ uuid.UUID) (*invoicing.BillingProfile, error) {
		return frProfile(orgID), nil
	}
	periodStart := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	records := threeReleasedRecords(periodStart)
	invRepo.listReleasedForOrgFn = func(_ context.Context, gotOrgID uuid.UUID, gotStart, gotEnd time.Time) ([]repository.ReleasedPaymentRecord, error) {
		assert.Equal(t, orgID, gotOrgID)
		assert.True(t, gotStart.Equal(periodStart), "start = first day of consolidated month")
		assert.True(t, gotEnd.Equal(periodStart.AddDate(0, 1, 0)), "end = first day of next month (exclusive)")
		return records, nil
	}

	out, err := svc.IssueMonthlyConsolidated(context.Background(), monthlyInput(orgID))

	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, invoicing.SourceMonthlyCommission, out.SourceType)
	assert.Equal(t, "FAC-000001", out.Number)
	require.Len(t, out.Items, 3, "one line per released payment record")
	// Total = sum of platform fees: 1000 + 2000 + 500 = 3500.
	assert.Equal(t, int64(3500), out.AmountInclTaxCents)
	for i, line := range out.Items {
		assert.Equal(t, records[i].PlatformFeeCents, line.AmountCents,
			"line amount must equal the platform commission, not the gross")
		require.NotNil(t, line.MilestoneID)
		require.NotNil(t, line.PaymentRecordID)
		assert.Equal(t, records[i].MilestoneID, *line.MilestoneID)
		assert.Equal(t, records[i].ID, *line.PaymentRecordID)
	}
	assert.Equal(t, 1, pdf.calls)
	assert.Equal(t, 1, storage.uploadCalls)
	assert.Equal(t, 1, deliverer.calls)
	require.Len(t, invRepo.persistedInvoices, 1)
	// Synthetic stripe_event_id is the idempotency key the second call
	// will look up — it must be deterministic and per-(org, period).
	assert.Contains(t, out.StripeEventID, "monthly_commission_")
	assert.Contains(t, out.StripeEventID, "2026-04")
}

func TestIssueMonthlyConsolidated_NoActivity_ReturnsNilNil(t *testing.T) {
	svc, invRepo, profileRepo, pdf, storage, deliverer, _ := newSvc(t)
	orgID := uuid.New()
	profileRepo.findByOrgFn = func(_ context.Context, _ uuid.UUID) (*invoicing.BillingProfile, error) {
		return frProfile(orgID), nil
	}
	invRepo.listReleasedForOrgFn = func(_ context.Context, _ uuid.UUID, _, _ time.Time) ([]repository.ReleasedPaymentRecord, error) {
		return nil, nil
	}

	out, err := svc.IssueMonthlyConsolidated(context.Background(), monthlyInput(orgID))

	require.NoError(t, err)
	assert.Nil(t, out, "an org with no released milestones in the period gets nil, nil — not an error")
	assert.Empty(t, invRepo.persistedInvoices)
	assert.Equal(t, 0, pdf.calls)
	assert.Equal(t, 0, storage.uploadCalls)
	assert.Equal(t, 0, deliverer.calls)
}

func TestIssueMonthlyConsolidated_Idempotent(t *testing.T) {
	svc, invRepo, profileRepo, pdf, storage, deliverer, _ := newSvc(t)
	orgID := uuid.New()
	now := time.Now()
	finalized := now.Add(-1 * time.Hour)
	existing := &invoicing.Invoice{
		ID:                      uuid.New(),
		Number:                  "FAC-000099",
		RecipientOrganizationID: orgID,
		Status:                  invoicing.StatusIssued,
		FinalizedAt:             &finalized,
		StripeEventID:           "monthly_commission_" + orgID.String() + "_2026-04",
		SourceType:              invoicing.SourceMonthlyCommission,
	}
	invRepo.findByEventIDFn = func(_ context.Context, _ string) (*invoicing.Invoice, error) {
		return existing, nil
	}
	profileLookups := 0
	profileRepo.findByOrgFn = func(_ context.Context, _ uuid.UUID) (*invoicing.BillingProfile, error) {
		profileLookups++
		return frProfile(orgID), nil
	}

	out, err := svc.IssueMonthlyConsolidated(context.Background(), monthlyInput(orgID))

	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, "FAC-000099", out.Number, "second call returns the existing row, no second issuance")
	assert.Equal(t, 0, profileLookups, "no profile lookup on a replay")
	assert.Equal(t, 0, pdf.calls)
	assert.Equal(t, 0, storage.uploadCalls)
	assert.Equal(t, 0, deliverer.calls)
	assert.Empty(t, invRepo.persistedInvoices, "no second persist on replay")
}

func TestIssueMonthlyConsolidated_MissingProfile_Errors(t *testing.T) {
	svc, _, profileRepo, _, _, _, _ := newSvc(t)
	orgID := uuid.New()
	profileRepo.findByOrgFn = func(_ context.Context, _ uuid.UUID) (*invoicing.BillingProfile, error) {
		return nil, invoicing.ErrNotFound
	}

	out, err := svc.IssueMonthlyConsolidated(context.Background(), monthlyInput(orgID))

	require.Error(t, err)
	assert.Nil(t, out)
	assert.True(t, errors.Is(err, invoicing.ErrNotFound), "error chain must wrap invoicing.ErrNotFound; got %v", err)
}

func TestIssueMonthlyConsolidated_InvalidMonth_Rejected(t *testing.T) {
	svc, _, _, _, _, _, _ := newSvc(t)
	orgID := uuid.New()

	for _, m := range []int{0, 13, -1} {
		out, err := svc.IssueMonthlyConsolidated(context.Background(), invoicingapp.IssueMonthlyConsolidatedInput{
			OrganizationID: orgID,
			Year:           2026,
			Month:          m,
		})
		require.Error(t, err, "month=%d must be rejected", m)
		assert.Nil(t, out)
	}
}

// TestIssueMonthlyConsolidated_AllPremiumWaived_ReturnsNilNil ensures
// that when every released record in the period has a zero platform
// fee (the Premium-waiver path), the service returns (nil, nil) — no
// 0 € invoice is issued. The Premium subscription invoice already
// covers the period, so consolidating zero-fee records would produce
// a meaningless artifact and confuse Premium recipients.
func TestIssueMonthlyConsolidated_AllPremiumWaived_ReturnsNilNil(t *testing.T) {
	svc, invRepo, profileRepo, pdf, storage, deliverer, _ := newSvc(t)
	orgID := uuid.New()
	profileRepo.findByOrgFn = func(_ context.Context, _ uuid.UUID) (*invoicing.BillingProfile, error) {
		return frProfile(orgID), nil
	}
	periodStart := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	// Three released records, all Premium-waived (PlatformFeeCents = 0).
	premiumRecords := []repository.ReleasedPaymentRecord{
		{
			ID:                  uuid.New(),
			MilestoneID:         uuid.New(),
			ProposalID:          uuid.New(),
			ProposalAmountCents: 100_00,
			PlatformFeeCents:    0,
			Currency:            "EUR",
			TransferredAt:       periodStart.Add(2 * 24 * time.Hour),
		},
		{
			ID:                  uuid.New(),
			MilestoneID:         uuid.New(),
			ProposalID:          uuid.New(),
			ProposalAmountCents: 200_00,
			PlatformFeeCents:    0,
			Currency:            "EUR",
			TransferredAt:       periodStart.Add(7 * 24 * time.Hour),
		},
	}
	invRepo.listReleasedForOrgFn = func(_ context.Context, _ uuid.UUID, _, _ time.Time) ([]repository.ReleasedPaymentRecord, error) {
		return premiumRecords, nil
	}

	out, err := svc.IssueMonthlyConsolidated(context.Background(), monthlyInput(orgID))

	require.NoError(t, err)
	assert.Nil(t, out, "premium-only month must return (nil, nil) — no 0 € invoice")
	assert.Empty(t, invRepo.persistedInvoices, "no invoice persisted when every record is premium-waived")
	assert.Equal(t, 0, pdf.calls, "no PDF generated for an empty consolidation")
	assert.Equal(t, 0, storage.uploadCalls, "no upload for an empty consolidation")
	assert.Equal(t, 0, deliverer.calls, "no email sent for an empty consolidation")
}

// TestIssueMonthlyConsolidated_MixedPremiumAndPaidRecords_OnlyBillsPaid
// covers an org that paid Premium AFTER some milestones were already
// released at the standard fee. The batch must invoice only the
// non-zero-fee records — the Premium-waived rows are silently skipped.
func TestIssueMonthlyConsolidated_MixedPremiumAndPaidRecords_OnlyBillsPaid(t *testing.T) {
	svc, invRepo, profileRepo, pdf, _, _, _ := newSvc(t)
	orgID := uuid.New()
	profileRepo.findByOrgFn = func(_ context.Context, _ uuid.UUID) (*invoicing.BillingProfile, error) {
		return frProfile(orgID), nil
	}
	periodStart := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	paidRecord := repository.ReleasedPaymentRecord{
		ID:                  uuid.New(),
		MilestoneID:         uuid.New(),
		ProposalID:          uuid.New(),
		ProposalAmountCents: 500_00,
		PlatformFeeCents:    25_00, // 25 € — the user's exact bug report scenario
		Currency:            "EUR",
		TransferredAt:       periodStart.Add(3 * 24 * time.Hour),
	}
	premiumRecord := repository.ReleasedPaymentRecord{
		ID:                  uuid.New(),
		MilestoneID:         uuid.New(),
		ProposalID:          uuid.New(),
		ProposalAmountCents: 300_00,
		PlatformFeeCents:    0, // Premium-waived
		Currency:            "EUR",
		TransferredAt:       periodStart.Add(20 * 24 * time.Hour),
	}
	invRepo.listReleasedForOrgFn = func(_ context.Context, _ uuid.UUID, _, _ time.Time) ([]repository.ReleasedPaymentRecord, error) {
		return []repository.ReleasedPaymentRecord{paidRecord, premiumRecord}, nil
	}

	out, err := svc.IssueMonthlyConsolidated(context.Background(), monthlyInput(orgID))

	require.NoError(t, err)
	require.NotNil(t, out, "mixed period with at least one paid record must produce an invoice")
	require.Len(t, out.Items, 1, "premium-waived line must be skipped")
	assert.Equal(t, paidRecord.PlatformFeeCents, out.Items[0].AmountCents)
	require.NotNil(t, out.Items[0].PaymentRecordID)
	assert.Equal(t, paidRecord.ID, *out.Items[0].PaymentRecordID,
		"the surviving line must be the paid one, not the premium-waived one")
	assert.Equal(t, int64(25_00), out.AmountInclTaxCents, "total = 25 € (the paid commission)")
	assert.Equal(t, 1, pdf.calls)
}

// TestIssueMonthlyConsolidated_SafetyNet_SkipsAlreadyInvoiced covers the
// per-milestone safety-net behaviour: when a milestone already carries
// a platform_fee invoice (the new synchronous emission path), the
// monthly consolidation must SKIP that milestone so it is never billed
// twice. If every milestone is already invoiced, the consolidation must
// be a no-op (no 0 € or duplicate row).
func TestIssueMonthlyConsolidated_SafetyNet_SkipsAlreadyInvoiced(t *testing.T) {
	svc, invRepo, profileRepo, pdf, storage, deliverer, _ := newSvc(t)
	orgID := uuid.New()
	profileRepo.findByOrgFn = func(_ context.Context, _ uuid.UUID) (*invoicing.BillingProfile, error) {
		return frProfile(orgID), nil
	}
	periodStart := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	mid1 := uuid.New()
	mid2 := uuid.New()
	records := []repository.ReleasedPaymentRecord{
		{
			ID: uuid.New(), MilestoneID: mid1, ProposalID: uuid.New(),
			ProposalAmountCents: 100_00, PlatformFeeCents: 5_00,
			Currency: "EUR", TransferredAt: periodStart.Add(24 * time.Hour),
		},
		{
			ID: uuid.New(), MilestoneID: mid2, ProposalID: uuid.New(),
			ProposalAmountCents: 100_00, PlatformFeeCents: 5_00,
			Currency: "EUR", TransferredAt: periodStart.Add(48 * time.Hour),
		},
	}
	invRepo.listReleasedForOrgFn = func(_ context.Context, _ uuid.UUID, _, _ time.Time) ([]repository.ReleasedPaymentRecord, error) {
		return records, nil
	}
	// Both milestones already have a synchronous per-milestone invoice
	// → the safety net must skip them entirely.
	invRepo.findPlatformFeeByMilFn = func(_ context.Context, mid uuid.UUID) (*invoicing.Invoice, error) {
		return &invoicing.Invoice{Number: "FAC-EXISTING", MilestoneID: &mid}, nil
	}

	out, err := svc.IssueMonthlyConsolidated(context.Background(), monthlyInput(orgID))
	require.NoError(t, err)
	assert.Nil(t, out, "consolidation must be a no-op when every milestone is already invoiced")
	assert.Empty(t, invRepo.persistedInvoices)
	assert.Equal(t, 0, pdf.calls)
	assert.Equal(t, 0, storage.uploadCalls)
	assert.Equal(t, 0, deliverer.calls)
}

// TestIssueMonthlyConsolidated_SafetyNet_BillsOnlyUninvoiced covers the
// mixed case: of two released milestones, one is already invoiced
// per-milestone (skipped) and one is not (billed by the safety net).
func TestIssueMonthlyConsolidated_SafetyNet_BillsOnlyUninvoiced(t *testing.T) {
	svc, invRepo, profileRepo, _, _, _, _ := newSvc(t)
	orgID := uuid.New()
	profileRepo.findByOrgFn = func(_ context.Context, _ uuid.UUID) (*invoicing.BillingProfile, error) {
		return frProfile(orgID), nil
	}
	periodStart := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	alreadyInvoicedMID := uuid.New()
	missedMID := uuid.New()
	missed := repository.ReleasedPaymentRecord{
		ID: uuid.New(), MilestoneID: missedMID, ProposalID: uuid.New(),
		ProposalAmountCents: 100_00, PlatformFeeCents: 5_00,
		Currency: "EUR", TransferredAt: periodStart.Add(24 * time.Hour),
	}
	alreadyInvoiced := repository.ReleasedPaymentRecord{
		ID: uuid.New(), MilestoneID: alreadyInvoicedMID, ProposalID: uuid.New(),
		ProposalAmountCents: 100_00, PlatformFeeCents: 5_00,
		Currency: "EUR", TransferredAt: periodStart.Add(48 * time.Hour),
	}
	invRepo.listReleasedForOrgFn = func(_ context.Context, _ uuid.UUID, _, _ time.Time) ([]repository.ReleasedPaymentRecord, error) {
		return []repository.ReleasedPaymentRecord{missed, alreadyInvoiced}, nil
	}
	invRepo.findPlatformFeeByMilFn = func(_ context.Context, mid uuid.UUID) (*invoicing.Invoice, error) {
		if mid == alreadyInvoicedMID {
			return &invoicing.Invoice{Number: "FAC-EXISTING", MilestoneID: &mid}, nil
		}
		return nil, invoicing.ErrNotFound
	}

	out, err := svc.IssueMonthlyConsolidated(context.Background(), monthlyInput(orgID))
	require.NoError(t, err)
	require.NotNil(t, out)
	require.Len(t, out.Items, 1, "only the not-yet-invoiced milestone is billed")
	require.NotNil(t, out.Items[0].MilestoneID)
	assert.Equal(t, missedMID, *out.Items[0].MilestoneID,
		"the surviving line must be the one without a synchronous invoice")
}

func TestGetCurrentMonthAggregate_SumsCorrectly(t *testing.T) {
	svc, invRepo, _, _, _, _, _ := newSvc(t)
	orgID := uuid.New()

	// We cannot pin time.Now in production code, but we can capture the
	// (start, end) the service computes by spying on the repo call.
	var capturedStart, capturedEnd time.Time
	records := []repository.ReleasedPaymentRecord{
		{ID: uuid.New(), MilestoneID: uuid.New(), PlatformFeeCents: 800, ProposalAmountCents: 8000, TransferredAt: time.Now()},
		{ID: uuid.New(), MilestoneID: uuid.New(), PlatformFeeCents: 1200, ProposalAmountCents: 12000, TransferredAt: time.Now()},
	}
	invRepo.listReleasedForOrgFn = func(_ context.Context, _ uuid.UUID, start, end time.Time) ([]repository.ReleasedPaymentRecord, error) {
		capturedStart = start
		capturedEnd = end
		return records, nil
	}

	got, err := svc.GetCurrentMonthAggregate(context.Background(), orgID)
	require.NoError(t, err)
	assert.Equal(t, 2, got.MilestoneCount)
	assert.Equal(t, int64(2000), got.TotalFeeCents)
	require.Len(t, got.Lines, 2)
	assert.Equal(t, int64(800), got.Lines[0].PlatformFeeCents)
	assert.Equal(t, int64(8000), got.Lines[0].ProposalAmountCents)
	// Period bounds: start must be the first day of the current month
	// at 00:00 UTC; end must be the first day of next month.
	now := time.Now().UTC()
	wantStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	assert.True(t, capturedStart.Equal(wantStart), "start = first day of current month UTC, got %v", capturedStart)
	assert.True(t, capturedEnd.Equal(wantStart.AddDate(0, 1, 0)), "end = first day of next month UTC, got %v", capturedEnd)
	assert.True(t, got.PeriodStart.Equal(wantStart))
	assert.True(t, got.PeriodEnd.Equal(wantStart.AddDate(0, 1, 0)))
}

func TestGetCurrentMonthAggregate_EmptyState(t *testing.T) {
	svc, invRepo, _, _, _, _, _ := newSvc(t)
	orgID := uuid.New()
	invRepo.listReleasedForOrgFn = func(_ context.Context, _ uuid.UUID, _, _ time.Time) ([]repository.ReleasedPaymentRecord, error) {
		return nil, nil
	}

	got, err := svc.GetCurrentMonthAggregate(context.Background(), orgID)
	require.NoError(t, err)
	assert.Equal(t, 0, got.MilestoneCount)
	assert.Equal(t, int64(0), got.TotalFeeCents)
	assert.NotNil(t, got.Lines, "lines must be a non-nil empty slice for stable JSON encoding")
	assert.Empty(t, got.Lines)
}

func TestGetCurrentMonthAggregate_NilOrgID_Rejected(t *testing.T) {
	svc, _, _, _, _, _, _ := newSvc(t)
	_, err := svc.GetCurrentMonthAggregate(context.Background(), uuid.Nil)
	require.Error(t, err)
}
