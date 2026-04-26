// Package invoicing is the application layer for the marketplace's
// outbound billing — the customer-facing invoices we issue when Stripe
// reports a successful subscription payment (and, in a later phase,
// every monthly batch of released milestones).
//
// The service orchestrates: idempotency claim, billing-profile lookup,
// invoice construction (domain), atomic numbering reservation, PDF
// rendering, R2 upload, finalization, persistence, and email delivery.
// All side-effects flow through ports — the service never imports an
// adapter directly so the feature stays test-driven and swappable.
//
// Removable-by-design: deleting this package and its lines in main.go
// disables outbound invoicing without breaking the rest of the
// backend. The Stripe webhook handler short-circuits when its
// invoicing pointer is nil.
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
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
)

// IdempotencyClaimer is the narrow Redis-backed dedup probe. Mirrors
// the same-named interface in handler/stripe_handler.go so tests can
// stub the dependency without pulling the redis SDK. TryClaim returns
// true when this caller is the first to claim eventID, false on a
// replay (already processed). A non-nil error means Redis itself
// failed; the caller decides whether to fall through to the DB-level
// dedup check.
type IdempotencyClaimer interface {
	TryClaim(ctx context.Context, eventID string) (bool, error)
}

// ServiceDeps groups every constructor parameter. The app service
// receives interfaces only — wiring concrete adapters happens in
// cmd/api/main.go.
type ServiceDeps struct {
	Invoices    repository.InvoiceRepository
	Profiles    repository.BillingProfileRepository
	PDF         service.PDFRenderer
	Storage     service.StorageService
	Deliverer   service.InvoiceDeliverer
	Issuer      invoicing.IssuerInfo
	Idempotency IdempotencyClaimer
}

// Service issues invoices for the marketplace. Stateless beyond its
// injected dependencies, safe to share across goroutines.
type Service struct {
	invoices    repository.InvoiceRepository
	profiles    repository.BillingProfileRepository
	pdf         service.PDFRenderer
	storage     service.StorageService
	deliverer   service.InvoiceDeliverer
	issuer      invoicing.IssuerInfo
	idempotency IdempotencyClaimer
}

// NewService constructs the Service. Every dependency is required;
// callers pass nil only when the entire feature is being torn down
// (and in that case main.go wires nothing at all rather than passing
// a partial struct).
func NewService(deps ServiceDeps) *Service {
	return &Service{
		invoices:    deps.Invoices,
		profiles:    deps.Profiles,
		pdf:         deps.PDF,
		storage:     deps.Storage,
		deliverer:   deps.Deliverer,
		issuer:      deps.Issuer,
		idempotency: deps.Idempotency,
	}
}

// IssueFromSubscriptionInput groups the fields the Stripe invoice.paid
// handler extracts from the event payload. The caller pre-formats
// PlanLabel ("Premium Agence — avril 2026" etc.) so the service does
// not own product-naming concerns.
type IssueFromSubscriptionInput struct {
	OrganizationID        uuid.UUID
	StripeEventID         string
	StripeInvoiceID       string
	StripePaymentIntentID string
	AmountCents           int64
	Currency              string
	PeriodStart           time.Time
	PeriodEnd             time.Time
	PlanLabel             string
}


