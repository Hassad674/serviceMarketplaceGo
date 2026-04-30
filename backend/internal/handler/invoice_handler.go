package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	invoicingapp "marketplace-backend/internal/app/invoicing"
	domain "marketplace-backend/internal/domain/invoicing"
	"marketplace-backend/internal/handler/middleware"
	res "marketplace-backend/pkg/response"
)

// InvoiceHandler exposes the read endpoints for invoices issued to the
// caller's organization, plus the live "current month aggregate" page.
// Auth is enforced by the router; ownership is enforced inside the
// service (the repository always filters by recipient_organization_id).
type InvoiceHandler struct {
	svc *invoicingapp.Service
}

func NewInvoiceHandler(svc *invoicingapp.Service) *InvoiceHandler {
	return &InvoiceHandler{svc: svc}
}

// presignedURLExpiry is the TTL of every PDF download link the handler
// hands out. 5 minutes is short enough that a copy-pasted URL goes
// stale before it can be shared, and long enough for the browser's own
// re-fetch on a slow connection.
const presignedURLExpiry = 5 * time.Minute

// ---- DTOs ----

type invoiceListItemResponse struct {
	ID                 string `json:"id"`
	Number             string `json:"number"`
	IssuedAt           string `json:"issued_at"`
	SourceType         string `json:"source_type"`
	AmountInclTaxCents int64  `json:"amount_incl_tax_cents"`
	Currency           string `json:"currency"`
	PDFURL             string `json:"pdf_url"`
}

type invoiceListResponse struct {
	Data       []invoiceListItemResponse `json:"data"`
	NextCursor string                    `json:"next_cursor,omitempty"`
}

type currentMonthLineResponse struct {
	MilestoneID         string `json:"milestone_id"`
	PaymentRecordID     string `json:"payment_record_id"`
	ReleasedAt          string `json:"released_at"`
	PlatformFeeCents    int64  `json:"platform_fee_cents"`
	ProposalAmountCents int64  `json:"proposal_amount_cents"`
}

type currentMonthResponse struct {
	PeriodStart    string                     `json:"period_start"`
	PeriodEnd      string                     `json:"period_end"`
	MilestoneCount int                        `json:"milestone_count"`
	TotalFeeCents  int64                      `json:"total_fee_cents"`
	Lines          []currentMonthLineResponse `json:"lines"`
}

// ---- handlers ----

// List — GET /api/v1/me/invoices?cursor=&limit=
func (h *InvoiceHandler) List(w http.ResponseWriter, r *http.Request) {
	if h.svc == nil {
		res.Error(w, http.StatusServiceUnavailable, "invoicing_disabled", "invoicing feature not configured")
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
	result, err := h.svc.ListMyInvoices(r.Context(), orgID, cursor, limit)
	if err != nil {
		res.Error(w, http.StatusInternalServerError, "invoice_list_error", err.Error())
		return
	}
	out := invoiceListResponse{
		Data:       make([]invoiceListItemResponse, 0, len(result.Items)),
		NextCursor: result.NextCursor,
	}
	for _, it := range result.Items {
		// We never expose PDF download URLs in the list — the user
		// fetches them through the dedicated /pdf endpoint that
		// re-checks ownership and signs a fresh URL. The PDFURL field
		// is left empty here on purpose so a stale list cache cannot
		// leak a long-lived URL.
		out.Data = append(out.Data, invoiceListItemResponse{
			ID:                 it.ID.String(),
			Number:             it.Number,
			IssuedAt:           it.IssuedAt.UTC().Format(time.RFC3339),
			SourceType:         string(it.SourceType),
			AmountInclTaxCents: it.AmountInclTaxCents,
			Currency:           it.Currency,
			PDFURL:             "",
		})
	}
	res.JSON(w, http.StatusOK, out)
}

// GetPDF — GET /api/v1/me/invoices/{id}/pdf
// Verifies ownership then 302-redirects to a short-lived presigned URL.
func (h *InvoiceHandler) GetPDF(w http.ResponseWriter, r *http.Request) {
	if h.svc == nil {
		res.Error(w, http.StatusServiceUnavailable, "invoicing_disabled", "invoicing feature not configured")
		return
	}
	orgID, ok := h.requireOrg(w, r)
	if !ok {
		return
	}
	idRaw := chi.URLParam(r, "id")
	invoiceID, err := uuid.Parse(idRaw)
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_invoice_id", "invoice id must be a valid UUID")
		return
	}
	rawURL, err := h.svc.GetInvoicePDFURL(r.Context(), orgID, invoiceID, presignedURLExpiry)
	if err != nil {
		switch {
		case errors.Is(err, invoicingapp.ErrCrossOrgInvoiceAccess):
			res.Error(w, http.StatusForbidden, "forbidden", "invoice does not belong to your organization")
		case errors.Is(err, domain.ErrNotFound):
			res.Error(w, http.StatusNotFound, "invoice_not_found", "invoice not found")
		default:
			res.Error(w, http.StatusInternalServerError, "invoice_pdf_error", err.Error())
		}
		return
	}
	// SEC: harden against a future bug where the storage adapter
	// returns a URL outside our control. gosec G710 flagged the
	// previous unconditional Redirect as an open-redirect taint sink.
	if _, vErr := validateStorageRedirect(rawURL); vErr != nil {
		slog.Error("invoice pdf: refusing to redirect to non-storage URL",
			"invoice_id", invoiceID, "url", rawURL, "error", vErr)
		res.Error(w, http.StatusBadGateway, "invoice_pdf_error",
			"presigned URL points outside the storage allowlist")
		return
	}
	// gosec G710: rawURL has been validated against the storage
	// allowlist immediately above (validateStorageRedirect). The
	// taint analyzer can't trace through the closure, so we suppress
	// here with the explicit gate as the rationale.
	http.Redirect(w, r, rawURL, http.StatusFound) // #nosec G710 -- validateStorageRedirect gate above
}

// CurrentMonth — GET /api/v1/me/invoicing/current-month
func (h *InvoiceHandler) CurrentMonth(w http.ResponseWriter, r *http.Request) {
	if h.svc == nil {
		res.Error(w, http.StatusServiceUnavailable, "invoicing_disabled", "invoicing feature not configured")
		return
	}
	orgID, ok := h.requireOrg(w, r)
	if !ok {
		return
	}
	agg, err := h.svc.GetCurrentMonthAggregate(r.Context(), orgID)
	if err != nil {
		res.Error(w, http.StatusInternalServerError, "current_month_error", err.Error())
		return
	}
	out := currentMonthResponse{
		PeriodStart:    agg.PeriodStart.UTC().Format(time.RFC3339),
		PeriodEnd:      agg.PeriodEnd.UTC().Format(time.RFC3339),
		MilestoneCount: agg.MilestoneCount,
		TotalFeeCents:  agg.TotalFeeCents,
		Lines:          make([]currentMonthLineResponse, 0, len(agg.Lines)),
	}
	for _, line := range agg.Lines {
		out.Lines = append(out.Lines, currentMonthLineResponse{
			MilestoneID:         line.MilestoneID.String(),
			PaymentRecordID:     line.PaymentRecordID.String(),
			ReleasedAt:          line.ReleasedAt.UTC().Format(time.RFC3339),
			PlatformFeeCents:    line.PlatformFeeCents,
			ProposalAmountCents: line.ProposalAmountCents,
		})
	}
	res.JSON(w, http.StatusOK, out)
}

// ---- helpers ----

func (h *InvoiceHandler) requireOrg(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	_, ok := middleware.GetUserID(r.Context())
	if !ok {
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
