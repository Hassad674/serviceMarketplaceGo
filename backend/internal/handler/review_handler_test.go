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

	reviewapp "marketplace-backend/internal/app/review"
	milestonedomain "marketplace-backend/internal/domain/milestone"
	"marketplace-backend/internal/domain/proposal"
	reviewdomain "marketplace-backend/internal/domain/review"
	"marketplace-backend/internal/handler/middleware"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
)

// ---------------------------------------------------------------------------
// Mocks specific to review tests
// ---------------------------------------------------------------------------

type mockReviewRepo struct {
	createFn           func(ctx context.Context, r *reviewdomain.Review) error
	createAndRevealFn  func(ctx context.Context, r *reviewdomain.Review) (*reviewdomain.Review, error)
	getByIDFn          func(ctx context.Context, id uuid.UUID) (*reviewdomain.Review, error)
	listByReviewedFn   func(ctx context.Context, userID uuid.UUID, cursor string, limit int) ([]*reviewdomain.Review, string, error)
	getAverageRatingFn func(ctx context.Context, userID uuid.UUID) (*reviewdomain.AverageRating, error)
	hasReviewedFn      func(ctx context.Context, proposalID, reviewerID uuid.UUID) (bool, error)
}

func (m *mockReviewRepo) Create(ctx context.Context, r *reviewdomain.Review) error {
	if m.createFn != nil {
		return m.createFn(ctx, r)
	}
	return nil
}

func (m *mockReviewRepo) CreateAndMaybeReveal(ctx context.Context, r *reviewdomain.Review) (*reviewdomain.Review, error) {
	if m.createAndRevealFn != nil {
		return m.createAndRevealFn(ctx, r)
	}
	return r, nil
}

func (m *mockReviewRepo) GetByID(ctx context.Context, id uuid.UUID) (*reviewdomain.Review, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, reviewdomain.ErrNotFound
}

func (m *mockReviewRepo) GetByIDForOrg(ctx context.Context, id, _ uuid.UUID) (*reviewdomain.Review, error) {
	return m.GetByID(ctx, id)
}

func (m *mockReviewRepo) ListByReviewedOrganization(ctx context.Context, userID uuid.UUID, cursor string, limit int) ([]*reviewdomain.Review, string, error) {
	if m.listByReviewedFn != nil {
		return m.listByReviewedFn(ctx, userID, cursor, limit)
	}
	return []*reviewdomain.Review{}, "", nil
}

func (m *mockReviewRepo) GetAverageRatingByOrganization(ctx context.Context, userID uuid.UUID) (*reviewdomain.AverageRating, error) {
	if m.getAverageRatingFn != nil {
		return m.getAverageRatingFn(ctx, userID)
	}
	return &reviewdomain.AverageRating{Average: 0, Count: 0}, nil
}

func (m *mockReviewRepo) ListClientReviewsByOrganization(_ context.Context, _ uuid.UUID, _ int) ([]*reviewdomain.Review, error) {
	return nil, nil
}

func (m *mockReviewRepo) GetClientAverageRating(_ context.Context, _ uuid.UUID) (*reviewdomain.AverageRating, error) {
	return &reviewdomain.AverageRating{Average: 0, Count: 0}, nil
}

func (m *mockReviewRepo) HasReviewed(ctx context.Context, proposalID, reviewerID uuid.UUID) (bool, error) {
	if m.hasReviewedFn != nil {
		return m.hasReviewedFn(ctx, proposalID, reviewerID)
	}
	return false, nil
}

func (m *mockReviewRepo) ListAdmin(_ context.Context, _ repository.AdminReviewFilters) ([]repository.AdminReview, error) {
	return nil, nil
}

func (m *mockReviewRepo) CountAdmin(_ context.Context, _ repository.AdminReviewFilters) (int, error) {
	return 0, nil
}

func (m *mockReviewRepo) GetAdminByID(_ context.Context, _ uuid.UUID) (*repository.AdminReview, error) {
	return nil, reviewdomain.ErrNotFound
}

