package handler

import (
	"bytes"
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

	proposalapp "marketplace-backend/internal/app/proposal"
	proposaldomain "marketplace-backend/internal/domain/proposal"
	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/handler/middleware"
	"marketplace-backend/internal/port/service"
)

// ---------------------------------------------------------------------------
// Mocks only for types NOT already defined elsewhere in the handler package.
// Reused: mockUserRepo, mockProposalRepo, mockNotificationSender,
//         mockStorageService, mockMessageSender (defined in call_handler_test.go).
// ---------------------------------------------------------------------------

type mockPaymentProcessor struct {
	createPaymentIntentFn  func(ctx context.Context, input service.PaymentIntentInput) (*service.PaymentIntentOutput, error)
	transferToProviderFn   func(ctx context.Context, proposalID uuid.UUID) error
	handlePaymentFn        func(ctx context.Context, piID string) (uuid.UUID, error)
	canProviderReceiveFn   func(ctx context.Context, providerOrgID uuid.UUID) (bool, error)
}

func (m *mockPaymentProcessor) CreatePaymentIntent(ctx context.Context, input service.PaymentIntentInput) (*service.PaymentIntentOutput, error) {
	if m.createPaymentIntentFn != nil {
		return m.createPaymentIntentFn(ctx, input)
	}
	return nil, nil
}

func (m *mockPaymentProcessor) TransferToProvider(ctx context.Context, proposalID uuid.UUID) error {
	if m.transferToProviderFn != nil {
		return m.transferToProviderFn(ctx, proposalID)
	}
	return nil
}

