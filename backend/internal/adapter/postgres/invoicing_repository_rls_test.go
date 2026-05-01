package postgres_test

// Tests for the RLS tenant-context wrap on InvoiceRepository.
//
// BUG-NEW-04 path 3/8 — invoice. The invoice table is RLS-protected
// by migration 125 with the policy
//
//   USING (recipient_organization_id = current_setting('app.current_org_id', true)::uuid)
//
// Migration 125 covers the invoice table; invoice_item and credit_note
// are NOT directly RLS-protected (they inherit security through the
// parent invoice row reachable via FK and via the application-level
// authorization layer).
//
// CreateInvoice runs the parent-row INSERT + N invoice_item INSERTs in
// a single transaction. Under prod NOSUPERUSER NOBYPASSRLS, the parent
// INSERT would be rejected unless app.current_org_id matches
// inv.RecipientOrganizationID. We wrap the whole tx with
// RunInTxWithTenant(inv.RecipientOrganizationID, uuid.Nil, ...) so the
// org context is set BEFORE the parent insert.
//
// Webhook-driven lookups (FindInvoiceByStripeEventID,
// FindInvoiceByStripePaymentIntentID, FindCreditNoteByStripeEventID)
// run as system-actor — there is no caller org. These use uuid.Nil
// for the org context and accept the limitation: under prod RLS with
// no privileged DB role, those reads return ErrNotFound (the policy
// filters every row out). The runtime impact is documented in the
// repository docstring; the production deployment needs to either
// keep the webhook handler on a privileged role OR do a two-step
// lookup. Out of scope for this round.

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/adapter/postgres"
	"marketplace-backend/internal/domain/invoicing"
)

// ---------------------------------------------------------------------------
// Unit tests (no DB)
// ---------------------------------------------------------------------------

func TestInvoiceRepository_WithTxRunner_ReturnsSameRepo(t *testing.T) {
	repo := postgres.NewInvoiceRepository(nil)
	runner := postgres.NewTxRunner(nil)
	got := repo.WithTxRunner(runner)
	assert.Same(t, repo, got)
}

func TestInvoiceRepository_WithTxRunner_NilRunner_NoPanic(t *testing.T) {
	repo := postgres.NewInvoiceRepository(nil)
	got := repo.WithTxRunner(nil)
	assert.NotNil(t, got)
}

