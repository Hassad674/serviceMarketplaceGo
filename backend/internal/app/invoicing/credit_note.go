package invoicing

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/invoicing"
)

// IssueCreditNoteInput groups the constructor arguments for the credit
// note pipeline. Origin matters:
//
//   - The Stripe charge.refunded webhook fills StripeEventID from the
//     event id; idempotency dedup keys off it.
//   - The admin manual flow (POST /admin/invoices/:id/credit-note) builds
//     a synthetic key like "manual_credit_note_<invoiceID>_<unix>" so
//     replays of that endpoint also dedup correctly.
//
// Reason is free text shown verbatim on the PDF and stored on the row —
// "Stripe refund", "Admin correction: wrong amount", etc.
type IssueCreditNoteInput struct {
	OriginalInvoiceID uuid.UUID
	Reason            string
	AmountCents       int64
	StripeEventID     string
	StripeRefundID    string
	Lang              string
}

// IssueCreditNote produces a customer-facing credit note (AV-NNNNNN)
// for a previously issued invoice. Mirrors IssueFromSubscription's
// pipeline almost step-for-step:
//
//	idempotency claim → DB-level dedup → load + validate original →
//	reserve number → render PDF → upload R2 → finalize → persist →
//	mark original credited if full → email best-effort.
//
// Snapshots (recipient + issuer) are inherited verbatim from the
// original invoice — re-snapshotting at credit time would be a forgery
// surface (legal immutability requirement).
//
// Return semantics match IssueFromSubscription:
//
//   - (cn, nil)  — credit note issued (or replayed and returned as-is).
//   - (nil, nil) — duplicate event silently absorbed.
//   - (nil, err) — caller logs + returns 200 to Stripe.
func (s *Service) IssueCreditNote(ctx context.Context, in IssueCreditNoteInput) (*invoicing.CreditNote, error) {
	logger := slog.With(
		"flow", "invoicing.issue_credit_note",
		"original_invoice_id", in.OriginalInvoiceID,
		"stripe_event_id", in.StripeEventID,
	)

	if existing, dup, err := s.creditNoteDedup(ctx, in.StripeEventID, logger); err != nil {
		return nil, err
	} else if dup {
		return existing, nil
	}

	original, err := s.loadOriginalForCreditNote(ctx, in)
	if err != nil {
		return nil, err
	}

	draft, err := invoicing.NewCreditNote(invoicing.NewCreditNoteInput{
		OriginalInvoice:     original,
		Reason:              in.Reason,
		AmountCreditedCents: in.AmountCents,
		StripeEventID:       in.StripeEventID,
		StripeRefundID:      in.StripeRefundID,
	})
	if err != nil {
		return nil, fmt.Errorf("invoicing: build draft credit note: %w", err)
	}

	lang := strings.TrimSpace(strings.ToLower(in.Lang))
	if lang == "" {
		lang = pickLanguage(original.RecipientSnapshot.Country)
	}

	return s.renderAndPersistCreditNote(ctx, draft, original, lang)
}

// creditNoteDedup runs the two-layer dedup probe (Redis idempotency
// claim + DB-level lookup). Returns (existing, true, nil) on a duplicate
// event so the caller can short-circuit; (nil, false, nil) when the
// caller should proceed with the issuance pipeline; (nil, false, err)
// only when the DB probe itself fails (Redis blip alone falls through).
func (s *Service) creditNoteDedup(ctx context.Context, stripeEventID string, logger *slog.Logger) (*invoicing.CreditNote, bool, error) {
	// Namespace the key so this flow's idempotency does NOT collide
	// with the outer webhook dispatcher's (which claims the bare
	// event id at gateway level for ALL events). Without the prefix
	// the inner claim always fails on webhook-driven calls.
	if s.idempotency != nil && stripeEventID != "" {
		claimed, cErr := s.idempotency.TryClaim(ctx, "invoicing:credit_note:"+stripeEventID)
		if cErr != nil {
			logger.Warn("invoicing: idempotency claim error, falling through to db dedup", "error", cErr)
		} else if !claimed {
			logger.Info("invoicing: skipping duplicate refund event")
			return nil, true, nil
		}
	}
	if existing, err := s.invoices.FindCreditNoteByStripeEventID(ctx, stripeEventID); err == nil && existing != nil {
		logger.Info("invoicing: stripe event already credited, returning existing row",
			"credit_note_number", existing.Number)
		return existing, true, nil
	} else if err != nil && !errors.Is(err, invoicing.ErrNotFound) {
		return nil, false, fmt.Errorf("invoicing: credit note dedup probe failed: %w", err)
	}
	return nil, false, nil
}

