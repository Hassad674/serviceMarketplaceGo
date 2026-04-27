package postgres_test

// Integration tests for InvoiceRepository.ListInvoicesAdmin (the
// UNION ALL admin listing across every org). Gated behind
// MARKETPLACE_TEST_DATABASE_URL via testDB() — skipped when unset.

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/adapter/postgres"
	"marketplace-backend/internal/domain/invoicing"
	repo "marketplace-backend/internal/port/repository"
)

// makeFinalizedCreditNote builds a finalized CreditNote ready to hand
// to CreateCreditNote. We use the issuer/recipient skeletons the
// invoice flow uses, so the round-trip via JSONB works identically.
func makeFinalizedCreditNote(t *testing.T, repository *postgres.InvoiceRepository, originalInvoice *invoicing.Invoice) *invoicing.CreditNote {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Second)

	cn, err := invoicing.NewCreditNote(invoicing.NewCreditNoteInput{
		OriginalInvoice:     originalInvoice,
		AmountCreditedCents: originalInvoice.AmountInclTaxCents,
		Reason:              "test refund",
		StripeEventID:       "evt_cn_" + uuid.New().String()[:8],
	})
	require.NoError(t, err)
	_ = now

	seq, err := repository.ReserveNumber(context.Background(), invoicing.ScopeCreditNote)
	require.NoError(t, err)
	number := invoicing.FormatCreditNoteNumber(seq)
	require.NoError(t, cn.Finalize(number, "r2://credit-notes/"+number+".pdf"))
	return cn
}

func TestInvoiceRepository_ListInvoicesAdmin_BasicMix(t *testing.T) {
	db := testDB(t)
	repository := postgres.NewInvoiceRepository(db)

	orgA := invoicingTestOrg(t, db)
	orgB := invoicingTestOrg(t, db)

	// Seed 3 invoices for orgA + 2 for orgB.
	seedInvoices := func(orgID uuid.UUID, count int) []*invoicing.Invoice {
		out := make([]*invoicing.Invoice, 0, count)
		for i := 0; i < count; i++ {
			inv := makeFinalizedInvoice(t, repository, orgID, "evt_admin_"+uuid.New().String()[:8])
			time.Sleep(2 * time.Millisecond)
			require.NoError(t, repository.CreateInvoice(context.Background(), inv))
			out = append(out, inv)
		}
		return out
	}
	invsA := seedInvoices(orgA, 3)
	_ = seedInvoices(orgB, 2)

	// Add 1 credit note for orgA against the first invoice.
	cn := makeFinalizedCreditNote(t, repository, invsA[0])
	require.NoError(t, repository.CreateCreditNote(context.Background(), cn))

	// No filters: all 3+2 invoices + 1 credit note must be returned by
	// the unfiltered listing (paginating to safety with limit=100).
	rows, _, err := repository.ListInvoicesAdmin(context.Background(), repo.AdminInvoiceFilters{}, "", 100)
	require.NoError(t, err)

	var invoiceCount, cnCount int
	for _, r := range rows {
		if r.IsCreditNote {
			cnCount++
			assert.Equal(t, "credit_note", r.Status, "credit notes use synthetic status")
			require.NotNil(t, r.OriginalInvoiceID)
			assert.Equal(t, invsA[0].ID, *r.OriginalInvoiceID)
		} else {
			invoiceCount++
		}
	}
	assert.GreaterOrEqual(t, invoiceCount, 5, "at least 5 invoices visible")
	assert.GreaterOrEqual(t, cnCount, 1, "at least 1 credit note visible")
}

func TestInvoiceRepository_ListInvoicesAdmin_FilterByRecipientOrg(t *testing.T) {
	db := testDB(t)
	repository := postgres.NewInvoiceRepository(db)

	orgA := invoicingTestOrg(t, db)
	orgB := invoicingTestOrg(t, db)

	for i := 0; i < 2; i++ {
		inv := makeFinalizedInvoice(t, repository, orgA, "evt_filt_a_"+uuid.New().String()[:8])
		require.NoError(t, repository.CreateInvoice(context.Background(), inv))
	}
	for i := 0; i < 3; i++ {
		inv := makeFinalizedInvoice(t, repository, orgB, "evt_filt_b_"+uuid.New().String()[:8])
		require.NoError(t, repository.CreateInvoice(context.Background(), inv))
	}

	rows, _, err := repository.ListInvoicesAdmin(context.Background(), repo.AdminInvoiceFilters{
		RecipientOrgID: &orgA,
	}, "", 100)
	require.NoError(t, err)
	assert.Len(t, rows, 2)
	for _, r := range rows {
		assert.Equal(t, orgA, r.RecipientOrgID, "must only see orgA rows")
		assert.False(t, r.IsCreditNote, "no credit notes seeded for orgA")
	}
}