func (m *mockReviewRepo) DeleteAdmin(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (m *mockReviewRepo) UpdateReviewModeration(_ context.Context, _ uuid.UUID, _ string, _ float64, _ []byte) error {
	return nil
}

func (m *mockReviewRepo) GetByProposalIDs(_ context.Context, _ []uuid.UUID, _ string) (map[uuid.UUID]*reviewdomain.Review, error) {
	return map[uuid.UUID]*reviewdomain.Review{}, nil
}

// Compile-time check.
var _ repository.ReviewRepository = (*mockReviewRepo)(nil)

type mockProposalRepo struct {
	createFn              func(ctx context.Context, p *proposal.Proposal) error
	createWithDocsFn      func(ctx context.Context, p *proposal.Proposal, docs []*proposal.ProposalDocument) error
	getByIDFn             func(ctx context.Context, id uuid.UUID) (*proposal.Proposal, error)
	updateFn              func(ctx context.Context, p *proposal.Proposal) error
	getLatestVersionFn    func(ctx context.Context, rootID uuid.UUID) (*proposal.Proposal, error)
	listByConversationFn  func(ctx context.Context, convID uuid.UUID) ([]*proposal.Proposal, error)
	listActiveProjectsFn  func(ctx context.Context, userID uuid.UUID, cursor string, limit int) ([]*proposal.Proposal, string, error)
	getDocumentsFn        func(ctx context.Context, proposalID uuid.UUID) ([]*proposal.ProposalDocument, error)
	createDocumentFn      func(ctx context.Context, doc *proposal.ProposalDocument) error
	isOrgAuthorizedFn     func(ctx context.Context, proposalID, orgID uuid.UUID) (bool, error)
}

func (m *mockProposalRepo) Create(ctx context.Context, p *proposal.Proposal) error {
	if m.createFn != nil {
		return m.createFn(ctx, p)
	}
	return nil
}

func (m *mockProposalRepo) CreateWithDocuments(ctx context.Context, p *proposal.Proposal, docs []*proposal.ProposalDocument) error {
	if m.createWithDocsFn != nil {
		return m.createWithDocsFn(ctx, p, docs)
	}
	return nil
}

// CreateWithDocumentsAndMilestones is the phase-4 atomic insert path.
// The handler tests don't exercise the milestone side, so this stub
// just delegates to CreateWithDocuments and ignores the milestone slice.
func (m *mockProposalRepo) CreateWithDocumentsAndMilestones(ctx context.Context, p *proposal.Proposal, docs []*proposal.ProposalDocument, _ []*milestonedomain.Milestone) error {
	return m.CreateWithDocuments(ctx, p, docs)
}

func (m *mockProposalRepo) GetByID(ctx context.Context, id uuid.UUID) (*proposal.Proposal, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *mockProposalRepo) GetByIDForOrg(ctx context.Context, id, _ uuid.UUID) (*proposal.Proposal, error) {
	return m.GetByID(ctx, id)
}

func (m *mockProposalRepo) GetByIDs(context.Context, []uuid.UUID) ([]*proposal.Proposal, error) {
	return nil, nil
}

func (m *mockProposalRepo) Update(ctx context.Context, p *proposal.Proposal) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, p)
	}
	return nil
}

func (m *mockProposalRepo) GetLatestVersion(ctx context.Context, rootID uuid.UUID) (*proposal.Proposal, error) {
	if m.getLatestVersionFn != nil {
		return m.getLatestVersionFn(ctx, rootID)
	}
	return nil, nil
}

func (m *mockProposalRepo) ListByConversation(ctx context.Context, convID uuid.UUID) ([]*proposal.Proposal, error) {
	if m.listByConversationFn != nil {
		return m.listByConversationFn(ctx, convID)
	}
	return nil, nil
}

func (m *mockProposalRepo) ListActiveProjectsByOrganization(ctx context.Context, orgID uuid.UUID, cursor string, limit int) ([]*proposal.Proposal, string, error) {
	if m.listActiveProjectsFn != nil {
		return m.listActiveProjectsFn(ctx, orgID, cursor, limit)
	}
	return nil, "", nil
}

func (m *mockProposalRepo) ListCompletedByOrganization(_ context.Context, _ uuid.UUID, _ string, _ int) ([]*proposal.Proposal, string, error) {
	return nil, "", nil
}

func (m *mockProposalRepo) GetDocuments(ctx context.Context, proposalID uuid.UUID) ([]*proposal.ProposalDocument, error) {
	if m.getDocumentsFn != nil {
		return m.getDocumentsFn(ctx, proposalID)
	}
	return nil, nil
}

func (m *mockProposalRepo) CreateDocument(ctx context.Context, doc *proposal.ProposalDocument) error {
	if m.createDocumentFn != nil {
		return m.createDocumentFn(ctx, doc)
	}
	return nil
}

// IsOrgAuthorizedForProposal: the handler-layer mock defaults to
// allow (true) because most existing tests passing a userID = orgID
// expect the new org check to succeed trivially. Tests that want to
// verify a denial set isOrgAuthorizedFn explicitly.
func (m *mockProposalRepo) IsOrgAuthorizedForProposal(ctx context.Context, proposalID, orgID uuid.UUID) (bool, error) {
	if m.isOrgAuthorizedFn != nil {
		return m.isOrgAuthorizedFn(ctx, proposalID, orgID)
	}
	return true, nil
}

