package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	receiptapp "marketplace-backend/internal/app/receipt"
	domain "marketplace-backend/internal/domain/receipt"
	"marketplace-backend/internal/handler/middleware"
	res "marketplace-backend/pkg/response"
)

// ReceiptHandler exposes the read endpoints for transaction receipts
// scoped to the caller's organization. Auth is enforced upstream by
// the router; ownership (the org is a party on the receipt) is
// enforced inside the service via the repository SQL filter AND the
// domain.IsParty check (defense in depth).
//
// Receipts are NOT legal invoices — see project_invoicing_model.md.
// The handler echoes that distinction in every response shape: the
// list response is named "receipts" (not "invoices"), and the PDF
// disclaimer is part of the template, not synthesised at the handler
// level.
type ReceiptHandler struct {
	svc      *receiptapp.Service
	auditLog AuditLogger // optional — nil disables audit log emission
}

// AuditLogger is the narrow port the handler uses to emit
// "view_receipt" / "download_receipt_pdf" audit events. Wired in
// cmd/api/main.go using the same audit service the auth + admin
// flows already use. nil disables the events — receipts still work,
// the audit trail simply does not record them.
//
// Defined here (rather than imported from app/audit) so the handler
// stays free of audit-feature compile dependencies — replaceable
// with a no-op for tests.
type AuditLogger interface {
	LogReceiptView(ctx interface{}, userID, receiptID uuid.UUID, ip string)
	LogReceiptPDFDownload(ctx interface{}, userID, receiptID uuid.UUID, ip string)
}

// NewReceiptHandler wires the handler. svc is mandatory; auditLog
// is optional.
func NewReceiptHandler(svc *receiptapp.Service) *ReceiptHandler {
	return &ReceiptHandler{svc: svc}
}

// WithAuditLogger attaches the audit logger. Fluent setter so the
// wiring layer stays single-line.
func (h *ReceiptHandler) WithAuditLogger(a AuditLogger) *ReceiptHandler {
	h.auditLog = a
	return h
}

// ---- DTOs ----

type receiptPartyResponse struct {
	OrganizationID string `json:"organization_id"`
	Name           string `json:"name"`
	SIRET          string `json:"siret"`
	VAT            string `json:"vat"`
	AddressLine1   string `json:"address_line1"`
	AddressLine2   string `json:"address_line2"`
	City           string `json:"city"`
	PostalCode     string `json:"postal_code"`
	Country        string `json:"country"`
}

type receiptResponse struct {
	ID                            string                `json:"id"`
	PaymentRecordID               string                `json:"payment_record_id"`
	ProposalID                    string                `json:"proposal_id,omitempty"`
	MilestoneID                   string                `json:"milestone_id,omitempty"`
	AmountCents                   int64                 `json:"amount_cents"`
	Currency                      string                `json:"currency"`
	CreatedAt                     string                `json:"created_at"`
	Client                        *receiptPartyResponse `json:"client"`
	Provider                      *receiptPartyResponse `json:"provider"`
	Referrer                      *receiptPartyResponse `json:"referrer"`
	ReferrerCommissionAmountCents int64                 `json:"referrer_commission_amount_cents"`
	SnapshotAvailable             bool                  `json:"snapshot_available"`
}

type receiptListResponse struct {
	Data       []receiptResponse `json:"data"`
	NextCursor string            `json:"next_cursor,omitempty"`
}

// ---- handlers ----

// List — GET /api/v1/receipts?cursor=&limit=
func (h *ReceiptHandler) List(w http.ResponseWriter, r *http.Request) {
	if h.svc == nil {
		res.Error(w, http.StatusServiceUnavailable, "receipt_disabled", "receipt feature not configured")
		return
	}
	orgID, ok := h.requireOrg(w, r)
	if !ok {
		return
	}
	cursor := r.URL.Query().Get("cursor")
	limit := 20
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v > 0 && v <= 100 {
			limit = v
		}
	}
	page, err := h.svc.List(r.Context(), orgID, cursor, limit)
	if err != nil {
		res.Error(w, http.StatusInternalServerError, "receipt_list_error", err.Error())
		return
	}
	out := receiptListResponse{
		Data:       make([]receiptResponse, 0, len(page.Receipts)),
		NextCursor: page.NextCursor,
	}
	for _, rec := range page.Receipts {
		out.Data = append(out.Data, toReceiptResponse(rec))
	}
	res.JSON(w, http.StatusOK, out)
}

// Get — GET /api/v1/receipts/{id}
func (h *ReceiptHandler) Get(w http.ResponseWriter, r *http.Request) {
	if h.svc == nil {
		res.Error(w, http.StatusServiceUnavailable, "receipt_disabled", "receipt feature not configured")
		return
	}
	orgID, ok := h.requireOrg(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_receipt_id", "receipt id must be a valid UUID")
		return
	}
	rec, err := h.svc.Get(r.Context(), id, orgID)
	if err != nil {
		h.handleReadError(w, err)
		return
	}
	h.emitView(r, id)
	res.JSON(w, http.StatusOK, toReceiptResponse(rec))
}