func TestInvoiceRepository_ListInvoicesAdmin_FilterByStatusCreditNote(t *testing.T) {
	db := testDB(t)
	repository := postgres.NewInvoiceRepository(db)

	orgID := invoicingTestOrg(t, db)
	inv := makeFinalizedInvoice(t, repository, orgID, "evt_status_"+uuid.New().String()[:8])
	require.NoError(t, repository.CreateInvoice(context.Background(), inv))
	cn := makeFinalizedCreditNote(t, repository, inv)
	require.NoError(t, repository.CreateCreditNote(context.Background(), cn))

	// status=credit_note → only credit notes survive.
	rows, _, err := repository.ListInvoicesAdmin(context.Background(), repo.AdminInvoiceFilters{
		Status:         "credit_note",
		RecipientOrgID: &orgID,
	}, "", 100)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.True(t, rows[0].IsCreditNote)
	assert.Equal(t, cn.ID, rows[0].ID)

	// status=subscription → only the source-type=subscription invoice
	// survives (the makeFinalizedInvoice helper uses SourceSubscription).
	rows, _, err = repository.ListInvoicesAdmin(context.Background(), repo.AdminInvoiceFilters{
		Status:         "subscription",
		RecipientOrgID: &orgID,
	}, "", 100)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.False(t, rows[0].IsCreditNote)
	assert.Equal(t, "subscription", rows[0].SourceType)
}

func TestInvoiceRepository_ListInvoicesAdmin_FilterByDateRange(t *testing.T) {
	db := testDB(t)
	repository := postgres.NewInvoiceRepository(db)

	orgID := invoicingTestOrg(t, db)

	// Seed three invoices with explicit issued_at values via direct
	// UPDATE — issued_at defaults to now() and we want the query to
	// distinguish them.
	older := makeFinalizedInvoice(t, repository, orgID, "evt_old_"+uuid.New().String()[:8])
	require.NoError(t, repository.CreateInvoice(context.Background(), older))
	_, err := db.Exec(`UPDATE invoice SET issued_at = $1 WHERE id = $2`,
		time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC), older.ID)
	require.NoError(t, err)

	middle := makeFinalizedInvoice(t, repository, orgID, "evt_mid_"+uuid.New().String()[:8])
	require.NoError(t, repository.CreateInvoice(context.Background(), middle))
	_, err = db.Exec(`UPDATE invoice SET issued_at = $1 WHERE id = $2`,
		time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC), middle.ID)
	require.NoError(t, err)

	newer := makeFinalizedInvoice(t, repository, orgID, "evt_new_"+uuid.New().String()[:8])
	require.NoError(t, repository.CreateInvoice(context.Background(), newer))
	_, err = db.Exec(`UPDATE invoice SET issued_at = $1 WHERE id = $2`,
		time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC), newer.ID)
	require.NoError(t, err)

	// Window over April only.
	from := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 4, 30, 23, 59, 59, 0, time.UTC)
	rows, _, err := repository.ListInvoicesAdmin(context.Background(), repo.AdminInvoiceFilters{
		RecipientOrgID: &orgID,
		DateFrom:       &from,
		DateTo:         &to,
	}, "", 100)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, middle.ID, rows[0].ID)
}

func TestInvoiceRepository_ListInvoicesAdmin_FilterBySearch(t *testing.T) {
	db := testDB(t)
	repository := postgres.NewInvoiceRepository(db)

	orgID := invoicingTestOrg(t, db)

	// Seed two invoices with distinct legal names. We patch the
	// recipient_snapshot directly to avoid plumbing a custom-name path
	// through makeFinalizedInvoice.
	inv1 := makeFinalizedInvoice(t, repository, orgID, "evt_s1_"+uuid.New().String()[:8])
	require.NoError(t, repository.CreateInvoice(context.Background(), inv1))
	_, err := db.Exec(`UPDATE invoice SET recipient_snapshot = jsonb_set(recipient_snapshot, '{legal_name}', '"Acme Studio SARL"') WHERE id = $1`, inv1.ID)
	require.NoError(t, err)

	inv2 := makeFinalizedInvoice(t, repository, orgID, "evt_s2_"+uuid.New().String()[:8])
	require.NoError(t, repository.CreateInvoice(context.Background(), inv2))
	_, err = db.Exec(`UPDATE invoice SET recipient_snapshot = jsonb_set(recipient_snapshot, '{legal_name}', '"Beta Workshop SAS"') WHERE id = $1`, inv2.ID)
	require.NoError(t, err)

	// Free-text "acme" → only inv1 matches via legal_name ILIKE.
	rows, _, err := repository.ListInvoicesAdmin(context.Background(), repo.AdminInvoiceFilters{
		RecipientOrgID: &orgID,
		Search:         "acme",
	}, "", 100)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, inv1.ID, rows[0].ID)

	// Free-text by invoice number → matches the same row.
	rows, _, err = repository.ListInvoicesAdmin(context.Background(), repo.AdminInvoiceFilters{
		RecipientOrgID: &orgID,
		Search:         inv2.Number,
	}, "", 100)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, inv2.ID, rows[0].ID)
}

