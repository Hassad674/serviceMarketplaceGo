package invoicing

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

// CreditNote (avoir) is the corrective sibling of Invoice — emitted
// on Stripe refund or admin-driven correction. Carries its own
// sequence (AV-NNNNNN) and a back-reference to the original invoice
// so the audit trail stays explicit.
//
// Like Invoice, a CreditNote is immutable once finalized. Status
// transitions are limited to draft → finalized.
type CreditNote struct {
	ID                      uuid.UUID
	Number                  string
	OriginalInvoiceID       uuid.UUID
	RecipientOrganizationID uuid.UUID
	RecipientSnapshot       RecipientInfo
	IssuerSnapshot          IssuerInfo
	IssuedAt                time.Time
	Reason                  string
	Currency                string
	AmountExclTaxCents      int64
	VATRate                 float64
	VATAmountCents          int64
	AmountInclTaxCents      int64
	TaxRegime               TaxRegime
	MentionsRendered        []string
	StripeEventID           string
	StripeRefundID          string
	PDFR2Key                string
	FinalizedAt             *time.Time
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

// NewCreditNoteInput groups the constructor arguments. The credited
// amount is given in positive cents (we don't represent negatives in
// the domain — the PDF template formats them as "-X €" and the DB
// stores positive values).
type NewCreditNoteInput struct {
	OriginalInvoice         *Invoice
	Reason                  string
	AmountCreditedCents     int64
	StripeEventID           string
	StripeRefundID          string
}

// NewCreditNote builds a draft credit note from an issued invoice.
// The recipient and issuer snapshots are inherited from the original
// invoice — we never re-snapshot at credit time because the legal
// reality is that we are reversing the original transaction, with the
// original parties' identities. Editing those at credit time would be
// a forgery surface.
func NewCreditNote(in NewCreditNoteInput) (*CreditNote, error) {
	if in.OriginalInvoice == nil {
		return nil, ErrCreditNoteOriginalRequired
	}
	if !in.OriginalInvoice.IsFinalized() {
		return nil, ErrAlreadyFinalized
	}
	if in.AmountCreditedCents <= 0 {
		return nil, ErrInvalidAmount
	}
	if in.AmountCreditedCents > in.OriginalInvoice.AmountInclTaxCents {
		return nil, ErrInvalidAmount
	}
	now := time.Now()
	return &CreditNote{
		ID:                      uuid.New(),
		OriginalInvoiceID:       in.OriginalInvoice.ID,
		RecipientOrganizationID: in.OriginalInvoice.RecipientOrganizationID,
		RecipientSnapshot:       in.OriginalInvoice.RecipientSnapshot,
		IssuerSnapshot:          in.OriginalInvoice.IssuerSnapshot,
		IssuedAt:                now,
		Reason:                  in.Reason,
		Currency:                "EUR",
		AmountExclTaxCents:      in.AmountCreditedCents,
		VATRate:                 0,
		VATAmountCents:          0,
		AmountInclTaxCents:      in.AmountCreditedCents,
		TaxRegime:               in.OriginalInvoice.TaxRegime,
		MentionsRendered:        append([]string(nil), in.OriginalInvoice.MentionsRendered...),
		StripeEventID:           in.StripeEventID,
		StripeRefundID:          in.StripeRefundID,
		CreatedAt:               now,
		UpdatedAt:               now,
	}, nil
}

// Finalize assigns the AV-NNNNNN number and PDF location, sealing
// the credit note. Same immutability contract as Invoice.Finalize.
func (c *CreditNote) Finalize(number, pdfR2Key string) error {
	if c.IsFinalized() {
		return ErrAlreadyFinalized
	}
	if strings.TrimSpace(number) == "" {
		return ErrInvalidNumber
	}
	if strings.TrimSpace(pdfR2Key) == "" {
		return ErrPDFKeyRequired
	}
	now := time.Now()
	c.Number = number
	c.PDFR2Key = pdfR2Key
	c.FinalizedAt = &now
	c.UpdatedAt = now
	return nil
}

// IsFinalized reports whether the credit note can still be mutated.
func (c *CreditNote) IsFinalized() bool {
	return c.FinalizedAt != nil
}