// IssueFromSubscription is the V1 entrypoint: a Stripe invoice.paid
// event for a subscription has just landed; produce a customer-facing
// invoice (FAC-NNNNNN) for the org that paid it.
//
// The pipeline is deliberately verbose at the log layer — every
// observable step emits a structured slog line so the webhook
// timeline can be reconstructed from logs alone when something goes
// sideways at 3am. Each external call inherits the caller's context;
// adapters are responsible for their own timeouts (see CLAUDE.md
// §"Context standards" — every adapter wraps with WithTimeout).
//
// Return semantics:
//
//   - (inv, nil)  — invoice issued (or replayed and returned as-is).
//   - (nil, nil)  — duplicate event silently absorbed.
//   - (nil, err)  — caller should log + return 200 to Stripe (so it
//     won't retry forever); webhook-level retries are NOT free
//     because the whole pipeline runs again on every retry.
func (s *Service) IssueFromSubscription(ctx context.Context, in IssueFromSubscriptionInput) (*invoicing.Invoice, error) {
	logger := slog.With(
		"flow", "invoicing.issue_from_subscription",
		"org_id", in.OrganizationID,
		"stripe_event_id", in.StripeEventID,
		"stripe_invoice_id", in.StripeInvoiceID,
	)

	// 1. Idempotency claim. A duplicate event is a SUCCESS — Stripe
	// retries on transient 5xx, and we do not want to issue a second
	// invoice on the same payment.
	if s.idempotency != nil && in.StripeEventID != "" {
		claimed, cErr := s.idempotency.TryClaim(ctx, in.StripeEventID)
		if cErr != nil {
			// Redis blip — fall through to the DB-level dedup probe.
			logger.Warn("invoicing: idempotency claim error, falling through to db dedup", "error", cErr)
		} else if !claimed {
			logger.Info("invoicing: skipping duplicate stripe event")
			return nil, nil
		}
	}

	// 2. DB-level dedup. Defense in depth: even if Redis lost its
	// claim history (eviction, restart) we never burn a number twice
	// for the same event.
	if existing, err := s.invoices.FindInvoiceByStripeEventID(ctx, in.StripeEventID); err == nil && existing != nil {
		logger.Info("invoicing: stripe event already invoiced, returning existing row",
			"invoice_number", existing.Number)
		return existing, nil
	} else if err != nil && !errors.Is(err, invoicing.ErrNotFound) {
		return nil, fmt.Errorf("invoicing: dedup probe failed: %w", err)
	}

	// 3. Currency invariant. V1 ships EUR only; anything else is a
	// configuration bug we want to fail loud on rather than silently
	// store a mis-currencied row.
	if !strings.EqualFold(in.Currency, "EUR") {
		return nil, fmt.Errorf("%w: got %q", invoicing.ErrInvalidCurrency, in.Currency)
	}

	// 4. Billing profile. The subscribe handler is supposed to gate
	// on completeness (Phase 6 task) so by the time invoice.paid
	// fires we ALWAYS have a complete profile. Defensive logging
	// here helps when Stripe replays an old event from before the
	// gate shipped.
	profile, err := s.profiles.FindByOrganization(ctx, in.OrganizationID)
	if err != nil {
		if errors.Is(err, invoicing.ErrNotFound) {
			return nil, fmt.Errorf("invoicing: subscription invoice for org without billing profile %s — completeness gate should have caught this: %w", in.OrganizationID, err)
		}
		return nil, fmt.Errorf("invoicing: load billing profile: %w", err)
	}

	// 5. Build the recipient snapshot. Frozen verbatim on the row —
	// later edits to billing_profile do NOT mutate already-issued
	// invoices.
	recipient := buildRecipient(profile)

	// 6. Build the draft invoice. Mentions + tax regime resolve
	// deterministically from issuer + recipient countries.
	draft, err := invoicing.NewInvoice(invoicing.NewInvoiceInput{
		RecipientOrganizationID: in.OrganizationID,
		Recipient:               recipient,
		Issuer:                  s.issuer,
		ServicePeriodStart:      in.PeriodStart,
		ServicePeriodEnd:        in.PeriodEnd,
		SourceType:              invoicing.SourceSubscription,
		StripeEventID:           in.StripeEventID,
		StripePaymentIntentID:   in.StripePaymentIntentID,
		StripeInvoiceID:         in.StripeInvoiceID,
		Items: []invoicing.InvoiceItem{
			{
				Description:    in.PlanLabel,
				Quantity:       1,
				UnitPriceCents: in.AmountCents,
				AmountCents:    in.AmountCents,
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("invoicing: build draft invoice: %w", err)
	}

	// 7. Reserve the next invoice number atomically (counter row is
	// SELECT FOR UPDATE inside the adapter).
	seq, err := s.invoices.ReserveNumber(ctx, invoicing.ScopeInvoice)
	if err != nil {
		return nil, fmt.Errorf("invoicing: reserve number: %w", err)
	}
	number := invoicing.FormatInvoiceNumber(seq)

	// 8. Pre-set the number on the draft so the PDF template prints
	// it. Direct assignment is explicitly OK pre-Finalize; after
	// Finalize the row becomes read-only.
	draft.Number = number

	logger = logger.With("invoice_number", number)
	logger.Info("invoicing: number reserved")

	// 9. Render PDF. Language is derived from recipient country —
	// matches the email deliverer convention to keep voice
	// consistent across PDF and notification.
	lang := pickLanguage(recipient.Country)
	pdfBytes, err := s.pdf.RenderInvoice(ctx, draft, lang)
	if err != nil {
		return nil, fmt.Errorf("invoicing: render pdf: %w", err)
	}

	// 10. Upload to R2. Key is deterministic and grouped by org so
	// future archival/scrubbing jobs can scope by prefix.
	pdfKey := fmt.Sprintf("invoices/%s/%s.pdf", in.OrganizationID, number)
	pdfURL, err := s.storage.Upload(
		ctx,
		pdfKey,
		bytes.NewReader(pdfBytes),
		"application/pdf",
		int64(len(pdfBytes)),
	)
	if err != nil {
		return nil, fmt.Errorf("invoicing: upload pdf to r2: %w", err)
	}
	logger.Info("invoicing: pdf uploaded", "pdf_key", pdfKey)

	// 11. Finalize — flips the row to read-only and stamps the
	// finalized_at timestamp. The number was already on the draft
	// for the PDF render but Finalize re-sets it (idempotent).
	if err := draft.Finalize(number, pdfKey); err != nil {
		return nil, fmt.Errorf("invoicing: finalize: %w", err)
	}

	// 12. Persist. The adapter wraps the INSERT in the same tx as
	// the counter increment so a half-issued invoice never lands.
	if err := s.invoices.CreateInvoice(ctx, draft); err != nil {
		// PDF is already in R2 at this point; flag the orphan in
		// logs so a scrubber job can clean up later. Persistence
		// failure must NOT swallow the error — the caller (webhook
		// handler) needs to know.
		logger.Warn("invoicing: persist failed AFTER pdf upload — pdf is orphaned in r2",
			"pdf_key", pdfKey, "error", err)
		return nil, fmt.Errorf("invoicing: persist invoice: %w", err)
	}
	logger.Info("invoicing: invoice persisted")

	// 13. Email delivery. Failures here are LOGGED but DO NOT bubble
	// — re-running the whole pipeline (Stripe retry on 5xx) wastes
	// a counter, re-renders the PDF, and re-uploads to R2. The
	// invoice is correctly issued; a stuck email is recoverable
	// out-of-band (admin can resend from the row).
	if err := s.deliverer.DeliverInvoice(ctx, draft, pdfBytes, pdfURL); err != nil {
		logger.Warn("invoicing: email delivery failed (invoice persisted, retry from admin)",
			"error", err)
	} else {
		logger.Info("invoicing: invoice email delivered")
	}

	return draft, nil
}

// buildRecipient maps a BillingProfile onto the immutable recipient
// snapshot stored on the invoice. Profile edits after issuance never
// touch the snapshot — this is the legal "freeze" the marketplace
// owes the recipient.
func buildRecipient(p *invoicing.BillingProfile) invoicing.RecipientInfo {
	return invoicing.RecipientInfo{
		OrganizationID: p.OrganizationID.String(),
		ProfileType:    string(p.ProfileType),
		LegalName:      p.LegalName,
		TradingName:    p.TradingName,
		LegalForm:      p.LegalForm,
		TaxID:          p.TaxID,
		VATNumber:      p.VATNumber,
		AddressLine1:   p.AddressLine1,
		AddressLine2:   p.AddressLine2,
		PostalCode:     p.PostalCode,
		City:           p.City,
		Country:        p.Country,
		Email:          p.InvoicingEmail,
	}
}

// pickLanguage routes recipients in francophone EU countries to the
// French templates and falls through to English everywhere else.
// Matches the convention in adapter/email.pickLanguage so the PDF
// and the email body speak the same language.
func pickLanguage(countryCode string) string {
	switch strings.ToUpper(strings.TrimSpace(countryCode)) {
	case "FR", "BE", "LU", "MC", "CH":
		return "fr"
	case "":
		return "fr"
	default:
		return "en"
	}
}
