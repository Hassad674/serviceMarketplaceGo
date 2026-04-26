package invoicing_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	invoicingapp "marketplace-backend/internal/app/invoicing"
	"marketplace-backend/internal/domain/invoicing"
)

// makeFinalizedInvoice fabricates a finalized invoice ready to be the
// origin of a credit note. Mirrors the snapshot rules — recipient and
// issuer are set so the credit-note inheritance is observable.
func makeFinalizedInvoice(orgID uuid.UUID) *invoicing.Invoice {
	now := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	finalized := now.Add(-1 * time.Hour)
	return &invoicing.Invoice{
		ID:                      uuid.New(),
		Number:                  "FAC-000100",
		RecipientOrganizationID: orgID,
		RecipientSnapshot: invoicing.RecipientInfo{
			OrganizationID: orgID.String(),
			LegalName:      "Acme Studio SARL",
			TaxID:          "98765432100018",
			AddressLine1:   "10 boulevard Test",
			PostalCode:     "75002",
			City:           "Paris",
			Country:        "FR",
			Email:          "billing@acme.example",
		},
		IssuerSnapshot: invoicing.IssuerInfo{
			LegalName: "Marketplace Test SAS",
			SIRET:     "12345678900012",
			Country:   "FR",
		},
		IssuedAt:           now,
		Currency:           "EUR",
		AmountExclTaxCents: 4900,
		AmountInclTaxCents: 4900,
		TaxRegime:          invoicing.RegimeFRFranchiseBase,
		MentionsRendered:   []string{"TVA non applicable, art. 293 B du CGI"},
		Status:             invoicing.StatusIssued,
		FinalizedAt:        &finalized,
		StripeEventID:      "evt_invoice_001",
		SourceType:         invoicing.SourceSubscription,
	}
}

func defaultCreditNoteInput(originalID uuid.UUID, amount int64) invoicingapp.IssueCreditNoteInput {
	return invoicingapp.IssueCreditNoteInput{
		OriginalInvoiceID: originalID,
		Reason:            "Stripe refund",
		AmountCents:       amount,
		StripeEventID:     "evt_refund_001",
		StripeRefundID:    "re_test_001",
	}
}

// ---------- happy paths ----------

func TestIssueCreditNote_HappyPath_FullRefund_MarksOriginalCredited(t *testing.T) {
	svc, invRepo, _, pdf, storage, deliverer, _ := newSvc(t)
	orgID := uuid.New()
	original := makeFinalizedInvoice(orgID)
	invRepo.findByIDFn = func(_ context.Context, id uuid.UUID) (*invoicing.Invoice, error) {
		if id == original.ID {
			return original, nil
		}
		return nil, invoicing.ErrNotFound
	}

	out, err := svc.IssueCreditNote(context.Background(), defaultCreditNoteInput(original.ID, original.AmountInclTaxCents))

	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, "AV-000001", out.Number)
	assert.Equal(t, original.AmountInclTaxCents, out.AmountInclTaxCents)
	assert.Equal(t, "EUR", out.Currency)
	assert.True(t, out.IsFinalized())
	assert.Equal(t, "Stripe refund", out.Reason)
	assert.Equal(t, original.ID, out.OriginalInvoiceID)
	assert.Equal(t, 1, storage.uploadCalls, "credit note PDF must be uploaded")
	assert.Contains(t, storage.lastUploadKey, "invoices/"+orgID.String()+"/AV-000001.pdf")
	require.Len(t, invRepo.persistedCreditNotes, 1)
	_ = pdf
	_ = deliverer
	require.Len(t, invRepo.markedCreditedIDs, 1, "full refund must mark original credited")
	assert.Equal(t, original.ID, invRepo.markedCreditedIDs[0])
}

