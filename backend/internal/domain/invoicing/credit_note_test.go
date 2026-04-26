package invoicing_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/domain/invoicing"
)

func finalizedInvoice(t *testing.T) *invoicing.Invoice {
	t.Helper()
	inv, err := invoicing.NewInvoice(validNewInvoiceInput())
	require.NoError(t, err)
	require.NoError(t, inv.Finalize("FAC-000001", "invoices/x/FAC-000001.pdf"))
	return inv
}

func TestNewCreditNote_HappyPath(t *testing.T) {
	original := finalizedInvoice(t)
	cn, err := invoicing.NewCreditNote(invoicing.NewCreditNoteInput{
		OriginalInvoice:     original,
		Reason:              "client requested refund",
		AmountCreditedCents: 4900,
		StripeEventID:       "evt_refund",
		StripeRefundID:      "re_123",
	})

	require.NoError(t, err)
	assert.Equal(t, original.ID, cn.OriginalInvoiceID)
	assert.Equal(t, original.RecipientOrganizationID, cn.RecipientOrganizationID)
	// Snapshots inherited verbatim — never re-built at credit time.
	assert.Equal(t, original.RecipientSnapshot, cn.RecipientSnapshot)
	assert.Equal(t, original.IssuerSnapshot, cn.IssuerSnapshot)
	assert.Equal(t, original.TaxRegime, cn.TaxRegime)
	assert.Equal(t, int64(4900), cn.AmountInclTaxCents)
	assert.Equal(t, "client requested refund", cn.Reason)
	assert.False(t, cn.IsFinalized())
}

func TestNewCreditNote_RejectsDraftInvoice(t *testing.T) {
	original, _ := invoicing.NewInvoice(validNewInvoiceInput())
	_, err := invoicing.NewCreditNote(invoicing.NewCreditNoteInput{
		OriginalInvoice:     original, // not finalized
		AmountCreditedCents: 1000,
	})
	assert.ErrorIs(t, err, invoicing.ErrAlreadyFinalized)
}

func TestNewCreditNote_RejectsMissingOriginal(t *testing.T) {
	_, err := invoicing.NewCreditNote(invoicing.NewCreditNoteInput{
		AmountCreditedCents: 1000,
	})
	assert.ErrorIs(t, err, invoicing.ErrCreditNoteOriginalRequired)
}

func TestNewCreditNote_AmountValidations(t *testing.T) {
	original := finalizedInvoice(t)

	tests := []struct {
		name    string
		amount  int64
		wantErr error
	}{
		{"zero", 0, invoicing.ErrInvalidAmount},
		{"negative", -100, invoicing.ErrInvalidAmount},
		{"exceeds original", original.AmountInclTaxCents + 1, invoicing.ErrInvalidAmount},
		{"equal to original", original.AmountInclTaxCents, nil},
		{"partial below original", original.AmountInclTaxCents - 1, nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := invoicing.NewCreditNote(invoicing.NewCreditNoteInput{
				OriginalInvoice:     original,
				AmountCreditedCents: tc.amount,
			})
			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestCreditNote_Finalize(t *testing.T) {
	original := finalizedInvoice(t)
	cn, _ := invoicing.NewCreditNote(invoicing.NewCreditNoteInput{
		OriginalInvoice:     original,
		AmountCreditedCents: 4900,
	})

	require.NoError(t, cn.Finalize("AV-000001", "credit_notes/x/AV-000001.pdf"))
	assert.Equal(t, "AV-000001", cn.Number)
	assert.True(t, cn.IsFinalized())

	// Re-finalize forbidden.
	err := cn.Finalize("AV-000002", "key")
	assert.ErrorIs(t, err, invoicing.ErrAlreadyFinalized)
}

func TestCreditNote_Finalize_RequiresInputs(t *testing.T) {
	original := finalizedInvoice(t)
	cn, _ := invoicing.NewCreditNote(invoicing.NewCreditNoteInput{
		OriginalInvoice:     original,
		AmountCreditedCents: 4900,
	})
	assert.ErrorIs(t, cn.Finalize("", "key"), invoicing.ErrInvalidNumber)

	cn2, _ := invoicing.NewCreditNote(invoicing.NewCreditNoteInput{
		OriginalInvoice:     original,
		AmountCreditedCents: 4900,
	})
	assert.ErrorIs(t, cn2.Finalize("AV-000001", ""), invoicing.ErrPDFKeyRequired)
}
