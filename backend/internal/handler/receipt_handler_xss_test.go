package handler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	receiptapp "marketplace-backend/internal/app/receipt"
	domain "marketplace-backend/internal/domain/receipt"
	"marketplace-backend/internal/handler"
	"marketplace-backend/internal/handler/middleware"
)

// langCapturingRenderer records the language string the renderer
// receives so the boundary-validation tests can assert that no
// untrusted query param ever reaches the rendering layer. Closes
// CodeQL #63 (go/xss G705) at its source.
type langCapturingRenderer struct {
	gotLanguage string
}

func (r *langCapturingRenderer) RenderReceipt(_ context.Context, _ *domain.Receipt, language string) ([]byte, error) {
	r.gotLanguage = language
	return []byte("PDF"), nil
}

func newRcXSSHarness(t *testing.T) (*handler.ReceiptHandler, *langCapturingRenderer, uuid.UUID, uuid.UUID) {
	t.Helper()
	repo := &rcFakeRepo{receipts: map[uuid.UUID]*domain.Receipt{}}
	renderer := &langCapturingRenderer{}
	svc := receiptapp.NewService(receiptapp.ServiceDeps{
		Repo:     repo,
		Renderer: renderer,
	})
	h := handler.NewReceiptHandler(svc)
	orgID := uuid.New()
	rec := &domain.Receipt{
		ID:              uuid.New(),
		PaymentRecordID: uuid.New(),
		AmountCents:     1000,
		Currency:        "EUR",
		Client:          &domain.PartyBilling{OrganizationID: orgID, Name: "Acme SAS"},
		Provider:        &domain.PartyBilling{OrganizationID: uuid.New(), Name: "Provider SARL"},
	}
	repo.receipts[rec.ID] = rec
	return h, renderer, rec.ID, orgID
}

// TestReceiptHandler_GetPDF_LangAllowlist asserts the handler maps
// every value of the user-controlled `lang` query param to the strict
// {"fr","en"} allowlist BEFORE invoking the renderer. Hostile payloads
// (script tags, control bytes, traversal, mixed case) all collapse to
// "fr" — no taint flow survives the boundary.
func TestReceiptHandler_GetPDF_LangAllowlist(t *testing.T) {
	tests := []struct {
		name     string
		rawLang  string
		expected string
	}{
		{"empty_defaults_to_fr", "", "fr"},
		{"fr_passthrough", "fr", "fr"},
		{"en_passthrough", "en", "en"},
		{"uppercase_fr_normalised", "FR", "fr"},
		{"uppercase_en_normalised", "EN", "en"},
		{"mixed_case_en_normalised", "En", "en"},
		{"surrounding_whitespace_trimmed", "  fr  ", "fr"},
		{"unsupported_locale_falls_back", "de", "fr"},
		{"long_string_falls_back", "frfrfrfrfrfrfrfrfrfr", "fr"},

		// Hostile XSS-style payloads must NEVER reach the renderer
		// untouched. These are the exact patterns gosec's taint
		// analysis flagged on `w.Write(pdf)` (#63 — go/xss G705).
		{"script_tag_falls_back", "\"><script>alert(1)</script>", "fr"},
		{"javascript_uri_falls_back", "javascript:alert(1)", "fr"},
		{"img_onerror_falls_back", "<img src=x onerror=alert(1)>", "fr"},
		{"sql_quote_falls_back", "fr' OR 1=1--", "fr"},
		{"null_byte_falls_back", "fr\x00", "fr"},
		{"newline_injection_falls_back", "fr\r\nSet-Cookie: x=y", "fr"},
		{"path_traversal_falls_back", "../../etc/passwd", "fr"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, renderer, recID, orgID := newRcXSSHarness(t)

			target := "/api/v1/receipts/" + recID.String() + "/pdf"
			if tt.rawLang != "" {
				target += "?lang=" + url.QueryEscape(tt.rawLang)
			}
			req := httptest.NewRequest(http.MethodGet, target, nil)
			ctx := req.Context()
			ctx = context.WithValue(ctx, middleware.ContextKeyUserID, uuid.New())
			ctx = context.WithValue(ctx, middleware.ContextKeyOrganizationID, orgID)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", recID.String())
			ctx = context.WithValue(ctx, chi.RouteCtxKey, rctx)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()
			h.GetPDF(w, req)

			require.Equal(t, http.StatusOK, w.Code, "PDF render must succeed regardless of input")
			assert.Equal(t, tt.expected, renderer.gotLanguage,
				"renderer must only receive an allowlisted language — taint stopped at the boundary")
		})
	}
}

// TestReceiptHandler_GetPDF_ContentTypeAlwaysPDF asserts the response
// Content-Type is application/pdf for every input. Even if the
// language allowlist somehow regressed, browsers would not HTML-render
// the body. Defense-in-depth probe for #63.
func TestReceiptHandler_GetPDF_ContentTypeAlwaysPDF(t *testing.T) {
	h, _, recID, orgID := newRcXSSHarness(t)
	target := "/api/v1/receipts/" + recID.String() + "/pdf?lang=" + url.QueryEscape("\"><script>alert(1)</script>")
	req := httptest.NewRequest(http.MethodGet, target, nil)
	ctx := req.Context()
	ctx = context.WithValue(ctx, middleware.ContextKeyUserID, uuid.New())
	ctx = context.WithValue(ctx, middleware.ContextKeyOrganizationID, orgID)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", recID.String())
	ctx = context.WithValue(ctx, chi.RouteCtxKey, rctx)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	h.GetPDF(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/pdf", w.Header().Get("Content-Type"),
		"hostile lang input must not alter Content-Type — second layer of XSS defense")
	// Filename interpolation only uses the parsed UUID, never user
	// input, so the header is structurally safe.
	assert.Contains(t, w.Header().Get("Content-Disposition"), recID.String())
}
