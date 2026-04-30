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
// ProposalCompletionHandler — focused tests for the completion surface.
// As with PaymentHandler, the service-layer happy paths are covered by
// the legacy ProposalHandler test suite. These tests prove the focused
// handler routes correctly through auth + URL parsing.
// ---------------------------------------------------------------------------

func newTestProposalCompletionHandler(
	ur *mockUserRepo,
	pr *mockProposalRepo,
	ms *mockMessageSender,
	ns *mockNotificationSender,
	pp *mockPaymentProcessor,
	ss *mockStorageService,
) *ProposalCompletionHandler {
	svc := proposalapp.NewService(proposalapp.ServiceDeps{
		Proposals:     pr,
		Users:         ur,
		Messages:      ms,
		Notifications: ns,
		Payments:      pp,
		Storage:       ss,
	})
	return NewProposalCompletionHandler(svc)
}

// ---------------------------------------------------------------------------
// RequestCompletion / CompleteProposal / RejectCompletion
// ---------------------------------------------------------------------------

func TestCompletionHandler_RequestCompletion_Unauthorized(t *testing.T) {
	h := newTestProposalCompletionHandler(&mockUserRepo{}, &mockProposalRepo{},
		&mockMessageSender{}, &mockNotificationSender{}, &mockPaymentProcessor{}, &mockStorageService{})

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	h.RequestCompletion(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestCompletionHandler_RequestCompletion_BadID(t *testing.T) {
	uid := uuid.New()
	h := newTestProposalCompletionHandler(&mockUserRepo{}, &mockProposalRepo{},
		&mockMessageSender{}, &mockNotificationSender{}, &mockPaymentProcessor{}, &mockStorageService{})

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req = proposalCtx(req, &uid, "bogus")
	rec := httptest.NewRecorder()
	h.RequestCompletion(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "invalid_proposal_id")
}

func TestCompletionHandler_CompleteProposal_Unauthorized(t *testing.T) {
	h := newTestProposalCompletionHandler(&mockUserRepo{}, &mockProposalRepo{},
		&mockMessageSender{}, &mockNotificationSender{}, &mockPaymentProcessor{}, &mockStorageService{})

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	h.CompleteProposal(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestCompletionHandler_CompleteProposal_BadID(t *testing.T) {
	uid := uuid.New()
	h := newTestProposalCompletionHandler(&mockUserRepo{}, &mockProposalRepo{},
		&mockMessageSender{}, &mockNotificationSender{}, &mockPaymentProcessor{}, &mockStorageService{})

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req = proposalCtx(req, &uid, "bogus")
	rec := httptest.NewRecorder()
	h.CompleteProposal(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCompletionHandler_RejectCompletion_Unauthorized(t *testing.T) {
	h := newTestProposalCompletionHandler(&mockUserRepo{}, &mockProposalRepo{},
		&mockMessageSender{}, &mockNotificationSender{}, &mockPaymentProcessor{}, &mockStorageService{})

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	h.RejectCompletion(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestCompletionHandler_RejectCompletion_BadID(t *testing.T) {
	uid := uuid.New()
	h := newTestProposalCompletionHandler(&mockUserRepo{}, &mockProposalRepo{},
		&mockMessageSender{}, &mockNotificationSender{}, &mockPaymentProcessor{}, &mockStorageService{})

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req = proposalCtx(req, &uid, "bogus")
	rec := httptest.NewRecorder()
	h.RejectCompletion(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// ---------------------------------------------------------------------------
// SubmitMilestone / ApproveMilestone / RejectMilestone
// ---------------------------------------------------------------------------

func TestCompletionHandler_SubmitMilestone_Unauthorized(t *testing.T) {
	h := newTestProposalCompletionHandler(&mockUserRepo{}, &mockProposalRepo{},
		&mockMessageSender{}, &mockNotificationSender{}, &mockPaymentProcessor{}, &mockStorageService{})

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	h.SubmitMilestone(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestCompletionHandler_ApproveMilestone_Unauthorized(t *testing.T) {
	h := newTestProposalCompletionHandler(&mockUserRepo{}, &mockProposalRepo{},
		&mockMessageSender{}, &mockNotificationSender{}, &mockPaymentProcessor{}, &mockStorageService{})

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	h.ApproveMilestone(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestCompletionHandler_RejectMilestone_Unauthorized(t *testing.T) {
	h := newTestProposalCompletionHandler(&mockUserRepo{}, &mockProposalRepo{},
		&mockMessageSender{}, &mockNotificationSender{}, &mockPaymentProcessor{}, &mockStorageService{})

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	h.RejectMilestone(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// ---------------------------------------------------------------------------
// Sub-handler accessor stable identity
// ---------------------------------------------------------------------------

func TestProposalHandler_CompletionAccessor_StableIdentity(t *testing.T) {
	h := newTestProposalHandler(&mockUserRepo{}, &mockProposalRepo{},
		&mockMessageSender{}, &mockNotificationSender{}, &mockPaymentProcessor{}, &mockStorageService{})

	a := h.Completion()
	b := h.Completion()
	require.NotNil(t, a)
	assert.Same(t, a, b, "Completion() must return the same pointer across calls")
}
