package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	proposalapp "marketplace-backend/internal/app/proposal"
	domaininv "marketplace-backend/internal/domain/invoicing"
	proposaldomain "marketplace-backend/internal/domain/proposal"
	"marketplace-backend/internal/handler/middleware"
)

// stubBillingGate is a 5-line fake that satisfies the
// billingProfileGate interface so tests can drive every branch of the
// gate without standing up the real invoicing service.
type stubBillingGate struct {
	complete bool
	missing  []domaininv.MissingField
	err      error
	callsCh  chan struct {
		orgID uuid.UUID
	}
}

func (s *stubBillingGate) IsBillingProfileComplete(_ context.Context, orgID uuid.UUID) (bool, []domaininv.MissingField, error) {
	if s.callsCh != nil {
		s.callsCh <- struct{ orgID uuid.UUID }{orgID}
	}
	return s.complete, s.missing, s.err
}

// newGatedPaymentHandler wires a ProposalPaymentHandler with a stub
// billing gate. The proposal service is a real instance — the proposal
// repo is set to short-circuit with ErrProposalNotFound so the gate is
// exercised in isolation BEFORE the service performs any work. We
// assert on the gate's response, not the service's; tests that want to
// verify the service was reached check that the response is NOT 412.
func newGatedPaymentHandler(t *testing.T, gate *stubBillingGate) *ProposalPaymentHandler {
	t.Helper()
	repo := &mockProposalRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*proposaldomain.Proposal, error) {
			return nil, proposaldomain.ErrProposalNotFound
		},
	}
	svc := proposalapp.NewService(proposalapp.ServiceDeps{
		Proposals: repo,
		Users:     &mockUserRepo{},
		Messages:  &mockMessageSender{},
	})
	h := NewProposalPaymentHandler(svc, nil)
	return h.withBillingGate(gate)
}

// authedReq builds a POST request with userID + orgID populated in
// context and the chi URL param "id" set to a real UUID. The caller can
// override the orgID by passing it explicitly.
func authedReq(t *testing.T, userID, orgID, proposalID uuid.UUID) *http.Request {
	t.Helper()
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", proposalID.String())
	ctx := context.WithValue(r.Context(), chi.RouteCtxKey, rctx)
	ctx = context.WithValue(ctx, middleware.ContextKeyUserID, userID)
	ctx = context.WithValue(ctx, middleware.ContextKeyOrganizationID, orgID)
	return r.WithContext(ctx)
}

// ---------------------------------------------------------------------------
// PayProposal — billing gate
// ---------------------------------------------------------------------------

func TestPaymentHandler_PayProposal_BillingIncomplete_412(t *testing.T) {
	gate := &stubBillingGate{
		complete: false,
		missing: []domaininv.MissingField{
			{Field: "legal_name", Reason: "legal name is required"},
			{Field: "tax_id", Reason: "SIRET is required"},
		},
	}
	h := newGatedPaymentHandler(t, gate)

	req := authedReq(t, uuid.New(), uuid.New(), uuid.New())
	rec := httptest.NewRecorder()
	h.PayProposal(rec, req)

	assert.Equal(t, http.StatusPreconditionRequired, rec.Code,
		"incomplete billing profile must surface as 412")
	body := rec.Body.String()
	assert.Contains(t, body, "billing_profile_incomplete",
		"response must carry the discriminator code")
	assert.Contains(t, body, "missing_fields",
		"response must include the missing fields list")
	assert.Contains(t, body, "legal_name")
	assert.Contains(t, body, "tax_id")
}

func TestPaymentHandler_PayProposal_BillingComplete_DoesNotShortCircuit(t *testing.T) {
	gate := &stubBillingGate{complete: true}
	h := newGatedPaymentHandler(t, gate)

	req := authedReq(t, uuid.New(), uuid.New(), uuid.New())
	rec := httptest.NewRecorder()
	h.PayProposal(rec, req)

	// 412 must NOT be returned. Whatever the service answers (here it
	// will fail with proposal_not_found because the mock repo is empty)
	// the gate does NOT block the call.
	assert.NotEqual(t, http.StatusPreconditionRequired, rec.Code,
		"complete billing profile must let the request through to the service layer")
}