// loadOriginalForCreditNote fetches the source invoice and runs the
// invariants that the domain constructor cannot check on its own
// (amount sanity is guarded twice — here for the API error contract,
// and inside NewCreditNote as a defensive backstop).
func (s *Service) loadOriginalForCreditNote(ctx context.Context, in IssueCreditNoteInput) (*invoicing.Invoice, error) {
	if in.AmountCents <= 0 {
		return nil, fmt.Errorf("invoicing: credit note amount must be positive: %w", invoicing.ErrInvalidAmount)
	}
	original, err := s.invoices.FindInvoiceByID(ctx, in.OriginalInvoiceID)
	if err != nil {
		return nil, fmt.Errorf("invoicing: load original invoice: %w", err)
	}
	if original == nil {
		return nil, invoicing.ErrNotFound
	}
	if !original.IsFinalized() {
		return nil, invoicing.ErrAlreadyFinalized
	}
	if in.AmountCents > original.AmountInclTaxCents {
		return nil, fmt.Errorf("invoicing: credit note amount exceeds original: %w", invoicing.ErrInvalidAmount)
	}
	return original, nil
}

// renderAndPersistCreditNote runs the deterministic post-NewCreditNote
// pipeline: reserve number → render PDF → upload R2 → finalize →
// persist → mark original credited (full refund only) → best-effort
// email. Kept under the 50-line budget by lifting tail steps into the
// helpers below.
func (s *Service) renderAndPersistCreditNote(
	ctx context.Context,
	draft *invoicing.CreditNote,
	original *invoicing.Invoice,
	language string,
) (*invoicing.CreditNote, error) {
	logger := slog.With(
		"flow", "invoicing.persist_credit_note",
		"original_invoice_id", original.ID,
		"original_invoice_number", original.Number,
	)

	seq, err := s.invoices.ReserveNumber(ctx, invoicing.ScopeCreditNote)
	if err != nil {
		return nil, fmt.Errorf("invoicing: reserve credit note number: %w", err)
	}
	number := invoicing.FormatCreditNoteNumber(seq)
	draft.Number = number
	logger = logger.With("credit_note_number", number)
	logger.Info("invoicing: credit note number reserved")

	pdfBytes, pdfKey, pdfURL, err := s.renderAndUploadCreditNotePDF(ctx, draft, number, language, logger)
	if err != nil {
		return nil, err
	}
	if err := draft.Finalize(number, pdfKey); err != nil {
		return nil, fmt.Errorf("invoicing: finalize credit note: %w", err)
	}
	if err := s.invoices.CreateCreditNote(ctx, draft); err != nil {
		logger.Warn("invoicing: persist failed AFTER pdf upload — pdf is orphaned in r2",
			"pdf_key", pdfKey, "error", err)
		return nil, fmt.Errorf("invoicing: persist credit note: %w", err)
	}
	logger.Info("invoicing: credit note persisted")

	s.tryMarkOriginalCredited(ctx, draft, original, logger)
	s.deliverCreditNoteEmail(ctx, draft, pdfBytes, pdfURL, logger)

	return draft, nil
}