func (m *mockPaymentProcessor) TransferMilestone(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (m *mockPaymentProcessor) HandlePaymentSucceeded(ctx context.Context, piID string) (uuid.UUID, error) {
	if m.handlePaymentFn != nil {
		return m.handlePaymentFn(ctx, piID)
	}
	return uuid.Nil, nil
}

func (m *mockPaymentProcessor) TransferPartialToProvider(_ context.Context, _ uuid.UUID, _ int64) error {
	return nil
}

func (m *mockPaymentProcessor) RefundToClient(_ context.Context, _ uuid.UUID, _ int64) error {
	return nil
}

func (m *mockPaymentProcessor) CanProviderReceivePayouts(ctx context.Context, providerOrgID uuid.UUID) (bool, error) {
	if m.canProviderReceiveFn != nil {
		return m.canProviderReceiveFn(ctx, providerOrgID)
	}
	// Default: provider is ready. Existing tests rely on the happy path
	// for milestone-release flows.
	return true, nil
}

func (m *mockPaymentProcessor) HasAutoPayoutConsent(_ context.Context, _ uuid.UUID) (bool, error) {
	return false, nil
}

var _ service.PaymentProcessor = (*mockPaymentProcessor)(nil)

// ---------------------------------------------------------------------------
// Helper
// ---------------------------------------------------------------------------

func newTestProposalHandler(
	ur *mockUserRepo,
	pr *mockProposalRepo,
	ms *mockMessageSender,
	ns *mockNotificationSender,
	pp *mockPaymentProcessor,
	ss *mockStorageService,
) *ProposalHandler {
	svc := proposalapp.NewService(proposalapp.ServiceDeps{
		Proposals:     pr,
		Users:         ur,
		Messages:      ms,
		Notifications: ns,
		Payments:      pp,
		Storage:       ss,
	})
	return NewProposalHandler(svc, nil)
}

func sampleProposal(senderID, recipientID uuid.UUID) *proposaldomain.Proposal {
	now := time.Now()
	return &proposaldomain.Proposal{
		ID:             uuid.New(),
		ConversationID: uuid.New(),
		SenderID:       senderID,
		RecipientID:    recipientID,
		Title:          "Build REST API",
		Description:    "Develop a Go REST API for the platform",
		Amount:         500000,
		Status:         proposaldomain.StatusPending,
		Version:        1,
		ClientID:       recipientID,
		ProviderID:     senderID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

func proposalCtx(r *http.Request, userID *uuid.UUID, urlParamID string) *http.Request {
	ctx := r.Context()
	if urlParamID != "" {
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", urlParamID)
		ctx = context.WithValue(ctx, chi.RouteCtxKey, rctx)
	}
	if userID != nil {
		ctx = context.WithValue(ctx, middleware.ContextKeyUserID, *userID)
		// Also expose the user id as an org id so tests exercising
		// org-scoped list queries (ListActiveProjects) pass the
		// middleware check.
		ctx = context.WithValue(ctx, middleware.ContextKeyOrganizationID, *userID)
	}
	return r.WithContext(ctx)
}

func enterpriseUser(id uuid.UUID) *user.User {
	return &user.User{
		ID: id, Role: user.RoleEnterprise,
		DisplayName: "Enterprise Co",
		CreatedAt:   time.Now(), UpdatedAt: time.Now(),
	}
}

func providerUser(id uuid.UUID) *user.User {
	return &user.User{
		ID: id, Role: user.RoleProvider,
		DisplayName: "Provider Dev",
		CreatedAt:   time.Now(), UpdatedAt: time.Now(),
	}
}

// ---------------------------------------------------------------------------
// Tests — CreateProposal
// ---------------------------------------------------------------------------

func TestProposalHandler_CreateProposal(t *testing.T) {
	enterpriseID := uuid.New()
	providerID := uuid.New()
	convID := uuid.New()

	validBody := map[string]any{
		"recipient_id":    providerID.String(),
		"conversation_id": convID.String(),
		"title":           "Build REST API",
		"description":     "Develop a Go REST API",
		"amount":          500000,
	}

	tests := []struct {
		name       string
		userID     *uuid.UUID
		body       map[string]any
		setupMocks func(*mockUserRepo, *mockProposalRepo)
		wantStatus int
		wantCode   string
	}{
		{
			name:   "success",
			userID: &enterpriseID,
			body:   validBody,
			setupMocks: func(ur *mockUserRepo, pr *mockProposalRepo) {
				ur.getByIDFn = userByIDLookup(enterpriseID, providerID)
			},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "unauthenticated returns 401",
			userID:     nil,
			body:       validBody,
			wantStatus: http.StatusUnauthorized,
			wantCode:   "unauthorized",
		},
		{
			name:   "missing body returns 400",
			userID: &enterpriseID,
			body:   nil,
			wantStatus: http.StatusBadRequest,
			wantCode:   "invalid_request",
		},
		{
			name:   "empty title returns 400",
			userID: &enterpriseID,
			body: map[string]any{
				"recipient_id":    providerID.String(),
				"conversation_id": convID.String(),
				"title":           "",
				"description":     "Some desc",
				"amount":          500000,
			},
			setupMocks: func(ur *mockUserRepo, pr *mockProposalRepo) {
				ur.getByIDFn = userByIDLookup(enterpriseID, providerID)
			},
			wantStatus: http.StatusBadRequest,
			wantCode:   "empty_title",
		},
		{
			name:   "invalid recipient_id returns 400",
			userID: &enterpriseID,
			body: map[string]any{
				"recipient_id":    "not-a-uuid",
				"conversation_id": convID.String(),
				"title":           "Title",
				"description":     "Desc",
				"amount":          100,
			},
			wantStatus: http.StatusBadRequest,
			wantCode:   "invalid_recipient_id",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ur := &mockUserRepo{}
			pr := &mockProposalRepo{}
			if tc.setupMocks != nil {
				tc.setupMocks(ur, pr)
			}
			h := newTestProposalHandler(ur, pr, &mockMessageSender{}, &mockNotificationSender{}, &mockPaymentProcessor{}, &mockStorageService{})

			var body []byte
			if tc.body != nil {
				body, _ = json.Marshal(tc.body)
			}
			req := httptest.NewRequest(http.MethodPost, "/api/v1/proposals", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req = proposalCtx(req, tc.userID, "")
			rec := httptest.NewRecorder()

			h.CreateProposal(rec, req)
			assert.Equal(t, tc.wantStatus, rec.Code)
			assertProposalErrorCode(t, rec, tc.wantCode)
		})
	}
}

// ---------------------------------------------------------------------------
// Tests — GetProposal
// ---------------------------------------------------------------------------

func TestProposalHandler_GetProposal(t *testing.T) {
	senderID := uuid.New()
	recipientID := uuid.New()
	pID := uuid.New()

	tests := []struct {
		name       string
		userID     *uuid.UUID
		urlParam   string
		setupMocks func(*mockUserRepo, *mockProposalRepo)
		wantStatus int
		wantCode   string
	}{
		{
			name:     "success",
			userID:   &senderID,
			urlParam: pID.String(),
			setupMocks: func(ur *mockUserRepo, pr *mockProposalRepo) {
				pr.getByIDFn = func(_ context.Context, id uuid.UUID) (*proposaldomain.Proposal, error) {
					p := sampleProposal(senderID, recipientID)
					p.ID = id
					return p, nil
				}
				ur.getByIDFn = userByIDLookup(recipientID, senderID)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:     "not found returns 404",
			userID:   &senderID,
			urlParam: pID.String(),
			setupMocks: func(_ *mockUserRepo, pr *mockProposalRepo) {
				pr.getByIDFn = func(_ context.Context, _ uuid.UUID) (*proposaldomain.Proposal, error) {
					return nil, proposaldomain.ErrProposalNotFound
				}
			},
			wantStatus: http.StatusNotFound,
			wantCode:   "proposal_not_found",
		},
		{
			name:       "invalid uuid returns 400",
			userID:     &senderID,
			urlParam:   "bad-uuid",
			wantStatus: http.StatusBadRequest,
			wantCode:   "invalid_proposal_id",
		},
		{
			name:       "unauthenticated returns 401",
			userID:     nil,
			urlParam:   pID.String(),
			wantStatus: http.StatusUnauthorized,
			wantCode:   "unauthorized",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ur := &mockUserRepo{}
			pr := &mockProposalRepo{}
			if tc.setupMocks != nil {
				tc.setupMocks(ur, pr)
			}
			h := newTestProposalHandler(ur, pr, &mockMessageSender{}, &mockNotificationSender{}, &mockPaymentProcessor{}, &mockStorageService{})

			req := httptest.NewRequest(http.MethodGet, "/api/v1/proposals/"+tc.urlParam, nil)
			req = proposalCtx(req, tc.userID, tc.urlParam)
			rec := httptest.NewRecorder()

			h.GetProposal(rec, req)
			assert.Equal(t, tc.wantStatus, rec.Code)
			assertProposalErrorCode(t, rec, tc.wantCode)
		})
	}
}

// ---------------------------------------------------------------------------
// Tests — AcceptProposal
// ---------------------------------------------------------------------------

func TestProposalHandler_AcceptProposal(t *testing.T) {
	senderID := uuid.New()
	recipientID := uuid.New()
	pID := uuid.New()

	tests := []struct {
		name       string
		userID     *uuid.UUID
		urlParam   string
		setupMocks func(*mockUserRepo, *mockProposalRepo)
		wantStatus int
		wantCode   string
	}{
		{
			name:     "success",
			userID:   &recipientID,
			urlParam: pID.String(),
			setupMocks: func(ur *mockUserRepo, pr *mockProposalRepo) {
				pr.getByIDFn = func(_ context.Context, _ uuid.UUID) (*proposaldomain.Proposal, error) {
					return sampleProposal(senderID, recipientID), nil
				}
				ur.getByIDFn = userByIDLookup(senderID, recipientID)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:     "not authorized returns 403",
			userID:   &senderID,
			urlParam: pID.String(),
			setupMocks: func(ur *mockUserRepo, pr *mockProposalRepo) {
				pr.getByIDFn = func(_ context.Context, _ uuid.UUID) (*proposaldomain.Proposal, error) {
					return sampleProposal(senderID, recipientID), nil
				}
			},
			wantStatus: http.StatusForbidden,
			wantCode:   "not_authorized",
		},
		{
			name:       "unauthenticated returns 401",
			userID:     nil,
			urlParam:   pID.String(),
			wantStatus: http.StatusUnauthorized,
			wantCode:   "unauthorized",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ur := &mockUserRepo{}
			pr := &mockProposalRepo{}
			if tc.setupMocks != nil {
				tc.setupMocks(ur, pr)
			}
			h := newTestProposalHandler(ur, pr, &mockMessageSender{}, &mockNotificationSender{}, &mockPaymentProcessor{}, &mockStorageService{})

			req := httptest.NewRequest(http.MethodPost, "/api/v1/proposals/"+pID.String()+"/accept", nil)
			req = proposalCtx(req, tc.userID, tc.urlParam)
			rec := httptest.NewRecorder()

			h.AcceptProposal(rec, req)
			assert.Equal(t, tc.wantStatus, rec.Code)
			assertProposalErrorCode(t, rec, tc.wantCode)
		})
	}
}

// ---------------------------------------------------------------------------
// Tests — DeclineProposal
// ---------------------------------------------------------------------------

func TestProposalHandler_DeclineProposal(t *testing.T) {
	senderID := uuid.New()
	recipientID := uuid.New()
	pID := uuid.New()

	tests := []struct {
		name       string
		userID     *uuid.UUID
		urlParam   string
		setupMocks func(*mockProposalRepo)
		wantStatus int
		wantCode   string
	}{
		{
			name:     "success",
			userID:   &recipientID,
			urlParam: pID.String(),
			setupMocks: func(pr *mockProposalRepo) {
				pr.getByIDFn = func(_ context.Context, _ uuid.UUID) (*proposaldomain.Proposal, error) {
					return sampleProposal(senderID, recipientID), nil
				}
			},
			wantStatus: http.StatusOK,
		},
		{
			name:     "not authorized returns 403",
			userID:   &senderID,
			urlParam: pID.String(),
			setupMocks: func(pr *mockProposalRepo) {
				pr.getByIDFn = func(_ context.Context, _ uuid.UUID) (*proposaldomain.Proposal, error) {
					return sampleProposal(senderID, recipientID), nil
				}
			},
			wantStatus: http.StatusForbidden,
			wantCode:   "not_authorized",
		},
		{
			name:       "invalid uuid returns 400",
			userID:     &recipientID,
			urlParam:   "not-valid",
			wantStatus: http.StatusBadRequest,
			wantCode:   "invalid_proposal_id",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pr := &mockProposalRepo{}
			if tc.setupMocks != nil {
				tc.setupMocks(pr)
			}
			h := newTestProposalHandler(&mockUserRepo{}, pr, &mockMessageSender{}, &mockNotificationSender{}, &mockPaymentProcessor{}, &mockStorageService{})

			req := httptest.NewRequest(http.MethodPost, "/api/v1/proposals/"+pID.String()+"/decline", nil)
			req = proposalCtx(req, tc.userID, tc.urlParam)
			rec := httptest.NewRecorder()

			h.DeclineProposal(rec, req)
			assert.Equal(t, tc.wantStatus, rec.Code)
			assertProposalErrorCode(t, rec, tc.wantCode)
		})
	}
}

// ---------------------------------------------------------------------------
// Tests — ListActiveProjects
// ---------------------------------------------------------------------------

func TestProposalHandler_ListActiveProjects(t *testing.T) {
	userID := uuid.New()

	tests := []struct {
		name       string
		userID     *uuid.UUID
		setupMocks func(*mockUserRepo, *mockProposalRepo)
		wantStatus int
		wantItems  int
	}{
		{
			name:   "success with results",
			userID: &userID,
			setupMocks: func(ur *mockUserRepo, pr *mockProposalRepo) {
				senderID := uuid.New()
				pr.listActiveProjectsFn = func(_ context.Context, _ uuid.UUID, _ string, _ int) ([]*proposaldomain.Proposal, string, error) {
					p := sampleProposal(senderID, userID)
					p.Status = proposaldomain.StatusActive
					return []*proposaldomain.Proposal{p}, "", nil
				}
				ur.getByIDFn = func(_ context.Context, _ uuid.UUID) (*user.User, error) {
					return &user.User{DisplayName: "Test"}, nil
				}
			},
			wantStatus: http.StatusOK,
			wantItems:  1,
		},
		{
			name:   "empty list",
			userID: &userID,
			setupMocks: func(_ *mockUserRepo, pr *mockProposalRepo) {
				pr.listActiveProjectsFn = func(_ context.Context, _ uuid.UUID, _ string, _ int) ([]*proposaldomain.Proposal, string, error) {
					return []*proposaldomain.Proposal{}, "", nil
				}
			},
			wantStatus: http.StatusOK,
			wantItems:  0,
		},
		{
			name:       "unauthenticated returns 401",
			userID:     nil,
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ur := &mockUserRepo{}
			pr := &mockProposalRepo{}
			if tc.setupMocks != nil {
				tc.setupMocks(ur, pr)
			}
			h := newTestProposalHandler(ur, pr, &mockMessageSender{}, &mockNotificationSender{}, &mockPaymentProcessor{}, &mockStorageService{})

			req := httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil)
			req = proposalCtx(req, tc.userID, "")
			rec := httptest.NewRecorder()

			h.ListActiveProjects(rec, req)
			assert.Equal(t, tc.wantStatus, rec.Code)
			if tc.wantStatus == http.StatusOK {
				var resp map[string]any
				require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
				items, ok := resp["data"].([]any)
				require.True(t, ok)
				assert.Len(t, items, tc.wantItems)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Shared test helpers
// ---------------------------------------------------------------------------

// userByIDLookup returns user records whose OrganizationID is set to
// the user's own id. That matches the convention used throughout the
// handler tests, where proposalCtx stores orgID := userID in the
// request context. It lets the new R14 org-directional checks in the
// proposal service (requireOrgIsSide) succeed when the test actor is
// on the correct side of the proposal.
func userByIDLookup(enterpriseID, providerID uuid.UUID) func(context.Context, uuid.UUID) (*user.User, error) {
	return func(_ context.Context, id uuid.UUID) (*user.User, error) {
		o := id // 1 user = its own org, matching proposalCtx convention
		switch id {
		case enterpriseID:
			u := enterpriseUser(enterpriseID)
			u.OrganizationID = &o
			return u, nil
		case providerID:
			u := providerUser(providerID)
			u.OrganizationID = &o
			return u, nil
		default:
			return &user.User{ID: id, DisplayName: "Unknown", OrganizationID: &o}, nil
		}
	}
}

func assertProposalErrorCode(t *testing.T, rec *httptest.ResponseRecorder, wantCode string) {
	t.Helper()
	if wantCode == "" {
		return
	}
	var resp map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, wantCode, resp["error"])
}
func (m *mockProposalRepo) CountAll(_ context.Context) (int, int, error) { return 0, 0, nil }
func (m *mockProposalRepo) SumPaidByClientOrganization(context.Context, uuid.UUID) (int64, error) {
	return 0, nil
}
func (m *mockProposalRepo) ListCompletedByClientOrganization(context.Context, uuid.UUID, int) ([]*proposaldomain.Proposal, error) {
	return nil, nil
}