// Compile-time check.
var _ repository.ProposalRepository = (*mockProposalRepo)(nil)

type mockNotificationSender struct {
	sendFn func(ctx context.Context, input service.NotificationInput) error
}

func (m *mockNotificationSender) Send(ctx context.Context, input service.NotificationInput) error {
	if m.sendFn != nil {
		return m.sendFn(ctx, input)
	}
	return nil
}

// Compile-time check.
var _ service.NotificationSender = (*mockNotificationSender)(nil)

// ---------------------------------------------------------------------------
// Helper
// ---------------------------------------------------------------------------

func newTestReviewHandler(
	rr *mockReviewRepo,
	pr *mockProposalRepo,
	ns *mockNotificationSender,
) *ReviewHandler {
	svc := reviewapp.NewService(reviewapp.ServiceDeps{
		Reviews:       rr,
		Proposals:     pr,
		Users:         &mockUserRepo{},
		Notifications: ns,
	})
	return NewReviewHandler(svc)
}

func testProposal(clientID, providerID uuid.UUID) *proposal.Proposal {
	now := time.Now()
	// CompletedAt must be set so the CreateReview service's 14-day
	// window check sees a recent completion and lets the review through.
	completedAt := now
	return &proposal.Proposal{
		ID:          uuid.New(),
		ClientID:    clientID,
		ProviderID:  providerID,
		Status:      proposal.StatusCompleted,
		CreatedAt:   now,
		UpdatedAt:   now,
		CompletedAt: &completedAt,
	}
}

func testReview(reviewerID, reviewedID, proposalID uuid.UUID) *reviewdomain.Review {
	now := time.Now()
	return &reviewdomain.Review{
		ID:           uuid.New(),
		ProposalID:   proposalID,
		ReviewerID:   reviewerID,
		ReviewedID:   reviewedID,
		GlobalRating: 4,
		Comment:      "Great work",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestReviewHandler_CreateReview(t *testing.T) {
	clientID := uuid.New()
	providerID := uuid.New()
	proposalID := uuid.New()

	tests := []struct {
		name       string
		userID     *uuid.UUID
		body       map[string]any
		setupMocks func(*mockReviewRepo, *mockProposalRepo)
		wantStatus int
		wantCode   string
	}{
		{
			name:   "success",
			userID: &clientID,
			body: map[string]any{
				"proposal_id":  proposalID.String(),
				"global_rating": 5,
				"comment":       "Excellent",
			},
			setupMocks: func(rr *mockReviewRepo, pr *mockProposalRepo) {
				pr.getByIDFn = func(_ context.Context, _ uuid.UUID) (*proposal.Proposal, error) {
					p := testProposal(clientID, providerID)
					p.ID = proposalID
					return p, nil
				}
			},
			wantStatus: http.StatusCreated,
		},
		{
			name:   "invalid rating returns bad request",
			userID: &clientID,
			body: map[string]any{
				"proposal_id":  proposalID.String(),
				"global_rating": 10,
				"comment":       "Too high",
			},
			setupMocks: func(rr *mockReviewRepo, pr *mockProposalRepo) {
				pr.getByIDFn = func(_ context.Context, _ uuid.UUID) (*proposal.Proposal, error) {
					p := testProposal(clientID, providerID)
					p.ID = proposalID
					return p, nil
				}
			},
			wantStatus: http.StatusBadRequest,
			wantCode:   "invalid_rating",
		},
		{
			name:       "unauthenticated returns 401",
			userID:     nil,
			body:       map[string]any{"proposal_id": proposalID.String(), "global_rating": 4},
			wantStatus: http.StatusUnauthorized,
			wantCode:   "unauthorized",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rr := &mockReviewRepo{}
			pr := &mockProposalRepo{}
			if tc.setupMocks != nil {
				tc.setupMocks(rr, pr)
			}
			h := newTestReviewHandler(rr, pr, &mockNotificationSender{})

			body, _ := json.Marshal(tc.body)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/reviews", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			if tc.userID != nil {
				ctx := context.WithValue(req.Context(), middleware.ContextKeyUserID, *tc.userID)
				req = req.WithContext(ctx)
			}
			rec := httptest.NewRecorder()

			h.CreateReview(rec, req)
			assert.Equal(t, tc.wantStatus, rec.Code)
			if tc.wantCode != "" {
				var resp map[string]any
				require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
				assert.Equal(t, tc.wantCode, resp["error"])
			}
		})
	}
}

