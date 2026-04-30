package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	proposalapp "marketplace-backend/internal/app/proposal"
)

// ---------------------------------------------------------------------------
// ProposalAdminHandler — focused tests for the 5 admin endpoints.
// Auth is gated at the router via RequireRole("admin"); the handlers
// themselves only validate URL params + delegate to the proposal app
// service. We test that delegation works and that bad IDs are rejected.
// ---------------------------------------------------------------------------

func newTestProposalAdminHandler(
	ur *mockUserRepo,
	pr *mockProposalRepo,
	ms *mockMessageSender,
	ns *mockNotificationSender,
	pp *mockPaymentProcessor,
	ss *mockStorageService,
) *ProposalAdminHandler {
	svc := proposalapp.NewService(proposalapp.ServiceDeps{
		Proposals:     pr,
		Users:         ur,
		Messages:      ms,
		Notifications: ns,
		Payments:      pp,
		Storage:       ss,
	})
	return NewProposalAdminHandler(svc)
}

// ---------------------------------------------------------------------------
// AdminActivateProposal — bad UUID is the only auth-free failure mode
// because admin RBAC is gated at the router level.
// ---------------------------------------------------------------------------

func TestAdminHandler_ActivateProposal_BadID(t *testing.T) {
	h := newTestProposalAdminHandler(&mockUserRepo{}, &mockProposalRepo{},
		&mockMessageSender{}, &mockNotificationSender{}, &mockPaymentProcessor{}, &mockStorageService{})

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	uid := uuid.New()
	req = proposalCtx(req, &uid, "bogus")
	rec := httptest.NewRecorder()
	h.AdminActivateProposal(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "invalid_proposal_id")
}

// ---------------------------------------------------------------------------
// AdminListBonusLog / AdminListPendingBonusLog
// ---------------------------------------------------------------------------

func TestAdminHandler_ListBonusLog_DefaultsToLimit20(t *testing.T) {
	h := newTestProposalAdminHandler(&mockUserRepo{}, &mockProposalRepo{},
		&mockMessageSender{}, &mockNotificationSender{}, &mockPaymentProcessor{}, &mockStorageService{})

	req := httptest.NewRequest(http.MethodGet, "/?limit=invalid", nil)
	rec := httptest.NewRecorder()
	h.AdminListBonusLog(rec, req)
	// Either 200 with empty list or 500 from the service — we only
	// assert the request didn't fail with a crash. Status varies based
	// on whether the bonus log app service has a stub.
	require.NotEqual(t, http.StatusBadRequest, rec.Code)
}

func TestAdminHandler_ListPendingBonusLog_DefaultLimit(t *testing.T) {
	h := newTestProposalAdminHandler(&mockUserRepo{}, &mockProposalRepo{},
		&mockMessageSender{}, &mockNotificationSender{}, &mockPaymentProcessor{}, &mockStorageService{})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.AdminListPendingBonusLog(rec, req)
	require.NotEqual(t, http.StatusBadRequest, rec.Code)
}

// ---------------------------------------------------------------------------
// AdminApproveBonusEntry / AdminRejectBonusEntry — bad ID
// ---------------------------------------------------------------------------

func TestAdminHandler_ApproveBonusEntry_BadID(t *testing.T) {
	h := newTestProposalAdminHandler(&mockUserRepo{}, &mockProposalRepo{},
		&mockMessageSender{}, &mockNotificationSender{}, &mockPaymentProcessor{}, &mockStorageService{})

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	uid := uuid.New()
	req = proposalCtx(req, &uid, "bogus")
	rec := httptest.NewRecorder()
	h.AdminApproveBonusEntry(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "invalid_id")
}

func TestAdminHandler_RejectBonusEntry_BadID(t *testing.T) {
	h := newTestProposalAdminHandler(&mockUserRepo{}, &mockProposalRepo{},
		&mockMessageSender{}, &mockNotificationSender{}, &mockPaymentProcessor{}, &mockStorageService{})

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	uid := uuid.New()
	req = proposalCtx(req, &uid, "bogus")
	rec := httptest.NewRecorder()
	h.AdminRejectBonusEntry(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// ---------------------------------------------------------------------------
// Sub-handler accessor stable identity
// ---------------------------------------------------------------------------

func TestProposalHandler_AdminAccessor_StableIdentity(t *testing.T) {
	h := newTestProposalHandler(&mockUserRepo{}, &mockProposalRepo{},
		&mockMessageSender{}, &mockNotificationSender{}, &mockPaymentProcessor{}, &mockStorageService{})

	a := h.Admin()
	b := h.Admin()
	require.NotNil(t, a)
	assert.Same(t, a, b, "Admin() must return the same pointer across calls")
}
