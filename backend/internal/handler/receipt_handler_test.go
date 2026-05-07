package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	receiptapp "marketplace-backend/internal/app/receipt"
	domain "marketplace-backend/internal/domain/receipt"
	"marketplace-backend/internal/handler"
	"marketplace-backend/internal/handler/middleware"
)

// rcFakeRepo is the in-memory repo used by the handler tests. The
// service-level tests already cover the business logic; here we only
// exercise the HTTP layer (status codes, body shape, ownership
// rejection).
type rcFakeRepo struct {
	receipts map[uuid.UUID]*domain.Receipt
}

func (r *rcFakeRepo) ListForOrganization(ctx context.Context, orgID uuid.UUID, cursor string, limit int) ([]*domain.Receipt, string, error) {
	out := make([]*domain.Receipt, 0)
	for _, rec := range r.receipts {
		if rec.IsParty(orgID) {
			out = append(out, rec)
		}
	}
	return out, "", nil
}

func (r *rcFakeRepo) GetForOrganization(ctx context.Context, receiptID, orgID uuid.UUID) (*domain.Receipt, error) {
	rec, ok := r.receipts[receiptID]
	if !ok {
		return nil, domain.ErrNotFound
	}
	if !rec.IsParty(orgID) {
		return nil, domain.ErrForbidden
	}
	return rec, nil
}

type rcFakeRenderer struct{ out []byte }

func (f *rcFakeRenderer) RenderReceipt(ctx context.Context, rec *domain.Receipt, language string) ([]byte, error) {
	if len(f.out) == 0 {
		return []byte("PDF"), nil
	}
	return f.out, nil
}

type rcHarness struct {
	handler *handler.ReceiptHandler
	repo    *rcFakeRepo
	userID  uuid.UUID
	orgID   uuid.UUID
}

func newRcHarness(t *testing.T) *rcHarness {
	repo := &rcFakeRepo{receipts: map[uuid.UUID]*domain.Receipt{}}
	svc := receiptapp.NewService(receiptapp.ServiceDeps{
		Repo:     repo,
		Renderer: &rcFakeRenderer{},
	})
	return &rcHarness{
		handler: handler.NewReceiptHandler(svc),
		repo:    repo,
		userID:  uuid.New(),
		orgID:   uuid.New(),
	}
}

func (h *rcHarness) reqAuth(method, target string) *http.Request {
	req := httptest.NewRequest(method, target, nil)
	ctx := req.Context()
	ctx = context.WithValue(ctx, middleware.ContextKeyUserID, h.userID)
	ctx = context.WithValue(ctx, middleware.ContextKeyOrganizationID, h.orgID)
	return req.WithContext(ctx)
}

