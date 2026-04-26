package service

import (
	"context"

	"marketplace-backend/internal/domain/invoicing"
)

// InvoiceDeliverer hands a finalized invoice off to the recipient.
// V1 implementation is email-based (wraps the existing
// service.EmailService with invoicing-specific templates). V2 will
// branch on regime/country and route domestic FR-B2B through a
// PDP (Plateforme de Dématérialisation Partenaire) for Factur-X
// e-invoicing once that mandate kicks in.
//
// Implementations MUST be idempotent at the application level — the
// app service guards against duplicate sends via a Redis claim
// before invoking Deliver.
type InvoiceDeliverer interface {
	DeliverInvoice(ctx context.Context, inv *invoicing.Invoice, pdfBytes []byte, downloadURL string) error
	DeliverCreditNote(ctx context.Context, cn *invoicing.CreditNote, pdfBytes []byte, downloadURL string) error
}