func TestPaymentHandler_PayProposal_GateNil_DoesNotBlock(t *testing.T) {
	// When invoicing is disabled (gate nil), the gate degrades open.
	// Same posture as WalletHandler — invoicing is removable and must
	// never block payments when not wired.
	repo := &mockProposalRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*proposaldomain.Proposal, error) {
			return nil, proposaldomain.ErrProposalNotFound
		},
	}
	svc := proposalapp.NewService(proposalapp.ServiceDeps{
		Proposals: repo,
		Users:     &mockUserRepo{},
		Messages:  &mockMessageSender{},
	})
	h := NewProposalPaymentHandler(svc, nil)
	require.Nil(t, h.invoicingSvc, "gate must be nil before WithInvoicing is called")

	req := authedReq(t, uuid.New(), uuid.New(), uuid.New())
	rec := httptest.NewRecorder()
	h.PayProposal(rec, req)

	assert.NotEqual(t, http.StatusPreconditionRequired, rec.Code,
		"nil gate must not surface a billing 412")
}

func TestPaymentHandler_PayProposal_GateProbeError_FailsOpen(t *testing.T) {
	// Probe errors are logged and the request is allowed through.
	// This matches the wallet handler's posture — fail-open is the
	// safer default because a transient invoicing-side failure must
	// not block real money flows on a near-final production app.
	gate := &stubBillingGate{err: errors.New("vies down")}
	h := newGatedPaymentHandler(t, gate)

	req := authedReq(t, uuid.New(), uuid.New(), uuid.New())
	rec := httptest.NewRecorder()
	h.PayProposal(rec, req)

	assert.NotEqual(t, http.StatusPreconditionRequired, rec.Code,
		"gate probe error must not surface as 412 — fail-open posture")
}

func TestPaymentHandler_PayProposal_GateUsesCallerOrgID(t *testing.T) {
	// The gate must check the CALLER's organization (the client side),
	// not some other id. Capture the orgID the gate was called with.
	gateCh := make(chan struct{ orgID uuid.UUID }, 1)
	gate := &stubBillingGate{complete: false, callsCh: gateCh}
	h := newGatedPaymentHandler(t, gate)

	expectedOrgID := uuid.New()
	req := authedReq(t, uuid.New(), expectedOrgID, uuid.New())
	rec := httptest.NewRecorder()
	h.PayProposal(rec, req)

	require.Equal(t, http.StatusPreconditionRequired, rec.Code)
	select {
	case call := <-gateCh:
		assert.Equal(t, expectedOrgID, call.orgID,
			"gate must be invoked with the caller's organization id")
	default:
		t.Fatal("gate was never called")
	}
}

// ---------------------------------------------------------------------------
// FundMilestone — billing gate (parity with PayProposal)
// ---------------------------------------------------------------------------

func TestPaymentHandler_FundMilestone_BillingIncomplete_412(t *testing.T) {
	gate := &stubBillingGate{
		complete: false,
		missing:  []domaininv.MissingField{{Field: "address_line1", Reason: "address required"}},
	}
	h := newGatedPaymentHandler(t, gate)

	// FundMilestone uses two URL params (proposal id + milestone id).
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", uuid.New().String())
	rctx.URLParams.Add("mid", uuid.New().String())
	ctx := context.WithValue(r.Context(), chi.RouteCtxKey, rctx)
	ctx = context.WithValue(ctx, middleware.ContextKeyUserID, uuid.New())
	ctx = context.WithValue(ctx, middleware.ContextKeyOrganizationID, uuid.New())
	r = r.WithContext(ctx)

	rec := httptest.NewRecorder()
	h.FundMilestone(rec, r)

	assert.Equal(t, http.StatusPreconditionRequired, rec.Code,
		"FundMilestone must enforce the same billing gate as PayProposal")
	assert.Contains(t, rec.Body.String(), "billing_profile_incomplete")
	assert.Contains(t, rec.Body.String(), "address_line1")
}

// ---------------------------------------------------------------------------
// respondClientBillingProfileIncomplete — envelope shape
// ---------------------------------------------------------------------------

func TestRespondClientBillingProfileIncomplete_NilMissingDefaultsToEmptyArray(t *testing.T) {
	rec := httptest.NewRecorder()
	respondClientBillingProfileIncomplete(rec, nil)

	assert.Equal(t, http.StatusPreconditionRequired, rec.Code)
	body := rec.Body.String()
	assert.Contains(t, body, `"missing_fields":[]`,
		"nil missing slice must serialise as an empty JSON array, never null")
}

func TestRespondClientBillingProfileIncomplete_FrenchTutoiement(t *testing.T) {
	rec := httptest.NewRecorder()
	respondClientBillingProfileIncomplete(rec, []domaininv.MissingField{})

	body := rec.Body.String()
	// The user-facing message must use tutoiement per the project
	// language convention.
	assert.True(t, strings.Contains(body, "tes informations") || strings.Contains(body, "Complète"),
		"response message must use French tutoiement: got %q", body)
}