func withChiID(req *http.Request, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", value)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

func makeReceipt(t *testing.T, clientOrg, providerOrg uuid.UUID) *domain.Receipt {
	t.Helper()
	return &domain.Receipt{
		ID:                uuid.New(),
		PaymentRecordID:   uuid.New(),
		ProposalID:        uuid.New(),
		AmountCents:       50000,
		Currency:          "EUR",
		CreatedAt:         time.Now().UTC(),
		SnapshotAvailable: true,
		Client:            &domain.PartyBilling{OrganizationID: clientOrg, Name: "Client SAS"},
		Provider:          &domain.PartyBilling{OrganizationID: providerOrg, Name: "Provider SARL"},
	}
}

// ---------- tests ----------

func TestReceiptHandler_List_ReturnsPartyReceipts(t *testing.T) {
	h := newRcHarness(t)
	other := uuid.New()
	mine := makeReceipt(t, h.orgID, other)
	notMine := makeReceipt(t, uuid.New(), uuid.New())
	h.repo.receipts[mine.ID] = mine
	h.repo.receipts[notMine.ID] = notMine

	w := httptest.NewRecorder()
	h.handler.List(w, h.reqAuth(http.MethodGet, "/api/v1/receipts"))

	assert.Equal(t, http.StatusOK, w.Code)
	var body struct {
		Data []map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Len(t, body.Data, 1)
	assert.Equal(t, mine.ID.String(), body.Data[0]["id"])
}

func TestReceiptHandler_List_NoOrg_Forbidden(t *testing.T) {
	h := newRcHarness(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/receipts", nil)
	// only user id set, no org id
	req = req.WithContext(context.WithValue(req.Context(), middleware.ContextKeyUserID, h.userID))

	w := httptest.NewRecorder()
	h.handler.List(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestReceiptHandler_Get_Party_Returns200(t *testing.T) {
	h := newRcHarness(t)
	rec := makeReceipt(t, h.orgID, uuid.New())
	h.repo.receipts[rec.ID] = rec

	req := withChiID(h.reqAuth(http.MethodGet, "/api/v1/receipts/"+rec.ID.String()), rec.ID.String())
	w := httptest.NewRecorder()
	h.handler.Get(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, rec.ID.String(), body["id"])
	assert.Equal(t, true, body["snapshot_available"])
	require.NotNil(t, body["client"])
}

func TestReceiptHandler_Get_NotParty_Returns403(t *testing.T) {
	h := newRcHarness(t)
	rec := makeReceipt(t, uuid.New(), uuid.New()) // someone else's parties
	h.repo.receipts[rec.ID] = rec

	req := withChiID(h.reqAuth(http.MethodGet, "/api/v1/receipts/"+rec.ID.String()), rec.ID.String())
	w := httptest.NewRecorder()
	h.handler.Get(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestReceiptHandler_Get_NotFound_Returns404(t *testing.T) {
	h := newRcHarness(t)
	id := uuid.New().String()
	req := withChiID(h.reqAuth(http.MethodGet, "/api/v1/receipts/"+id), id)
	w := httptest.NewRecorder()
	h.handler.Get(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestReceiptHandler_Get_InvalidID_Returns400(t *testing.T) {
	h := newRcHarness(t)
	req := withChiID(h.reqAuth(http.MethodGet, "/api/v1/receipts/not-a-uuid"), "not-a-uuid")
	w := httptest.NewRecorder()
	h.handler.Get(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestReceiptHandler_GetPDF_Party_Returns200(t *testing.T) {
	h := newRcHarness(t)
	rec := makeReceipt(t, h.orgID, uuid.New())
	h.repo.receipts[rec.ID] = rec

	req := withChiID(h.reqAuth(http.MethodGet, "/api/v1/receipts/"+rec.ID.String()+"/pdf"), rec.ID.String())
	w := httptest.NewRecorder()
	h.handler.GetPDF(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/pdf", w.Header().Get("Content-Type"))
	assert.NotEmpty(t, w.Body.Bytes())
}

func TestReceiptHandler_GetPDF_NotParty_Returns403(t *testing.T) {
	h := newRcHarness(t)
	rec := makeReceipt(t, uuid.New(), uuid.New())
	h.repo.receipts[rec.ID] = rec

	req := withChiID(h.reqAuth(http.MethodGet, "/api/v1/receipts/"+rec.ID.String()+"/pdf"), rec.ID.String())
	w := httptest.NewRecorder()
	h.handler.GetPDF(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestReceiptHandler_GetPDF_NoRenderer_Returns503(t *testing.T) {
	repo := &rcFakeRepo{receipts: map[uuid.UUID]*domain.Receipt{}}
	svc := receiptapp.NewService(receiptapp.ServiceDeps{Repo: repo})
	h := handler.NewReceiptHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/receipts/"+uuid.New().String()+"/pdf", nil)
	ctx := req.Context()
	ctx = context.WithValue(ctx, middleware.ContextKeyUserID, uuid.New())
	ctx = context.WithValue(ctx, middleware.ContextKeyOrganizationID, uuid.New())
	req = withChiID(req.WithContext(ctx), uuid.New().String())

	w := httptest.NewRecorder()
	h.GetPDF(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestReceiptHandler_NilService_503(t *testing.T) {
	h := handler.NewReceiptHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/receipts", nil)
	w := httptest.NewRecorder()
	h.List(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

// Verify a list with empty results renders [] (not null).
func TestReceiptHandler_List_EmptyArray_NotNull(t *testing.T) {
	h := newRcHarness(t)
	w := httptest.NewRecorder()
	h.handler.List(w, h.reqAuth(http.MethodGet, "/api/v1/receipts"))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"data":[]`)
}
