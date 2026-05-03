package handler

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	invoicingapp "marketplace-backend/internal/app/invoicing"
	domain "marketplace-backend/internal/domain/invoicing"
	jsondec "marketplace-backend/pkg/decode"
	res "marketplace-backend/pkg/response"
)

// AdminCreditNoteHandler exposes the manual credit-note correction
// endpoint. Admin-only — the route registers RequireAdmin middleware.
// Construction is decoupled from the read-side InvoiceHandler so the
// admin surface stays removable in isolation.
type AdminCreditNoteHandler struct {
	svc *invoicingapp.Service
}

func NewAdminCreditNoteHandler(svc *invoicingapp.Service) *AdminCreditNoteHandler {
	return &AdminCreditNoteHandler{svc: svc}
}

// ---- DTOs ----

type adminCreditNoteRequest struct {
	Reason      string `json:"reason"`
	AmountCents int64  `json:"amount_cents"`
	Lang        string `json:"lang,omitempty"`
}

type adminCreditNoteResponse struct {
	ID                 string `json:"id"`
	Number             string `json:"number"`
	OriginalInvoiceID  string `json:"original_invoice_id"`
	AmountInclTaxCents int64  `json:"amount_incl_tax_cents"`
	Currency           string `json:"currency"`
	IssuedAt           string `json:"issued_at"`
	PDFURL             string `json:"pdf_url"`
}

// ---- handler ----

// Issue — POST /api/v1/admin/invoices/{id}/credit-note
//
// Admin-driven manual correction. Mirrors the Stripe-driven flow but
// builds a synthetic event id so a double-click within the same second
// still dedupes correctly.
func (h *AdminCreditNoteHandler) Issue(w http.ResponseWriter, r *http.Request) {
	if h.svc == nil {
		res.Error(w, http.StatusServiceUnavailable, "invoicing_disabled", "invoicing feature not configured")
		return
	}

	invoiceID, ok := h.parseInvoiceID(w, r)
	if !ok {
		return
	}

	req, ok := h.decodeAndValidate(w, r)
	if !ok {
		return
	}

	syntheticEventID := invoicingapp.SyntheticManualCreditNoteEventID(invoiceID, time.Now().UTC())

	cn, err := h.svc.IssueCreditNote(r.Context(), invoicingapp.IssueCreditNoteInput{
		OriginalInvoiceID: invoiceID,
		Reason:            req.Reason,
		AmountCents:       req.AmountCents,
		StripeEventID:     syntheticEventID,
		Lang:              req.Lang,
	})
	if err != nil {
		h.respondError(w, err)
		return
	}
	if cn == nil {
		// Idempotency duplicate — should be near-impossible on the manual
		// path (synthetic id includes time.Unix), but handle it gracefully.
		res.Error(w, http.StatusConflict, "credit_note_duplicate", "duplicate credit note request")
		return
	}

	pdfURL := h.presignPDF(r, cn)
	res.JSON(w, http.StatusCreated, adminCreditNoteResponse{
		ID:                 cn.ID.String(),
		Number:             cn.Number,
		OriginalInvoiceID:  cn.OriginalInvoiceID.String(),
		AmountInclTaxCents: cn.AmountInclTaxCents,
		Currency:           cn.Currency,
		IssuedAt:           cn.IssuedAt.UTC().Format(time.RFC3339),
		PDFURL:             pdfURL,
	})
}

// ---- helpers ----

func (h *AdminCreditNoteHandler) parseInvoiceID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	idRaw := chi.URLParam(r, "id")
	id, err := uuid.Parse(idRaw)
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_invoice_id", "invoice id must be a valid UUID")
		return uuid.Nil, false
	}
	return id, true
}

func (h *AdminCreditNoteHandler) decodeAndValidate(w http.ResponseWriter, r *http.Request) (adminCreditNoteRequest, bool) {
	var req adminCreditNoteRequest
	// F.5 B1: bound + reject unknown fields. Credit-note body is small.
	if err := jsondec.DecodeBody(w, r, &req, 16<<10); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_body", "request body must be valid JSON")
		return req, false
	}
	if strings.TrimSpace(req.Reason) == "" {
		res.Error(w, http.StatusBadRequest, "reason_required", "reason is required")
		return req, false
	}
	if req.AmountCents <= 0 {
		res.Error(w, http.StatusBadRequest, "invalid_amount", "amount_cents must be a positive integer")
		return req, false
	}
	return req, true
}

// presignPDF best-effort: a missing presigned URL must not break the
// 201 response — the admin can re-fetch the credit note via the read
// endpoint. Failure is silently swallowed (no PDF URL on the response).
func (h *AdminCreditNoteHandler) presignPDF(r *http.Request, cn *domain.CreditNote) string {
	if cn == nil || cn.PDFR2Key == "" {
		return ""
	}
	url, err := h.svc.PresignCreditNotePDF(r.Context(), cn.PDFR2Key, presignedURLExpiry)
	if err != nil {
		return ""
	}
	return url
}

func (h *AdminCreditNoteHandler) respondError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		res.Error(w, http.StatusNotFound, "invoice_not_found", "invoice not found")
	case errors.Is(err, domain.ErrAlreadyFinalized):
		res.Error(w, http.StatusConflict, "invoice_not_finalized", "invoice cannot be credited in its current state")
	case errors.Is(err, domain.ErrInvalidAmount):
		res.Error(w, http.StatusBadRequest, "invalid_amount", err.Error())
	default:
		res.Error(w, http.StatusInternalServerError, "credit_note_error", err.Error())
	}
}
