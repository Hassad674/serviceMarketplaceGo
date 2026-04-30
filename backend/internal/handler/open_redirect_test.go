package handler_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	invoicingapp "marketplace-backend/internal/app/invoicing"
	domain "marketplace-backend/internal/domain/invoicing"
	"marketplace-backend/internal/handler"
	"marketplace-backend/internal/handler/middleware"
)

// withInvoiceUserCtx attaches user + org IDs to the request context
// the same way the auth middleware would. Used by the open-redirect
// suite to drive InvoiceHandler.GetPDF without spinning up the full
// router.
func withInvoiceUserCtx(req *http.Request, userID, orgID uuid.UUID) *http.Request {
	ctx := context.WithValue(req.Context(), middleware.ContextKeyUserID, userID)
	ctx = context.WithValue(ctx, middleware.ContextKeyOrganizationID, orgID)
	return req.WithContext(ctx)
}

// hostileStorage is a service.StorageService that returns a
// caller-controlled URL from the presigned download endpoints. Used
// to simulate a (hypothetical) compromised storage adapter that
// produces an attacker-host URL — exactly the scenario gosec G710
// flagged on invoice_handler.go:145 / admin_invoice_handler.go:145.
type hostileStorage struct{ url string }

func (h hostileStorage) Upload(_ context.Context, _ string, _ io.Reader, _ string, _ int64) (string, error) {
	return h.url, nil
}
func (hostileStorage) Delete(_ context.Context, _ string) error { return nil }
func (h hostileStorage) GetPublicURL(_ string) string           { return h.url }
func (h hostileStorage) GetPresignedUploadURL(_ context.Context, _, _ string, _ time.Duration) (string, error) {
	return h.url, nil
}
func (h hostileStorage) GetPresignedDownloadURL(_ context.Context, _ string, _ time.Duration) (string, error) {
	return h.url, nil
}
func (h hostileStorage) GetPresignedDownloadURLAsAttachment(_ context.Context, _ string, _ string, _ time.Duration) (string, error) {
	return h.url, nil
}
func (hostileStorage) Download(_ context.Context, _ string) ([]byte, error) {
	return nil, nil
}

// newHandlerWithHostileStorage builds an InvoiceHandler whose
// underlying storage adapter returns the supplied URL on every
// presigned-download call. The harness pre-populates the invoice
// repository so the GetPDF handler reaches the redirect step.
func newHandlerWithHostileStorage(t *testing.T, attackURL string) (*handler.InvoiceHandler, uuid.UUID, uuid.UUID) {
	t.Helper()
	repo := newInvRepo()
	orgID := uuid.New()
	inv := makeInvoice(t, orgID, "FAC-000001", time.Now().UTC())
	repo.byID[inv.ID] = inv

	svc := invoicingapp.NewService(invoicingapp.ServiceDeps{
		Invoices:    repo,
		Profiles:    newBPRepo(),
		PDF:         bpFakePDF{},
		Storage:     hostileStorage{url: attackURL},
		Deliverer:   bpFakeDeliverer{},
		Issuer:      domain.IssuerInfo{Country: "FR", LegalName: "Test"},
		Idempotency: bpFakeIdempotency{},
	})
	return handler.NewInvoiceHandler(svc), orgID, inv.ID
}

// TestInvoice_GetPDF_RejectsOpenRedirectPayloads is the regression
// guard for SEC-related G710: every attack payload below must
// produce 502 instead of a 302-redirect to the attacker-controlled
// destination. The fake storage simulates a misbehaving adapter that
// hands back a hostile URL — without the validateStorageRedirect
// gate, the handler would forward the user agent to that URL.
func TestInvoice_GetPDF_RejectsOpenRedirectPayloads(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{"javascript URI", "javascript:alert(document.cookie)"},
		{"data URI", "data:text/html,<script>fetch('/api/v1/me')</script>"},
		{"protocol-relative", "//evil.com/exfil"},
		{"attacker host", "https://attacker.example.com/file.pdf"},
		{"suffix-confusion", "https://amazonaws.com.attacker.com/file.pdf"},
		{"CRLF smuggling", "https://valid.r2.dev/file.pdf\r\nLocation: https://evil.com"},
		{"file URI", "file:///etc/passwd"},
		{"empty URL", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, orgID, invID := newHandlerWithHostileStorage(t, tt.url)
			req := httptest.NewRequest(http.MethodGet,
				"/api/v1/me/invoices/"+invID.String()+"/pdf", nil)
			req = withChiURLParam(req, "id", invID.String())
			req = withInvoiceUserCtx(req, uuid.New(), orgID)
			rec := httptest.NewRecorder()
			h.GetPDF(rec, req)

			assert.Equal(t, http.StatusBadGateway, rec.Code,
				"attack URL %q should be rejected, got %d %s",
				tt.url, rec.Code, rec.Body.String())
			// Crucial: the Location header MUST NOT be set —
			// otherwise the user agent follows it.
			assert.Empty(t, rec.Header().Get("Location"))
		})
	}
}

