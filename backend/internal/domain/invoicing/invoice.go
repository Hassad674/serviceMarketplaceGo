package invoicing

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

// InvoiceItem is a single line on an invoice. Subscription invoices
// carry exactly one item; monthly commission invoices carry one item
// per released milestone in the period.
type InvoiceItem struct {
	ID              uuid.UUID
	InvoiceID       uuid.UUID
	Description     string
	Quantity        float64
	UnitPriceCents  int64
	AmountCents     int64
	MilestoneID     *uuid.UUID
	PaymentRecordID *uuid.UUID
	CreatedAt       time.Time
}

// Validate checks the line totals reconcile. Called before persistence
// so a bad line never lands in the DB.
func (it InvoiceItem) Validate() error {
	if strings.TrimSpace(it.Description) == "" {
		return ErrInvalidAmount
	}
	if it.Quantity <= 0 {
		return ErrInvalidAmount
	}
	if it.UnitPriceCents < 0 || it.AmountCents < 0 {
		return ErrInvalidAmount
	}
	expected := int64(float64(it.UnitPriceCents) * it.Quantity)
	if expected != it.AmountCents {
		return ErrItemAmountMismatch
	}
	return nil
}

// Invoice is the aggregate root of the package. Once Finalize() has
// been called the row is read-only — corrections happen via a
// CreditNote and a fresh Invoice.
type Invoice struct {
	ID                      uuid.UUID
	Number                  string
	RecipientOrganizationID uuid.UUID
	RecipientSnapshot       RecipientInfo
	IssuerSnapshot          IssuerInfo
	IssuedAt                time.Time
	ServicePeriodStart      time.Time
	ServicePeriodEnd        time.Time
	Currency                string
	AmountExclTaxCents      int64
	VATRate                 float64 // percentage (0 in V1)
	VATAmountCents          int64
	AmountInclTaxCents      int64
	TaxRegime               TaxRegime
	MentionsRendered        []string
	SourceType              SourceType
	StripeEventID           string
	StripePaymentIntentID   string
	StripeInvoiceID         string
	PDFR2Key                string
	Status                  Status
	FinalizedAt             *time.Time
	Items                   []InvoiceItem
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

// NewInvoiceInput groups the constructor arguments. Validated as a
// whole — partial states are rejected up-front rather than caught
// later during persistence.
type NewInvoiceInput struct {
	RecipientOrganizationID uuid.UUID
	Recipient               RecipientInfo
	Issuer                  IssuerInfo
	ServicePeriodStart      time.Time
	ServicePeriodEnd        time.Time
	SourceType              SourceType
	StripeEventID           string
	StripePaymentIntentID   string
	StripeInvoiceID         string
	Items                   []InvoiceItem
}

// NewInvoice builds a draft invoice with the totals computed from its
// items. Number and PDFR2Key stay empty — they're assigned by Finalize
// once the counter has been reserved and the PDF has been uploaded.
//
// Status starts at "draft" and tax_regime is derived deterministically
// from issuer + recipient countries; mentions are pre-rendered so the
// PDF template only has to print them in order.
func NewInvoice(in NewInvoiceInput) (*Invoice, error) {
	if in.RecipientOrganizationID == uuid.Nil {
		return nil, ErrInvalidOrganization
	}
	if !in.SourceType.IsValid() {
		return nil, ErrInvalidSourceType
	}
	if in.ServicePeriodEnd.Before(in.ServicePeriodStart) {
		return nil, ErrInvalidPeriod
	}
	if len(in.Items) == 0 {
		return nil, ErrEmptyItems
	}
	if strings.TrimSpace(in.Recipient.Country) == "" {
		return nil, ErrCountryRequired
	}

	// Sum + per-item validation in a single pass.
	var subtotal int64
	for _, it := range in.Items {
		if err := it.Validate(); err != nil {
			return nil, err
		}
		subtotal += it.AmountCents
	}
	if subtotal < 0 {
		return nil, ErrInvalidAmount
	}

	regime := DetermineRegime(in.Issuer.Country, in.Recipient.Country, in.Recipient.HasValidVAT())
	mentions := ResolveMentions(regime, in.Issuer, in.Recipient)

	now := time.Now()
	inv := &Invoice{
		ID:                      uuid.New(),
		RecipientOrganizationID: in.RecipientOrganizationID,
		RecipientSnapshot:       in.Recipient,
		IssuerSnapshot:          in.Issuer,
		IssuedAt:                now,
		ServicePeriodStart:      in.ServicePeriodStart,
		ServicePeriodEnd:        in.ServicePeriodEnd,
		Currency:                "EUR",
		AmountExclTaxCents:      subtotal,
		VATRate:                 0, // franchise en base — never charges VAT in V1
		VATAmountCents:          0,
		AmountInclTaxCents:      subtotal,
		TaxRegime:               regime,
		MentionsRendered:        mentions,
		SourceType:              in.SourceType,
		StripeEventID:           in.StripeEventID,
		StripePaymentIntentID:   in.StripePaymentIntentID,
		StripeInvoiceID:         in.StripeInvoiceID,
		Status:                  StatusDraft,
		Items:                   append([]InvoiceItem(nil), in.Items...),
		CreatedAt:               now,
		UpdatedAt:               now,
	}
	return inv, nil
}

// Finalize seals the invoice. Sets the assigned number (drawn by the
// caller from the atomic counter), the R2 location of the rendered
// PDF, and flips the status to "issued". After Finalize the row is
// read-only — every other Mutator returns ErrAlreadyFinalized.
func (i *Invoice) Finalize(number, pdfR2Key string) error {
	if i.IsFinalized() {
		return ErrAlreadyFinalized
	}
	if strings.TrimSpace(number) == "" {
		return ErrInvalidNumber
	}
	if strings.TrimSpace(pdfR2Key) == "" {
		return ErrPDFKeyRequired
	}
	now := time.Now()
	i.Number = number
	i.PDFR2Key = pdfR2Key
	i.Status = StatusIssued
	i.FinalizedAt = &now
	i.UpdatedAt = now
	return nil
}

// IsFinalized reports whether the invoice can still be mutated. The
// app layer asks before any update and the postgres adapter rejects
// UPDATEs on rows where finalized_at IS NOT NULL as a backup.
func (i *Invoice) IsFinalized() bool {
	return i.FinalizedAt != nil
}

// MarkCredited records that a credit note has been issued against
// this invoice. The status transitions to "credited" but the rest of
// the invoice stays untouched — the credit note is a separate row,
// the original invoice is never edited beyond its status.
func (i *Invoice) MarkCredited() error {
	if !i.IsFinalized() {
		return ErrAlreadyFinalized // can't credit a draft — finalize it first
	}
	i.Status = StatusCredited
	i.UpdatedAt = time.Now()
	return nil
}
