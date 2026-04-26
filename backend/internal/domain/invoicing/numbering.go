package invoicing

import "fmt"

// Sequential numbering format: prefix + dash + 6-digit zero-padded
// counter. Continuous to infinity (no annual reset in V1) — wraps
// gracefully past 999999 by widening the printed width while keeping
// the format machine-parseable.
//
// Two scopes share one helper: invoices are FAC-NNNNNN, credit notes
// are AV-NNNNNN. The DB counter table has one row per scope.
const (
	numberPrefixInvoice    = "FAC"
	numberPrefixCreditNote = "AV"
	numberMinWidth         = 6
)

// CounterScope identifies which sequence to draw from. Mirrors the
// `scope` PK column on `invoice_number_counter`.
type CounterScope string

const (
	ScopeInvoice    CounterScope = "invoice"
	ScopeCreditNote CounterScope = "credit_note"
)

// IsValid reports whether the scope is one of the two allowed values.
func (s CounterScope) IsValid() bool {
	return s == ScopeInvoice || s == ScopeCreditNote
}

// FormatInvoiceNumber renders the human-facing invoice identifier.
// Uses a minimum width of 6 digits so early invoices read as
// "FAC-000123"; numbers above 999999 keep the format and grow to
// 7+ digits ("FAC-1000000") without breaking downstream parsers.
func FormatInvoiceNumber(seq int64) string {
	return formatNumber(numberPrefixInvoice, seq)
}

// FormatCreditNoteNumber is the avoir equivalent of FormatInvoiceNumber.
func FormatCreditNoteNumber(seq int64) string {
	return formatNumber(numberPrefixCreditNote, seq)
}

// FormatForScope dispatches to the right formatter based on scope —
// useful for a generic numbering pipeline that doesn't want to switch
// at every callsite.
func FormatForScope(scope CounterScope, seq int64) (string, error) {
	switch scope {
	case ScopeInvoice:
		return FormatInvoiceNumber(seq), nil
	case ScopeCreditNote:
		return FormatCreditNoteNumber(seq), nil
	}
	return "", ErrCounterScopeUnknown
}

func formatNumber(prefix string, seq int64) string {
	return fmt.Sprintf("%s-%0*d", prefix, numberMinWidth, seq)
}
