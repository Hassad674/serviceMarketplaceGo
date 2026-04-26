package invoicing_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"marketplace-backend/internal/domain/invoicing"
)

func TestFormatInvoiceNumber(t *testing.T) {
	assert.Equal(t, "FAC-000001", invoicing.FormatInvoiceNumber(1))
	assert.Equal(t, "FAC-000042", invoicing.FormatInvoiceNumber(42))
	assert.Equal(t, "FAC-999999", invoicing.FormatInvoiceNumber(999999))
	// Past 6 digits: format keeps the prefix and grows naturally.
	assert.Equal(t, "FAC-1000000", invoicing.FormatInvoiceNumber(1_000_000))
	assert.Equal(t, "FAC-12345678", invoicing.FormatInvoiceNumber(12_345_678))
}

func TestFormatCreditNoteNumber(t *testing.T) {
	assert.Equal(t, "AV-000001", invoicing.FormatCreditNoteNumber(1))
	assert.Equal(t, "AV-000123", invoicing.FormatCreditNoteNumber(123))
	assert.Equal(t, "AV-1234567", invoicing.FormatCreditNoteNumber(1_234_567))
}

func TestFormatForScope(t *testing.T) {
	tests := []struct {
		scope   invoicing.CounterScope
		seq     int64
		want    string
		wantErr error
	}{
		{invoicing.ScopeInvoice, 7, "FAC-000007", nil},
		{invoicing.ScopeCreditNote, 8, "AV-000008", nil},
		{invoicing.CounterScope("garbage"), 1, "", invoicing.ErrCounterScopeUnknown},
	}
	for _, tc := range tests {
		t.Run(string(tc.scope), func(t *testing.T) {
			got, err := invoicing.FormatForScope(tc.scope, tc.seq)
			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestCounterScope_IsValid(t *testing.T) {
	assert.True(t, invoicing.ScopeInvoice.IsValid())
	assert.True(t, invoicing.ScopeCreditNote.IsValid())
	assert.False(t, invoicing.CounterScope("nope").IsValid())
	assert.False(t, invoicing.CounterScope("").IsValid())
}
