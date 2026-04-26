package service

import (
	"context"

	"marketplace-backend/internal/domain/invoicing"
)

// PDFRenderer turns a finalized invoice or credit note into PDF bytes
// ready for upload to object storage. Implementations choose their
// templating + headless-Chrome strategy; the contract only specifies
// "give me a domain entity and a language, return PDF bytes".
//
// Language is a 2-letter code ("fr", "en") matching the available
// templates. Implementations MUST fall back to a known default when
// they receive an unknown locale rather than erroring — language
// drift should never block invoicing.
type PDFRenderer interface {
	RenderInvoice(ctx context.Context, inv *invoicing.Invoice, language string) ([]byte, error)
	RenderCreditNote(ctx context.Context, cn *invoicing.CreditNote, language string) ([]byte, error)
}