// GetPDF — GET /api/v1/receipts/{id}/pdf
func (h *ReceiptHandler) GetPDF(w http.ResponseWriter, r *http.Request) {
	if h.svc == nil {
		res.Error(w, http.StatusServiceUnavailable, "receipt_disabled", "receipt feature not configured")
		return
	}
	orgID, ok := h.requireOrg(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_receipt_id", "receipt id must be a valid UUID")
		return
	}
	// Boundary validation: strict allowlist on the user-controlled
	// `lang` query param so no tainted string ever reaches the
	// renderer or the response. Closes CodeQL #63 (go/xss G705) —
	// even though the response Content-Type is `application/pdf` and
	// the chromedp renderer uses `html/template` (auto-escaping),
	// gosec rightly flags the taint flow. Allowing only the two
	// supported languages kills the flow at the entry point.
	language := normalizeReceiptLang(r.URL.Query().Get("lang"))
	pdf, _, err := h.svc.RenderPDF(r.Context(), id, orgID, language)
	if err != nil {
		if errors.Is(err, receiptapp.ErrPDFRendererUnavailable) {
			res.Error(w, http.StatusServiceUnavailable, "pdf_renderer_disabled", "PDF rendering not configured")
			return
		}
		h.handleReadError(w, err)
		return
	}
	h.emitPDFDownload(r, id)
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("Content-Disposition", "inline; filename=\"receipt-"+id.String()+".pdf\"")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(pdf); err != nil {
		slog.Error("receipt pdf: write response failed", "receipt_id", id, "error", err)
	}
}

// ---- helpers ----

func (h *ReceiptHandler) requireOrg(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	if _, ok := middleware.GetUserID(r.Context()); !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return uuid.Nil, false
	}
	orgID, ok := middleware.GetOrganizationID(r.Context())
	if !ok {
		res.Error(w, http.StatusForbidden, "no_organization", "user is not yet a member of any organization")
		return uuid.Nil, false
	}
	return orgID, true
}

func (h *ReceiptHandler) handleReadError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		res.Error(w, http.StatusNotFound, "receipt_not_found", "receipt not found")
	case errors.Is(err, domain.ErrForbidden):
		res.Error(w, http.StatusForbidden, "forbidden", "receipt does not belong to your organization")
	default:
		slog.Error("receipt read failed", "error", err)
		res.Error(w, http.StatusInternalServerError, "receipt_error", "internal error reading receipt")
	}
}

func (h *ReceiptHandler) emitView(r *http.Request, receiptID uuid.UUID) {
	if h.auditLog == nil {
		return
	}
	uid, _ := middleware.GetUserID(r.Context())
	h.auditLog.LogReceiptView(r.Context(), uid, receiptID, clientIP(r))
}

func (h *ReceiptHandler) emitPDFDownload(r *http.Request, receiptID uuid.UUID) {
	if h.auditLog == nil {
		return
	}
	uid, _ := middleware.GetUserID(r.Context())
	h.auditLog.LogReceiptPDFDownload(r.Context(), uid, receiptID, clientIP(r))
}

// normalizeReceiptLang collapses any user-supplied `lang` query param
// into the strict allowlist {"fr", "en"} the renderer supports.
// Anything else — including empty, mixed-case, surrounding whitespace,
// or hostile HTML/JS payloads — falls back to "fr" (the primary
// market). This is the boundary defense for CodeQL #63 (go/xss): no
// untrusted string is allowed to flow further into the rendering
// pipeline, even though the chromedp renderer uses html/template and
// the response Content-Type is application/pdf.
func normalizeReceiptLang(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "en":
		return "en"
	default:
		return "fr"
	}
}

// (clientIP is defined in role_overrides_handler.go and reused
// across the handler package — receipt audit calls reuse the same
// best-effort proxy-header parsing so the audit trail stays
// uniform.)

func toReceiptResponse(rec *domain.Receipt) receiptResponse {
	out := receiptResponse{
		ID:                            rec.ID.String(),
		PaymentRecordID:               rec.PaymentRecordID.String(),
		AmountCents:                   rec.AmountCents,
		Currency:                      rec.Currency,
		CreatedAt:                     rec.CreatedAt.UTC().Format(time.RFC3339),
		Client:                        partyResponse(rec.Client),
		Provider:                      partyResponse(rec.Provider),
		Referrer:                      partyResponse(rec.Referrer),
		ReferrerCommissionAmountCents: rec.ReferrerCommissionAmountCents,
		SnapshotAvailable:             rec.SnapshotAvailable,
	}
	if rec.ProposalID != uuid.Nil {
		out.ProposalID = rec.ProposalID.String()
	}
	if rec.MilestoneID != uuid.Nil {
		out.MilestoneID = rec.MilestoneID.String()
	}
	return out
}

func partyResponse(p *domain.PartyBilling) *receiptPartyResponse {
	if p == nil {
		return nil
	}
	return &receiptPartyResponse{
		OrganizationID: p.OrganizationID.String(),
		Name:           p.Name,
		SIRET:          p.SIRET,
		VAT:            p.VAT,
		AddressLine1:   p.AddressLine1,
		AddressLine2:   p.AddressLine2,
		City:           p.City,
		PostalCode:     p.PostalCode,
		Country:        p.Country,
	}
}
