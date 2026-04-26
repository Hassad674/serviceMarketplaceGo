package pdf_test

import (
	"bytes"
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/adapter/pdf"
	"marketplace-backend/internal/domain/invoicing"
)

// These tests spawn a real headless Chrome via chromedp. CI runners
// without Chrome installed must skip them — gate the run behind
// MARKETPLACE_PDF_TEST=1 (the brief documents this).
func skipIfNotEnabled(t *testing.T) {
	t.Helper()
	if os.Getenv("MARKETPLACE_PDF_TEST") != "1" {
		t.Skip("PDF render tests require Chrome — set MARKETPLACE_PDF_TEST=1 to run")
	}
}

func newTestInvoice(t *testing.T) *invoicing.Invoice {
	t.Helper()
	now := time.Now()
	finalized := now
	return &invoicing.Invoice{
		ID:                      uuid.New(),
		Number:                  "FAC-000123",
		RecipientOrganizationID: uuid.New(),
		RecipientSnapshot: invoicing.RecipientInfo{
			OrganizationID: uuid.NewString(),
			ProfileType:    "business",
			LegalName:      "Recipient SARL",
			AddressLine1:   "12 rue de Rivoli",
			PostalCode:     "75001",
			City:           "Paris",
			Country:        "FR",
			TaxID:          "12345678900012",
			Email:          "client@example.fr",
		},
		IssuerSnapshot: invoicing.IssuerInfo{
			LegalName:    "Marketplace Service SAS",
			LegalForm:    "SAS",
			SIRET:        "87891296300012",
			APECode:      "6202A",
			VATNumber:    "FR26878912963",
			AddressLine1: "1 rue de la Paix",
			PostalCode:   "75002",
			City:         "Paris",
			Country:      "FR",
			Email:        "billing@marketplace.example",
			IBAN:         "FR7612345678901234567890123",
		},
		IssuedAt:           now,
		ServicePeriodStart: now.AddDate(0, -1, 0),
		ServicePeriodEnd:   now,
		Currency:           "EUR",
		AmountExclTaxCents: 12345,
		VATAmountCents:     0,
		AmountInclTaxCents: 12345,
		TaxRegime:          invoicing.RegimeFRFranchiseBase,
		MentionsRendered: []string{
			"TVA non applicable, art. 293 B du CGI",
			"Pas d'escompte pour règlement anticipé",
		},
		SourceType:  invoicing.SourceSubscription,
		Status:      invoicing.StatusIssued,
		FinalizedAt: &finalized,
		Items: []invoicing.InvoiceItem{
			{
				ID:             uuid.New(),
				Description:    "Premium subscription — April 2026",
				Quantity:       1,
				UnitPriceCents: 12345,
				AmountCents:    12345,
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func newTestCreditNote(t *testing.T, parent *invoicing.Invoice) *invoicing.CreditNote {
	t.Helper()
	now := time.Now()
	finalized := now
	return &invoicing.CreditNote{
		ID:                      uuid.New(),
		Number:                  "AV-000007",
		OriginalInvoiceID:       parent.ID,
		RecipientOrganizationID: parent.RecipientOrganizationID,
		RecipientSnapshot:       parent.RecipientSnapshot,
		IssuerSnapshot:          parent.IssuerSnapshot,
		IssuedAt:                now,
		Reason:                  "Refund issued via Stripe webhook",
		Currency:                "EUR",
		AmountExclTaxCents:      5000,
		VATAmountCents:          0,
		AmountInclTaxCents:      5000,
		TaxRegime:               parent.TaxRegime,
		MentionsRendered:        parent.MentionsRendered,
		FinalizedAt:             &finalized,
		CreatedAt:               now,
		UpdatedAt:               now,
	}
}

func TestRenderer_New_ParsesEmbeddedTemplates(t *testing.T) {
	r, err := pdf.New()
	require.NoError(t, err)
	require.NotNil(t, r)
}

func TestRenderer_RenderInvoice_FR(t *testing.T) {
	skipIfNotEnabled(t)
	r, err := pdf.New()
	require.NoError(t, err)

	inv := newTestInvoice(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	out, err := r.RenderInvoice(ctx, inv, "fr")
	require.NoError(t, err)
	require.NotEmpty(t, out)

	assert.True(t, bytes.HasPrefix(out, []byte("%PDF-1.")), "output must be a PDF document")
	assert.Greater(t, len(out), 5*1024, "PDF must be at least 5KB")
}

func TestRenderer_RenderInvoice_EN(t *testing.T) {
	skipIfNotEnabled(t)
	r, err := pdf.New()
	require.NoError(t, err)

	inv := newTestInvoice(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	out, err := r.RenderInvoice(ctx, inv, "en")
	require.NoError(t, err)
	assert.True(t, bytes.HasPrefix(out, []byte("%PDF-1.")))
	assert.Greater(t, len(out), 5*1024)
}

func TestRenderer_RenderInvoice_UnknownLanguage_FallsBackToEN(t *testing.T) {
	skipIfNotEnabled(t)
	r, err := pdf.New()
	require.NoError(t, err)

	inv := newTestInvoice(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	out, err := r.RenderInvoice(ctx, inv, "de")
	require.NoError(t, err, "unknown language must not error — port contract requires graceful fallback")
	assert.True(t, bytes.HasPrefix(out, []byte("%PDF-1.")))
}

func TestRenderer_RenderCreditNote_FR(t *testing.T) {
	skipIfNotEnabled(t)
	r, err := pdf.New()
	require.NoError(t, err)

	inv := newTestInvoice(t)
	cn := newTestCreditNote(t, inv)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	out, err := r.RenderCreditNote(ctx, cn, "fr")
	require.NoError(t, err)
	assert.True(t, bytes.HasPrefix(out, []byte("%PDF-1.")))
	assert.Greater(t, len(out), 5*1024)
}

func TestRenderer_RenderCreditNote_EN(t *testing.T) {
	skipIfNotEnabled(t)
	r, err := pdf.New()
	require.NoError(t, err)

	inv := newTestInvoice(t)
	cn := newTestCreditNote(t, inv)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	out, err := r.RenderCreditNote(ctx, cn, "en")
	require.NoError(t, err)
	assert.True(t, bytes.HasPrefix(out, []byte("%PDF-1.")))
}

func TestRenderer_RenderInvoice_NilInvoice_Errors(t *testing.T) {
	r, err := pdf.New()
	require.NoError(t, err)
	_, err = r.RenderInvoice(context.Background(), nil, "fr")
	assert.Error(t, err)
}
