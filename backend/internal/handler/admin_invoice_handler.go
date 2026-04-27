package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	invoicingapp "marketplace-backend/internal/app/invoicing"
	domain "marketplace-backend/internal/domain/invoicing"
	"marketplace-backend/internal/port/repository"
	res "marketplace-backend/pkg/response"
)

// AdminInvoiceHandler exposes the admin-only "all invoices ever
// emitted" listing + PDF download surface. The router gates the routes
// behind middleware.RequireAdmin — the handler does not double-check
// the role.
type AdminInvoiceHandler struct {
	svc *invoicingapp.Service
}

func NewAdminInvoiceHandler(svc *invoicingapp.Service) *AdminInvoiceHandler {
	return &AdminInvoiceHandler{svc: svc}
}

// ---- DTOs ----

type adminInvoiceRowResponse struct {
	ID                 string  `json:"id"`
	Number             string  `json:"number"`
	IsCreditNote       bool    `json:"is_credit_note"`
	RecipientOrgID     string  `json:"recipient_org_id"`
	RecipientLegalName string  `json:"recipient_legal_name"`
	IssuedAt           string  `json:"issued_at"`
	AmountInclTaxCents int64   `json:"amount_incl_tax_cents"`
	Currency           string  `json:"currency"`
	TaxRegime          string  `json:"tax_regime"`
	Status             string  `json:"status"`
	OriginalInvoiceID  *string `json:"original_invoice_id,omitempty"`
	SourceType         string  `json:"source_type,omitempty"`
}

type adminInvoiceListResponse struct {
	Data       []adminInvoiceRowResponse `json:"data"`
	NextCursor string                    `json:"next_cursor,omitempty"`
	HasMore    bool                      `json:"has_more"`
}

// ---- handlers ----

// List — GET /api/v1/admin/invoices
//
// Filters (all optional):
//
//	recipient_org_id  uuid
//	status            "subscription" | "monthly_commission" | "credit_note"
//	date_from         RFC3339 timestamp
//	date_to           RFC3339 timestamp
//	min_amount_cents  int64
//	max_amount_cents  int64
//	search            free-text against number + recipient legal_name
//	cursor            opaque, from a previous response
//	limit             1..100, default 20
func (h *AdminInvoiceHandler) List(w http.ResponseWriter, r *http.Request) {
	if h.svc == nil {
		res.Error(w, http.StatusServiceUnavailable, "invoicing_disabled", "invoicing feature not configured")
		return
	}

	filters, ok := h.parseFilters(w, r)
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

	rows, nextCursor, err := h.svc.AdminListInvoices(r.Context(), filters, cursor, limit)
	if err != nil {
		res.Error(w, http.StatusInternalServerError, "admin_invoice_list_error", err.Error())
		return
	}

	out := adminInvoiceListResponse{
		Data:       make([]adminInvoiceRowResponse, 0, len(rows)),
		NextCursor: nextCursor,
		HasMore:    nextCursor != "",
	}
	for _, row := range rows {
		out.Data = append(out.Data, toAdminInvoiceRowResponse(row))
	}
	res.JSON(w, http.StatusOK, out)
}

// GetPDF — GET /api/v1/admin/invoices/{id}/pdf?type=invoice|credit_note
//
// 302-redirects to a 5-minute presigned download URL. The `type` query
// parameter is required and disambiguates the source table — we cannot
// blindly look up by id because invoice and credit_note have separate
// id spaces (both use gen_random_uuid()).
func (h *AdminInvoiceHandler) GetPDF(w http.ResponseWriter, r *http.Request) {
	if h.svc == nil {
		res.Error(w, http.StatusServiceUnavailable, "invoicing_disabled", "invoicing feature not configured")
		return
	}

	idRaw := chi.URLParam(r, "id")
	id, err := uuid.Parse(idRaw)
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_invoice_id", "invoice id must be a valid UUID")
		return
	}

	typeParam := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("type")))
	var isCreditNote bool
	switch typeParam {
	case "", "invoice":
		isCreditNote = false
	case "credit_note":
		isCreditNote = true
	default:
		res.Error(w, http.StatusBadRequest, "invalid_type", "type must be 'invoice' or 'credit_note'")
		return
	}

	url, err := h.svc.AdminGetInvoicePDF(r.Context(), id, isCreditNote, presignedURLExpiry)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrNotFound):
			res.Error(w, http.StatusNotFound, "invoice_not_found", "invoice not found")
		default:
			res.Error(w, http.StatusInternalServerError, "invoice_pdf_error", err.Error())
		}
		return
	}
	http.Redirect(w, r, url, http.StatusFound)
}

// ---- helpers ----

func (h *AdminInvoiceHandler) parseFilters(w http.ResponseWriter, r *http.Request) (repository.AdminInvoiceFilters, bool) {
	q := r.URL.Query()
	out := repository.AdminInvoiceFilters{
		Status: strings.TrimSpace(q.Get("status")),
		Search: strings.TrimSpace(q.Get("search")),
	}

	if raw := q.Get("recipient_org_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			res.Error(w, http.StatusBadRequest, "invalid_recipient_org_id", "recipient_org_id must be a valid UUID")
			return out, false
		}
		out.RecipientOrgID = &id
	}
	if raw := q.Get("date_from"); raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			res.Error(w, http.StatusBadRequest, "invalid_date_from", "date_from must be RFC3339")
			return out, false
		}
		out.DateFrom = &t
	}
	if raw := q.Get("date_to"); raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			res.Error(w, http.StatusBadRequest, "invalid_date_to", "date_to must be RFC3339")
			return out, false
		}
		out.DateTo = &t
	}
	if raw := q.Get("min_amount_cents"); raw != "" {
		v, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			res.Error(w, http.StatusBadRequest, "invalid_min_amount", "min_amount_cents must be an integer")
			return out, false
		}
		out.MinAmountCents = &v
	}
	if raw := q.Get("max_amount_cents"); raw != "" {
		v, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			res.Error(w, http.StatusBadRequest, "invalid_max_amount", "max_amount_cents must be an integer")
			return out, false
		}
		out.MaxAmountCents = &v
	}
	return out, true
}

func toAdminInvoiceRowResponse(row *repository.AdminInvoiceRow) adminInvoiceRowResponse {
	out := adminInvoiceRowResponse{
		ID:                 row.ID.String(),
		Number:             row.Number,
		IsCreditNote:       row.IsCreditNote,
		RecipientOrgID:     row.RecipientOrgID.String(),
		RecipientLegalName: row.RecipientLegalName,
		IssuedAt:           row.IssuedAt.UTC().Format(time.RFC3339),
		AmountInclTaxCents: row.AmountInclTaxCents,
		Currency:           row.Currency,
		TaxRegime:          row.TaxRegime,
		Status:             row.Status,
		SourceType:         row.SourceType,
	}
	if row.OriginalInvoiceID != nil {
		s := row.OriginalInvoiceID.String()
		out.OriginalInvoiceID = &s
	}
	return out
}