func TestReviewHandler_ListByUser(t *testing.T) {
	userID := uuid.New()
	providerID := uuid.New()
	proposalID := uuid.New()

	tests := []struct {
		name       string
		urlParam   string
		setupMock  func(*mockReviewRepo)
		wantStatus int
		wantItems  int
	}{
		{
			name:     "success with results",
			urlParam: userID.String(),
			setupMock: func(rr *mockReviewRepo) {
				rr.listByReviewedFn = func(_ context.Context, _ uuid.UUID, _ string, _ int) ([]*reviewdomain.Review, string, error) {
					return []*reviewdomain.Review{
						testReview(userID, providerID, proposalID),
					}, "next_abc", nil
				}
			},
			wantStatus: http.StatusOK,
			wantItems:  1,
		},
		{
			name:       "empty list",
			urlParam:   userID.String(),
			wantStatus: http.StatusOK,
			wantItems:  0,
		},
		{
			name:       "invalid uuid",
			urlParam:   "not-a-uuid",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rr := &mockReviewRepo{}
			if tc.setupMock != nil {
				tc.setupMock(rr)
			}
			h := newTestReviewHandler(rr, &mockProposalRepo{}, &mockNotificationSender{})

			req := httptest.NewRequest(http.MethodGet, "/api/v1/reviews/user/"+tc.urlParam, nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("orgId", tc.urlParam)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
			rec := httptest.NewRecorder()

			h.ListByOrganization(rec, req)
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

func TestReviewHandler_GetAverageRatingByOrganization(t *testing.T) {
	userID := uuid.New()

	tests := []struct {
		name       string
		urlParam   string
		setupMock  func(*mockReviewRepo)
		wantStatus int
		wantAvg    float64
	}{
		{
			name:     "success",
			urlParam: userID.String(),
			setupMock: func(rr *mockReviewRepo) {
				rr.getAverageRatingFn = func(_ context.Context, _ uuid.UUID) (*reviewdomain.AverageRating, error) {
					return &reviewdomain.AverageRating{Average: 4.5, Count: 10}, nil
				}
			},
			wantStatus: http.StatusOK,
			wantAvg:    4.5,
		},
		{
			name:       "invalid uuid",
			urlParam:   "bad-id",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rr := &mockReviewRepo{}
			if tc.setupMock != nil {
				tc.setupMock(rr)
			}
			h := newTestReviewHandler(rr, &mockProposalRepo{}, &mockNotificationSender{})

			req := httptest.NewRequest(http.MethodGet, "/api/v1/reviews/average/"+tc.urlParam, nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("orgId", tc.urlParam)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
			rec := httptest.NewRecorder()

			h.GetAverageRating(rec, req)
			assert.Equal(t, tc.wantStatus, rec.Code)
			if tc.wantStatus == http.StatusOK {
				var resp map[string]any
				require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
				data := resp["data"].(map[string]any)
				assert.Equal(t, tc.wantAvg, data["average"])
			}
		})
	}
}

func TestReviewHandler_CanReview(t *testing.T) {
	clientID := uuid.New()
	providerID := uuid.New()
	proposalID := uuid.New()

	tests := []struct {
		name       string
		userID     *uuid.UUID
		urlParam   string
		setupMocks func(*mockReviewRepo, *mockProposalRepo)
		wantStatus int
		wantCan    *bool
	}{
		{
			name:     "success can review",
			userID:   &clientID,
			urlParam: proposalID.String(),
			setupMocks: func(rr *mockReviewRepo, pr *mockProposalRepo) {
				pr.getByIDFn = func(_ context.Context, _ uuid.UUID) (*proposal.Proposal, error) {
					p := testProposal(clientID, providerID)
					p.ID = proposalID
					return p, nil
				}
			},
			wantStatus: http.StatusOK,
			wantCan:    boolPtr(true),
		},
		{
			name:       "unauthenticated returns 401",
			userID:     nil,
			urlParam:   proposalID.String(),
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rr := &mockReviewRepo{}
			pr := &mockProposalRepo{}
			if tc.setupMocks != nil {
				tc.setupMocks(rr, pr)
			}
			h := newTestReviewHandler(rr, pr, &mockNotificationSender{})

			req := httptest.NewRequest(http.MethodGet, "/api/v1/reviews/can-review/"+tc.urlParam, nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("proposalId", tc.urlParam)
			ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
			if tc.userID != nil {
				ctx = context.WithValue(ctx, middleware.ContextKeyUserID, *tc.userID)
			}
			req = req.WithContext(ctx)
			rec := httptest.NewRecorder()

			h.CanReview(rec, req)
			assert.Equal(t, tc.wantStatus, rec.Code)
			if tc.wantCan != nil {
				var resp map[string]any
				require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
				data := resp["data"].(map[string]any)
				assert.Equal(t, *tc.wantCan, data["can_review"])
			}
		})
	}
}

func boolPtr(v bool) *bool { return &v }