func TestInvoiceRepository_CreateInvoice_NilEntry_ReturnsError(t *testing.T) {
	repo := postgres.NewInvoiceRepository(nil)
	err := repo.CreateInvoice(context.Background(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nil invoice")
}

// ---------------------------------------------------------------------------
// Integration tests — gated on MARKETPLACE_TEST_DATABASE_URL
// ---------------------------------------------------------------------------

func newRLSInvoiceFixture(t *testing.T, db *sql.DB) (orgID uuid.UUID, ownerID uuid.UUID) {
	t.Helper()
	ensureRLSTestRole(t, db)
	ownerID = insertTestUser(t, db)
	orgID = insertOrgRaw(t, db, ownerID, "InvoiceOrg-"+uuid.NewString()[:6])
	return orgID, ownerID
}

func buildFinalizedInvoiceRLS(orgID uuid.UUID) *invoicing.Invoice {
	now := time.Now()
	finalized := now
	return &invoicing.Invoice{
		ID:                      uuid.New(),
		Number:                  "FAC-" + uuid.NewString()[:8],
		RecipientOrganizationID: orgID,
		RecipientSnapshot: invoicing.RecipientInfo{
			LegalName: "Test Recipient",
			Country:   "FR",
		},
		IssuerSnapshot: invoicing.IssuerInfo{
			LegalName: "Marketplace Service",
			Country:   "FR",
		},
		IssuedAt:           now,
		ServicePeriodStart: now.AddDate(0, -1, 0),
		ServicePeriodEnd:   now,
		Currency:           "EUR",
		AmountExclTaxCents: 1000,
		VATRate:            0,
		VATAmountCents:     0,
		AmountInclTaxCents: 1000,
		TaxRegime:          invoicing.TaxRegime("fr_franchise_base"),
		MentionsRendered:   []string{"art. 293 B"},
		SourceType:         invoicing.SourceSubscription,
		Status:             invoicing.StatusIssued,
		FinalizedAt:        &finalized,
		CreatedAt:          now,
		UpdatedAt:          now,
		Items: []invoicing.InvoiceItem{
			{
				ID:             uuid.New(),
				Description:    "Test item",
				Quantity:       1,
				UnitPriceCents: 1000,
				AmountCents:    1000,
				CreatedAt:      now,
			},
		},
	}
}

// TestInvoiceRepository_Create_UnderRLS_Succeeds is the regression test
// for path 3/8. The parent invoice + child invoice_item rows must land
// inside a tenant-aware tx so app.current_org_id matches
// inv.RecipientOrganizationID before the parent INSERT fires.
func TestInvoiceRepository_Create_UnderRLS_Succeeds(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	orgID, _ := newRLSInvoiceFixture(t, db)
	repo := postgres.NewInvoiceRepository(db).WithTxRunner(postgres.NewTxRunner(db))

	inv := buildFinalizedInvoiceRLS(orgID)
	require.NoError(t, repo.CreateInvoice(ctx, inv),
		"CreateInvoice with tenant context must persist the parent row + items under RLS")

	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM invoice_item WHERE invoice_id = $1`, inv.ID)
		_, _ = db.Exec(`DELETE FROM invoice WHERE id = $1`, inv.ID)
	})

	// Read back via tenant-aware path.
	got, _, err := repo.ListInvoicesByOrganization(ctx, orgID, "", 10)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(got), 1, "the row we inserted must come back")
}

// TestInvoiceRepository_Create_WithoutWrap_Rejected proves the bug:
// without the txRunner wrap, attempting the same INSERT under the
// non-superuser RLS test role would be rejected by RLS.
func TestInvoiceRepository_Create_WithoutWrap_Rejected(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	orgID, _ := newRLSInvoiceFixture(t, db)

	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	setRLSRole(t, ctx, tx)
	// Do NOT set app.current_org_id — this is the prod failure mode.

	invID := uuid.New()
	_, err = tx.ExecContext(ctx, `
		INSERT INTO invoice (
			id, number, recipient_organization_id, recipient_snapshot,
			issuer_snapshot, service_period_start, service_period_end,
			amount_excl_tax_cents, amount_incl_tax_cents, tax_regime, source_type
		) VALUES ($1, $2, $3, '{}'::jsonb, '{}'::jsonb, now(), now(),
				  1000, 1200, 'fr_franchise_base', 'subscription')`,
		invID, "FAC-"+invID.String()[:8], orgID)
	require.Error(t, err, "INSERT without tenant context MUST be rejected by RLS — locks the regression")
	assert.Contains(t, err.Error(), "row-level security",
		"the rejection reason must be RLS")
}

// TestInvoiceRepository_FindByID_UnderRLS asserts the read path also
// wraps in tenant context. We need the org id at call time; using a
// known invoice's owner org demonstrates the policy fires correctly.
func TestInvoiceRepository_FindByID_UnderRLS(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	orgID, _ := newRLSInvoiceFixture(t, db)
	repo := postgres.NewInvoiceRepository(db).WithTxRunner(postgres.NewTxRunner(db))

	inv := buildFinalizedInvoiceRLS(orgID)
	require.NoError(t, repo.CreateInvoice(ctx, inv))
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM invoice_item WHERE invoice_id = $1`, inv.ID)
		_, _ = db.Exec(`DELETE FROM invoice WHERE id = $1`, inv.ID)
	})

	got, err := repo.FindInvoiceByIDForOrg(ctx, inv.ID, orgID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, inv.ID, got.ID)
}

// TestInvoiceRepository_Legacy_NoTxRunner_StillWorks confirms backwards
// compat: a repo built without WithTxRunner still uses plain
// db.ExecContext and works under superuser test setups.
func TestInvoiceRepository_Legacy_NoTxRunner_StillWorks(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	orgID, _ := newRLSInvoiceFixture(t, db)
	repo := postgres.NewInvoiceRepository(db) // no txRunner — legacy path

	inv := buildFinalizedInvoiceRLS(orgID)
	require.NoError(t, repo.CreateInvoice(ctx, inv),
		"legacy path must keep working for unit tests with only *sql.DB")
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM invoice_item WHERE invoice_id = $1`, inv.ID)
		_, _ = db.Exec(`DELETE FROM invoice WHERE id = $1`, inv.ID)
	})
}