func TestIssueCreditNote_HappyPath_PartialRefund_LeavesOriginalAsIssued(t *testing.T) {
	svc, invRepo, _, _, _, _, _ := newSvc(t)
	orgID := uuid.New()
	original := makeFinalizedInvoice(orgID)
	invRepo.findByIDFn = func(_ context.Context, _ uuid.UUID) (*invoicing.Invoice, error) {
		return original, nil
	}

	half := original.AmountInclTaxCents / 2
	out, err := svc.IssueCreditNote(context.Background(), defaultCreditNoteInput(original.ID, half))

	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, half, out.AmountInclTaxCents)
	require.Len(t, invRepo.persistedCreditNotes, 1)
	assert.Empty(t, invRepo.markedCreditedIDs, "partial refund must NOT mark original credited")
}

func TestIssueCreditNote_SnapshotsInheritedFromOriginal(t *testing.T) {
	svc, invRepo, _, _, _, _, _ := newSvc(t)
	orgID := uuid.New()
	original := makeFinalizedInvoice(orgID)
	invRepo.findByIDFn = func(_ context.Context, _ uuid.UUID) (*invoicing.Invoice, error) {
		return original, nil
	}

	out, err := svc.IssueCreditNote(context.Background(), defaultCreditNoteInput(original.ID, original.AmountInclTaxCents))

	require.NoError(t, err)
	require.NotNil(t, out)
	// Recipient snapshot must be a deep copy of the original (we don't
	// rebuild — the legal trail requires the original parties).
	assert.Equal(t, original.RecipientSnapshot, out.RecipientSnapshot)
	assert.Equal(t, original.IssuerSnapshot, out.IssuerSnapshot)
	assert.Equal(t, original.TaxRegime, out.TaxRegime)
	assert.Equal(t, orgID, out.RecipientOrganizationID)
}

// ---------- idempotency ----------

func TestIssueCreditNote_IdempotencyReplay_NoOp(t *testing.T) {
	svc, invRepo, _, _, storage, deliverer, idem := newSvc(t)
	orgID := uuid.New()
	original := makeFinalizedInvoice(orgID)
	idem.tryClaimFn = func(_ context.Context, _ string) (bool, error) { return false, nil }
	invRepo.findByIDFn = func(_ context.Context, _ uuid.UUID) (*invoicing.Invoice, error) {
		return original, nil
	}

	out, err := svc.IssueCreditNote(context.Background(), defaultCreditNoteInput(original.ID, 1000))

	require.NoError(t, err)
	assert.Nil(t, out, "duplicate event returns (nil, nil)")
	assert.Empty(t, invRepo.persistedCreditNotes, "no persistence on duplicate event")
	assert.Empty(t, invRepo.markedCreditedIDs)
	assert.Equal(t, 0, storage.uploadCalls)
	_ = deliverer
}

func TestIssueCreditNote_DBLevelDedup_ReturnsExistingWithoutReissue(t *testing.T) {
	svc, invRepo, _, _, storage, _, _ := newSvc(t)
	orgID := uuid.New()
	now := time.Now()
	finalized := now.Add(-1 * time.Hour)
	existing := &invoicing.CreditNote{
		ID:                      uuid.New(),
		Number:                  "AV-000042",
		OriginalInvoiceID:       uuid.New(),
		RecipientOrganizationID: orgID,
		FinalizedAt:             &finalized,
		StripeEventID:           "evt_refund_001",
	}
	invRepo.findCnByEventIDFn = func(_ context.Context, _ string) (*invoicing.CreditNote, error) {
		return existing, nil
	}

	out, err := svc.IssueCreditNote(context.Background(), defaultCreditNoteInput(uuid.New(), 1000))

	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, "AV-000042", out.Number, "must return pre-existing row, not issue a new one")
	assert.Equal(t, 0, storage.uploadCalls)
	assert.Empty(t, invRepo.persistedCreditNotes)
}

// ---------- error cases ----------