// renderAndUploadCreditNotePDF renders the PDF and uploads it to R2.
// Returns (bytes, key, public-URL) so the caller can finalize the row,
// log the orphan on persistence failure, and pass the URL to the email
// deliverer.
func (s *Service) renderAndUploadCreditNotePDF(
	ctx context.Context,
	draft *invoicing.CreditNote,
	number, language string,
	logger *slog.Logger,
) ([]byte, string, string, error) {
	pdfBytes, err := s.pdf.RenderCreditNote(ctx, draft, language)
	if err != nil {
		return nil, "", "", fmt.Errorf("invoicing: render credit note pdf: %w", err)
	}
	pdfKey := fmt.Sprintf("invoices/%s/%s.pdf", draft.RecipientOrganizationID, number)
	pdfURL, err := s.storage.Upload(
		ctx,
		pdfKey,
		bytes.NewReader(pdfBytes),
		"application/pdf",
		int64(len(pdfBytes)),
	)
	if err != nil {
		return nil, "", "", fmt.Errorf("invoicing: upload credit note pdf to r2: %w", err)
	}
	logger.Info("invoicing: credit note pdf uploaded", "pdf_key", pdfKey)
	return pdfBytes, pdfKey, pdfURL, nil
}

// tryMarkOriginalCredited flips the original invoice status to 'credited'
// when the credit note covers the full outstanding amount. Partial
// refunds leave the original as 'issued'. Failure is logged — a stuck
// status is recoverable manually and must NOT bubble (the credit note
// itself is correctly issued).
func (s *Service) tryMarkOriginalCredited(
	ctx context.Context,
	cn *invoicing.CreditNote,
	original *invoicing.Invoice,
	logger *slog.Logger,
) {
	if cn.AmountInclTaxCents != original.AmountInclTaxCents {
		logger.Info("invoicing: partial refund, leaving original invoice status unchanged",
			"original_amount", original.AmountInclTaxCents,
			"credit_amount", cn.AmountInclTaxCents)
		return
	}
	if err := s.invoices.MarkInvoiceCredited(ctx, original.ID); err != nil {
		logger.Warn("invoicing: failed to mark original invoice credited (recoverable manually)",
			"error", err)
		return
	}
	logger.Info("invoicing: original invoice marked credited (full refund)")
}

// deliverCreditNoteEmail is the best-effort email step. Same policy as
// the invoice email: failures are logged, never bubbled — the row is
// correctly persisted and admins can resend out-of-band.
func (s *Service) deliverCreditNoteEmail(
	ctx context.Context,
	cn *invoicing.CreditNote,
	pdfBytes []byte,
	pdfURL string,
	logger *slog.Logger,
) {
	if s.deliverer == nil {
		return
	}
	if err := s.deliverer.DeliverCreditNote(ctx, cn, pdfBytes, pdfURL); err != nil {
		logger.Warn("invoicing: credit note email delivery failed (persisted, retry from admin)",
			"error", err)
		return
	}
	logger.Info("invoicing: credit note email delivered")
}

// PresignCreditNotePDF signs a short-lived download URL for a credit
// note PDF. The admin handler uses this right after issuance so the
// 201 response can carry a clickable link without re-querying the row.
// Callers pass the PDFR2Key from the just-persisted CreditNote — there
// is no ownership check here because the only consumer is admin-gated
// at the route level.
func (s *Service) PresignCreditNotePDF(ctx context.Context, pdfR2Key string, expiry time.Duration) (string, error) {
	if pdfR2Key == "" {
		return "", fmt.Errorf("invoicing: pdf key required")
	}
	if expiry <= 0 {
		expiry = 5 * time.Minute
	}
	return s.storage.GetPresignedDownloadURL(ctx, pdfR2Key, expiry)
}

// FindInvoiceByPaymentIntentID is the lookup the refund webhook uses to
// bridge a charge.refunded event back to the invoice we originally
// issued. Pass-through on the repo with a typed return; kept on the
// service so the handler doesn't need to know the repo interface.
func (s *Service) FindInvoiceByPaymentIntentID(ctx context.Context, paymentIntentID string) (*invoicing.Invoice, error) {
	if paymentIntentID == "" {
		return nil, invoicing.ErrNotFound
	}
	return s.invoices.FindInvoiceByStripePaymentIntentID(ctx, paymentIntentID)
}

// SyntheticManualCreditNoteEventID builds the StripeEventID stand-in for
// the admin manual flow. Same row → same id → idempotent if the admin
// double-clicks within a second. Kept on the service so the handler
// doesn't have to know about the format.
func SyntheticManualCreditNoteEventID(invoiceID uuid.UUID, at time.Time) string {
	return fmt.Sprintf("manual_credit_note_%s_%d", invoiceID, at.Unix())
}
