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
// ProposalPaymentHandler — focused tests for the funding endpoints.
// The PaymentHandler exists to isolate the payment surface — these tests
// document that pure routing / parsing / auth work in isolation. The
// service-layer happy paths are exhaustively covered by the legacy
// ProposalHandler test suite which runs against the same underlying
// proposal app service through the facade.
// ---------------------------------------------------------------------------

func newTestProposalPaymentHandler(
	ur *mockUserRepo,
	pr *mockProposalRepo,
	ms *mockMessageSender,
	ns *mockNotificationSender,
	pp *mockPaymentProcessor,
	ss *mockStorageService,
) *ProposalPaymentHandler {
	svc := proposalapp.NewService(proposalapp.ServiceDeps{
		Proposals:     pr,
		Users:         ur,
		Messages:      ms,
		Notifications: ns,
		Payments:      pp,
		Storage:       ss,
	})
	return NewProposalPaymentHandler(svc, nil) // paymentSvc nil OK for parse/auth tests
}

// ---------------------------------------------------------------------------
// PayProposal — auth + parsing
// ---------------------------------------------------------------------------

func TestPaymentHandler_PayProposal_Unauthorized(t *testing.T) {
	h := newTestProposalPaymentHandler(&mockUserRepo{}, &mockProposalRepo{},
		&mockMessageSender{}, &mockNotificationSender{}, &mockPaymentProcessor{}, &mockStorageService{})

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	h.PayProposal(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestPaymentHandler_PayProposal_BadID(t *testing.T) {
	uid := uuid.New()
	h := newTestProposalPaymentHandler(&mockUserRepo{}, &mockProposalRepo{},
		&mockMessageSender{}, &mockNotificationSender{}, &mockPaymentProcessor{}, &mockStorageService{})

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req = proposalCtx(req, &uid, "bogus")
	rec := httptest.NewRecorder()
	h.PayProposal(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "invalid_proposal_id")
}

// ---------------------------------------------------------------------------
// ConfirmPayment — auth + parsing
// ---------------------------------------------------------------------------

func TestPaymentHandler_ConfirmPayment_NoUserID(t *testing.T) {
	h := newTestProposalPaymentHandler(&mockUserRepo{}, &mockProposalRepo{},
		&mockMessageSender{}, &mockNotificationSender{}, &mockPaymentProcessor{}, &mockStorageService{})

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	h.ConfirmPayment(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestPaymentHandler_ConfirmPayment_BadID(t *testing.T) {
	uid := uuid.New()
	h := newTestProposalPaymentHandler(&mockUserRepo{}, &mockProposalRepo{},
		&mockMessageSender{}, &mockNotificationSender{}, &mockPaymentProcessor{}, &mockStorageService{})

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req = proposalCtx(req, &uid, "bogus")
	rec := httptest.NewRecorder()
	h.ConfirmPayment(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// ---------------------------------------------------------------------------
// FundMilestone — auth + parsing of both URL params
// ---------------------------------------------------------------------------

func TestPaymentHandler_FundMilestone_Unauthorized(t *testing.T) {
	h := newTestProposalPaymentHandler(&mockUserRepo{}, &mockProposalRepo{},
		&mockMessageSender{}, &mockNotificationSender{}, &mockPaymentProcessor{}, &mockStorageService{})

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	h.FundMilestone(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// ---------------------------------------------------------------------------
// Sub-handler accessor stable identity
// ---------------------------------------------------------------------------

func TestProposalHandler_PaymentAccessor_StableIdentity(t *testing.T) {
	h := newTestProposalHandler(&mockUserRepo{}, &mockProposalRepo{},
		&mockMessageSender{}, &mockNotificationSender{}, &mockPaymentProcessor{}, &mockStorageService{})

	a := h.Payment()
	b := h.Payment()
	require.NotNil(t, a)
	assert.Same(t, a, b, "Payment() must return the same pointer across calls")
}