func TestIssueCreditNote_OriginalNotFinalized_Errors(t *testing.T) {
	svc, invRepo, _, _, _, _, _ := newSvc(t)
	orgID := uuid.New()
	original := makeFinalizedInvoice(orgID)
	original.FinalizedAt = nil // not finalized
	original.Status = invoicing.StatusDraft
	invRepo.findByIDFn = func(_ context.Context, _ uuid.UUID) (*invoicing.Invoice, error) {
		return original, nil
	}

	out, err := svc.IssueCreditNote(context.Background(), defaultCreditNoteInput(original.ID, 1000))

	require.Error(t, err)
	assert.Nil(t, out)
	assert.True(t, errors.Is(err, invoicing.ErrAlreadyFinalized) || errors.Is(err, invoicing.ErrNotFound),
		"expected finalize-related error, got: %v", err)
	assert.Empty(t, invRepo.persistedCreditNotes)
}

func TestIssueCreditNote_AmountExceedsOriginal_Errors(t *testing.T) {
	svc, invRepo, _, _, _, _, _ := newSvc(t)
	orgID := uuid.New()
	original := makeFinalizedInvoice(orgID)
	invRepo.findByIDFn = func(_ context.Context, _ uuid.UUID) (*invoicing.Invoice, error) {
		return original, nil
	}

	out, err := svc.IssueCreditNote(context.Background(), defaultCreditNoteInput(original.ID, original.AmountInclTaxCents+1))

	require.Error(t, err)
	assert.Nil(t, out)
	assert.True(t, errors.Is(err, invoicing.ErrInvalidAmount))
	assert.Empty(t, invRepo.persistedCreditNotes)
}

func TestIssueCreditNote_AmountZero_Errors(t *testing.T) {
	svc, invRepo, _, _, _, _, _ := newSvc(t)
	orgID := uuid.New()
	original := makeFinalizedInvoice(orgID)
	invRepo.findByIDFn = func(_ context.Context, _ uuid.UUID) (*invoicing.Invoice, error) {
		return original, nil
	}

	out, err := svc.IssueCreditNote(context.Background(), defaultCreditNoteInput(original.ID, 0))

	require.Error(t, err)
	assert.Nil(t, out)
	assert.True(t, errors.Is(err, invoicing.ErrInvalidAmount))
}

func TestIssueCreditNote_AmountNegative_Errors(t *testing.T) {
	svc, _, _, _, _, _, _ := newSvc(t)

	out, err := svc.IssueCreditNote(context.Background(), defaultCreditNoteInput(uuid.New(), -100))

	require.Error(t, err)
	assert.Nil(t, out)
	assert.True(t, errors.Is(err, invoicing.ErrInvalidAmount))
}

func TestIssueCreditNote_OriginalNotFound_Errors(t *testing.T) {
	svc, invRepo, _, _, _, _, _ := newSvc(t)
	invRepo.findByIDFn = func(_ context.Context, _ uuid.UUID) (*invoicing.Invoice, error) {
		return nil, invoicing.ErrNotFound
	}

	out, err := svc.IssueCreditNote(context.Background(), defaultCreditNoteInput(uuid.New(), 1000))

	require.Error(t, err)
	assert.Nil(t, out)
	assert.True(t, errors.Is(err, invoicing.ErrNotFound))
}

func TestIssueCreditNote_PDFRenderFailure_NoDBWrite(t *testing.T) {
	svc, invRepo, _, pdf, storage, _, _ := newSvc(t)
	orgID := uuid.New()
	original := makeFinalizedInvoice(orgID)
	invRepo.findByIDFn = func(_ context.Context, _ uuid.UUID) (*invoicing.Invoice, error) {
		return original, nil
	}
	pdf.renderCreditNoteFn = func(_ context.Context, _ *invoicing.CreditNote, _ string) ([]byte, error) {
		return nil, fmt.Errorf("chromium: process crashed")
	}

	out, err := svc.IssueCreditNote(context.Background(), defaultCreditNoteInput(original.ID, 1000))

	require.Error(t, err)
	assert.Nil(t, out)
	assert.Equal(t, 0, storage.uploadCalls, "no upload when pdf render fails")
	assert.Empty(t, invRepo.persistedCreditNotes, "no DB row when pdf render fails")
}