// TestInvoice_GetPDF_AcceptsAllowlistedHosts confirms every storage
// host the application uses today survives validation. If a new
// backend is wired up and the test fails, the operator must
// explicitly add the suffix to allowedRedirectHostSuffixes — there
// is no silent-broaden path.
func TestInvoice_GetPDF_AcceptsAllowlistedHosts(t *testing.T) {
	tests := []string{
		"https://my-bucket.s3.eu-west-3.amazonaws.com/file.pdf?X-Amz-Signature=abc",
		"https://x.r2.cloudflarestorage.com/file.pdf?X-Amz-Signature=abc",
		"https://pub-abc.r2.dev/file.pdf",
		"http://minio:9000/bucket/file.pdf",
		"http://localhost:9000/bucket/file.pdf",
		"http://127.0.0.1:9000/bucket/file.pdf",
	}
	for _, u := range tests {
		t.Run(u, func(t *testing.T) {
			h, orgID, invID := newHandlerWithHostileStorage(t, u)
			req := httptest.NewRequest(http.MethodGet,
				"/api/v1/me/invoices/"+invID.String()+"/pdf", nil)
			req = withChiURLParam(req, "id", invID.String())
			req = withInvoiceUserCtx(req, uuid.New(), orgID)
			rec := httptest.NewRecorder()
			h.GetPDF(rec, req)

			require.Equal(t, http.StatusFound, rec.Code,
				"allowlisted URL %q should redirect, got %d %s",
				u, rec.Code, rec.Body.String())
			assert.Equal(t, u, rec.Header().Get("Location"))
		})
	}
}

// ---- admin invoice handler — same coverage on the admin path. ----

func newAdminHandlerWithHostileStorage(t *testing.T, attackURL string) (*handler.AdminInvoiceHandler, uuid.UUID) {
	t.Helper()
	repo := newInvRepo()
	orgID := uuid.New()
	inv := makeInvoice(t, orgID, "FAC-000001", time.Now().UTC())
	repo.byID[inv.ID] = inv

	svc := invoicingapp.NewService(invoicingapp.ServiceDeps{
		Invoices:    repo,
		Profiles:    newBPRepo(),
		PDF:         bpFakePDF{},
		Storage:     hostileStorage{url: attackURL},
		Deliverer:   bpFakeDeliverer{},
		Issuer:      domain.IssuerInfo{Country: "FR", LegalName: "Test"},
		Idempotency: bpFakeIdempotency{},
	})
	return handler.NewAdminInvoiceHandler(svc), inv.ID
}

func TestAdminInvoice_GetPDF_RejectsOpenRedirectPayloads(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{"javascript URI", "javascript:alert(1)"},
		{"protocol-relative", "//evil.com/x"},
		{"attacker host", "https://attacker.com/x"},
		{"suffix-confusion", "https://r2.dev.attacker.com/x"},
		{"CRLF smuggling", "https://x.r2.dev/x\nLocation: https://evil.com"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, invID := newAdminHandlerWithHostileStorage(t, tt.url)
			req := httptest.NewRequest(http.MethodGet,
				"/api/v1/admin/invoices/"+invID.String()+"/pdf", nil)
			req = withChiURLParam(req, "id", invID.String())
			rec := httptest.NewRecorder()
			h.GetPDF(rec, req)

			assert.Equal(t, http.StatusBadGateway, rec.Code,
				"admin attack URL %q should be rejected, got %d %s",
				tt.url, rec.Code, rec.Body.String())
			assert.Empty(t, rec.Header().Get("Location"))
		})
	}
}

func TestAdminInvoice_GetPDF_AcceptsAllowlistedHosts(t *testing.T) {
	tests := []string{
		"https://x.r2.cloudflarestorage.com/x.pdf",
		"https://my-bucket.s3.amazonaws.com/x.pdf",
		"https://pub-abc.r2.dev/x.pdf",
	}
	for _, u := range tests {
		t.Run(u, func(t *testing.T) {
			h, invID := newAdminHandlerWithHostileStorage(t, u)
			req := httptest.NewRequest(http.MethodGet,
				"/api/v1/admin/invoices/"+invID.String()+"/pdf", nil)
			req = withChiURLParam(req, "id", invID.String())
			rec := httptest.NewRecorder()
			h.GetPDF(rec, req)

			require.Equal(t, http.StatusFound, rec.Code, rec.Body.String())
			assert.Equal(t, u, rec.Header().Get("Location"))
		})
	}
}