func TestInvoiceRepository_ListInvoicesAdmin_FilterByAmountRange(t *testing.T) {
	db := testDB(t)
	repository := postgres.NewInvoiceRepository(db)

	orgID := invoicingTestOrg(t, db)

	inv := makeFinalizedInvoice(t, repository, orgID, "evt_amt_"+uuid.New().String()[:8])
	require.NoError(t, repository.CreateInvoice(context.Background(), inv))
	// Patch the amount columns directly so we have a known value.
	_, err := db.Exec(`UPDATE invoice SET amount_incl_tax_cents = 4900, amount_excl_tax_cents = 4900 WHERE id = $1`, inv.ID)
	require.NoError(t, err)

	// Window 1000..10000 → match.
	min := int64(1000)
	max := int64(10000)
	rows, _, err := repository.ListInvoicesAdmin(context.Background(), repo.AdminInvoiceFilters{
		RecipientOrgID: &orgID,
		MinAmountCents: &min,
		MaxAmountCents: &max,
	}, "", 100)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, int64(4900), rows[0].AmountInclTaxCents)

	// Window 5000.. → exclude.
	min = 5000
	rows, _, err = repository.ListInvoicesAdmin(context.Background(), repo.AdminInvoiceFilters{
		RecipientOrgID: &orgID,
		MinAmountCents: &min,
	}, "", 100)
	require.NoError(t, err)
	assert.Len(t, rows, 0)
}

func TestInvoiceRepository_ListInvoicesAdmin_CursorPagination(t *testing.T) {
	db := testDB(t)
	repository := postgres.NewInvoiceRepository(db)

	orgID := invoicingTestOrg(t, db)

	// Seed 5 invoices, each with a distinct issued_at to force a stable
	// DESC order.
	const total = 5
	for i := 0; i < total; i++ {
		inv := makeFinalizedInvoice(t, repository, orgID, "evt_pag_"+uuid.New().String()[:8])
		require.NoError(t, repository.CreateInvoice(context.Background(), inv))
		_, err := db.Exec(`UPDATE invoice SET issued_at = $1 WHERE id = $2`,
			time.Date(2026, 5, i+1, 0, 0, 0, 0, time.UTC), inv.ID)
		require.NoError(t, err)
	}

	const pageLimit = 2
	collected := make([]uuid.UUID, 0, total)
	cursor := ""
	pages := 0
	for {
		page, next, err := repository.ListInvoicesAdmin(context.Background(), repo.AdminInvoiceFilters{
			RecipientOrgID: &orgID,
		}, cursor, pageLimit)
		require.NoError(t, err)
		pages++
		require.LessOrEqual(t, len(page), pageLimit)
		for _, row := range page {
			collected = append(collected, row.ID)
		}
		if next == "" {
			break
		}
		cursor = next
		require.Less(t, pages, 10, "pagination must not loop forever")
	}
	assert.Len(t, collected, total)
}

func TestInvoiceRepository_FindCreditNoteByID(t *testing.T) {
	db := testDB(t)
	repository := postgres.NewInvoiceRepository(db)

	orgID := invoicingTestOrg(t, db)
	inv := makeFinalizedInvoice(t, repository, orgID, "evt_find_cn_"+uuid.New().String()[:8])
	require.NoError(t, repository.CreateInvoice(context.Background(), inv))
	cn := makeFinalizedCreditNote(t, repository, inv)
	require.NoError(t, repository.CreateCreditNote(context.Background(), cn))

	got, err := repository.FindCreditNoteByID(context.Background(), cn.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, cn.ID, got.ID)
	assert.Equal(t, cn.Number, got.Number)

	_, err = repository.FindCreditNoteByID(context.Background(), uuid.New())
	assert.ErrorIs(t, err, invoicing.ErrNotFound)
}

// silenceUnused prevents the unused-import linter complaining when the
// package is built without running the integration tests (testDB
// triggers t.Skip).
var _ = sql.Drivers