func TestIssueCreditNote_StorageUploadFailure_NoDBWrite(t *testing.T) {
	svc, invRepo, _, _, storage, _, _ := newSvc(t)
	orgID := uuid.New()
	original := makeFinalizedInvoice(orgID)
	invRepo.findByIDFn = func(_ context.Context, _ uuid.UUID) (*invoicing.Invoice, error) {
		return original, nil
	}
	storage.uploadFn = func(_ context.Context, _ string, _ io.Reader, _ string, _ int64) (string, error) {
		return "", fmt.Errorf("r2: connection refused")
	}

	out, err := svc.IssueCreditNote(context.Background(), defaultCreditNoteInput(original.ID, 1000))

	require.Error(t, err)
	assert.Nil(t, out)
	assert.Empty(t, invRepo.persistedCreditNotes, "no DB row on upload failure")
}

func TestIssueCreditNote_EmailFailure_DoesNotFailCall(t *testing.T) {
	svc, invRepo, _, _, _, deliverer, _ := newSvc(t)
	orgID := uuid.New()
	original := makeFinalizedInvoice(orgID)
	invRepo.findByIDFn = func(_ context.Context, _ uuid.UUID) (*invoicing.Invoice, error) {
		return original, nil
	}
	deliverer.deliverCreditNoteFn = func(_ context.Context, _ *invoicing.CreditNote, _ []byte, _ string) error {
		return fmt.Errorf("resend: rate limited")
	}

	out, err := svc.IssueCreditNote(context.Background(), defaultCreditNoteInput(original.ID, 1000))

	require.NoError(t, err, "email failure must NOT bubble — credit note is already persisted")
	require.NotNil(t, out)
	assert.True(t, out.IsFinalized())
	require.Len(t, invRepo.persistedCreditNotes, 1)
}

// ---------- format / synthetic id ----------

func TestIssueCreditNote_NumberFormat_AV_NNNNNN(t *testing.T) {
	svc, invRepo, _, _, _, _, _ := newSvc(t)
	orgID := uuid.New()
	original := makeFinalizedInvoice(orgID)
	invRepo.findByIDFn = func(_ context.Context, _ uuid.UUID) (*invoicing.Invoice, error) {
		return original, nil
	}

	out, err := svc.IssueCreditNote(context.Background(), defaultCreditNoteInput(original.ID, 1000))

	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Regexp(t, `^AV-\d{6,}$`, out.Number)
}

func TestSyntheticManualCreditNoteEventID_StableForSameInputs(t *testing.T) {
	id := uuid.New()
	at := time.Date(2026, 4, 26, 10, 0, 0, 0, time.UTC)
	a := invoicingapp.SyntheticManualCreditNoteEventID(id, at)
	b := invoicingapp.SyntheticManualCreditNoteEventID(id, at)
	assert.Equal(t, a, b, "same inputs must yield the same synthetic id")
	assert.Contains(t, a, "manual_credit_note_")
	assert.Contains(t, a, id.String())
}

func TestFindInvoiceByPaymentIntentID_DelegatesToRepo(t *testing.T) {
	svc, invRepo, _, _, _, _, _ := newSvc(t)
	orgID := uuid.New()
	target := makeFinalizedInvoice(orgID)
	invRepo.findByPIIDFn = func(_ context.Context, pi string) (*invoicing.Invoice, error) {
		assert.Equal(t, "pi_test_999", pi)
		return target, nil
	}

	got, err := svc.FindInvoiceByPaymentIntentID(context.Background(), "pi_test_999")

	require.NoError(t, err)
	assert.Equal(t, target.ID, got.ID)
}

func TestFindInvoiceByPaymentIntentID_EmptyID_NotFound(t *testing.T) {
	svc, _, _, _, _, _, _ := newSvc(t)
	got, err := svc.FindInvoiceByPaymentIntentID(context.Background(), "")
	require.Error(t, err)
	assert.Nil(t, got)
	assert.True(t, errors.Is(err, invoicing.ErrNotFound))
}
