package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	proposalapp "marketplace-backend/internal/app/proposal"
	milestonedomain "marketplace-backend/internal/domain/milestone"
	proposaldomain "marketplace-backend/internal/domain/proposal"
	"marketplace-backend/internal/handler/middleware"
	"marketplace-backend/internal/port/repository"
)

// ---------------------------------------------------------------------------
// mockMilestoneRepo — handler-layer stub for repository.MilestoneRepository.
//
// The handler-level tests only need GetByIDForUpdate / Update / ListByProposal
// / GetCurrentActive to drive the milestone state machine, but the compile
// check forces us to stub the full interface. Methods the tests don't care
// about return zero values so they stay invisible unless a test actively
// overrides them.
// ---------------------------------------------------------------------------

type mockMilestoneRepo struct {
	listByProposalFn func(ctx context.Context, proposalID uuid.UUID) ([]*milestonedomain.Milestone, error)
}

func (m *mockMilestoneRepo) CreateBatch(_ context.Context, _ []*milestonedomain.Milestone) error {
	return nil
}

func (m *mockMilestoneRepo) GetByID(_ context.Context, _ uuid.UUID) (*milestonedomain.Milestone, error) {
	return nil, milestonedomain.ErrMilestoneNotFound
}

func (m *mockMilestoneRepo) GetByIDForUpdate(_ context.Context, _ uuid.UUID) (*milestonedomain.Milestone, error) {
	return nil, milestonedomain.ErrMilestoneNotFound
}

func (m *mockMilestoneRepo) ListByProposal(ctx context.Context, proposalID uuid.UUID) ([]*milestonedomain.Milestone, error) {
	if m.listByProposalFn != nil {
		return m.listByProposalFn(ctx, proposalID)
	}
	return nil, nil
}

func (m *mockMilestoneRepo) GetCurrentActive(_ context.Context, _ uuid.UUID) (*milestonedomain.Milestone, error) {
	return nil, milestonedomain.ErrMilestoneNotFound
}

func (m *mockMilestoneRepo) Update(_ context.Context, _ *milestonedomain.Milestone) error {
	return nil
}

func (m *mockMilestoneRepo) CreateDeliverable(_ context.Context, _ *milestonedomain.Deliverable) error {
	return nil
}

func (m *mockMilestoneRepo) ListDeliverables(_ context.Context, _ uuid.UUID) ([]*milestonedomain.Deliverable, error) {
	return nil, nil
}

func (m *mockMilestoneRepo) DeleteDeliverable(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (m *mockMilestoneRepo) ListByProposals(_ context.Context, _ []uuid.UUID) (map[uuid.UUID][]*milestonedomain.Milestone, error) {
	return nil, nil
}

// Compile-time check.
var _ repository.MilestoneRepository = (*mockMilestoneRepo)(nil)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// newTestProposalHandlerForMilestones builds a handler wired with a real
// proposal app service and a mock milestone repo. This is the milestone-
// aware variant of newTestProposalHandler — existing tests that don't
// care about milestones keep using the legacy helper.
func newTestProposalHandlerForMilestones(
	ur *mockUserRepo,
	pr *mockProposalRepo,
	mr *mockMilestoneRepo,
) *ProposalHandler {
	svc := proposalapp.NewService(proposalapp.ServiceDeps{
		Proposals:     pr,
		Users:         ur,
		Milestones:    mr,
		Messages:      &mockMessageSender{},
		Notifications: &mockNotificationSender{},
		Payments:      &mockPaymentProcessor{},
		Storage:       &mockStorageService{},
	})
	return NewProposalHandler(svc, nil)
}

// milestoneCtx extends proposalCtx with the `mid` URL param so the
// milestone handlers can read it via chi.URLParam(r, "mid").
func milestoneCtx(r *http.Request, userID *uuid.UUID, proposalID, milestoneID string) *http.Request {
	ctx := r.Context()
	rctx := chi.NewRouteContext()
	if proposalID != "" {
		rctx.URLParams.Add("id", proposalID)
	}
	if milestoneID != "" {
		rctx.URLParams.Add("mid", milestoneID)
	}
	ctx = context.WithValue(ctx, chi.RouteCtxKey, rctx)
	if userID != nil {
		ctx = context.WithValue(ctx, middleware.ContextKeyUserID, *userID)
		ctx = context.WithValue(ctx, middleware.ContextKeyOrganizationID, *userID)
	}
	return r.WithContext(ctx)
}

// singleFundedMilestone seeds a single funded milestone so
// validateMilestoneMatchesCurrent treats it as the current active one.
// The ID is whatever the caller supplies so the test can pass it to
// the URL as the `mid` param (happy case) or swap it for a different
// UUID (stale case).
func singleFundedMilestone(id, proposalID uuid.UUID) []*milestonedomain.Milestone {
	now := time.Now()
	return []*milestonedomain.Milestone{{
		ID:         id,
		ProposalID: proposalID,
		Sequence:   1,
		Title:      "Milestone 1",
		Amount:     500000,
		Status:     milestonedomain.StatusFunded,
		Version:    1,
		FundedAt:   &now,
	}}
}

// ---------------------------------------------------------------------------
// Shared milestone-endpoint test table.
//
// The four milestone transitions (fund / submit / approve / reject) share
// the exact same HTTP boundary behaviour: auth, URL parsing, milestone
// validation, error mapping. They only differ in the downstream service
// method they delegate to. We exercise all of the boundary concerns once
// per endpoint via this table so a future change to one handler can't
// silently diverge from the others.
// ---------------------------------------------------------------------------

type milestoneHandlerCase struct {
	name       string
	userID     *uuid.UUID
	proposalID string
	milestoneID string
	setupMocks func(*mockProposalRepo, *mockMilestoneRepo)
	wantStatus int
	wantCode   string
}

func runMilestoneHandlerTests(
	t *testing.T,
	invoke func(*ProposalHandler, http.ResponseWriter, *http.Request),
	cases []milestoneHandlerCase,
) {
	t.Helper()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ur := &mockUserRepo{}
			pr := &mockProposalRepo{}
			mr := &mockMilestoneRepo{}
			if tc.setupMocks != nil {
				tc.setupMocks(pr, mr)
			}
			h := newTestProposalHandlerForMilestones(ur, pr, mr)

			req := httptest.NewRequest(
				http.MethodPost,
				"/api/v1/proposals/"+tc.proposalID+"/milestones/"+tc.milestoneID+"/action",
				nil,
			)
			req = milestoneCtx(req, tc.userID, tc.proposalID, tc.milestoneID)
			rec := httptest.NewRecorder()

			invoke(h, rec, req)

			assert.Equal(t, tc.wantStatus, rec.Code)
			assertProposalErrorCode(t, rec, tc.wantCode)
		})
	}
}

// boundaryCases returns the HTTP-boundary cases every milestone endpoint
// must honour. They share an identical shape so the four tests below can
// reuse them without a per-handler copy. Callers still supply a happy-
// path case because the "success" expectation depends on which transition
// the handler performs (some cases intentionally omit it when the full
// service wiring is not justified at the handler level — the service-
// level suite already covers the successful transitions end-to-end).
func boundaryCases(proposalID, milestoneID uuid.UUID, actorID uuid.UUID) []milestoneHandlerCase {
	return []milestoneHandlerCase{
		{
			name:        "unauthenticated returns 401",
			userID:      nil,
			proposalID:  proposalID.String(),
			milestoneID: milestoneID.String(),
			wantStatus:  http.StatusUnauthorized,
			wantCode:    "unauthorized",
		},
		{
			name:        "invalid proposal id returns 400",
			userID:      &actorID,
			proposalID:  "not-a-uuid",
			milestoneID: milestoneID.String(),
			wantStatus:  http.StatusBadRequest,
			wantCode:    "invalid_id",
		},
		{
			name:        "invalid milestone id returns 400",
			userID:      &actorID,
			proposalID:  proposalID.String(),
			milestoneID: "not-a-uuid",
			wantStatus:  http.StatusBadRequest,
			wantCode:    "invalid_milestone_id",
		},
		{
			name:        "no active milestone returns 409",
			userID:      &actorID,
			proposalID:  proposalID.String(),
			milestoneID: milestoneID.String(),
			setupMocks: func(_ *mockProposalRepo, mr *mockMilestoneRepo) {
				mr.listByProposalFn = func(_ context.Context, _ uuid.UUID) ([]*milestonedomain.Milestone, error) {
					return nil, nil
				}
			},
			wantStatus: http.StatusConflict,
			wantCode:   "no_active_milestone",
		},
		{
			name:        "stale milestone id returns 409",
			userID:      &actorID,
			proposalID:  proposalID.String(),
			milestoneID: milestoneID.String(),
			setupMocks: func(_ *mockProposalRepo, mr *mockMilestoneRepo) {
				// Return a milestone whose ID differs from the one in the
				// URL → validateMilestoneMatchesCurrent must 409.
				mr.listByProposalFn = func(_ context.Context, pID uuid.UUID) ([]*milestonedomain.Milestone, error) {
					return singleFundedMilestone(uuid.New(), pID), nil
				}
			},
			wantStatus: http.StatusConflict,
			wantCode:   "stale_milestone",
		},
	}
}

// ---------------------------------------------------------------------------
// FundMilestone
// ---------------------------------------------------------------------------

func TestProposalHandler_FundMilestone(t *testing.T) {
	proposalID := uuid.New()
	milestoneID := uuid.New()
	clientID := uuid.New()

	runMilestoneHandlerTests(
		t,
		func(h *ProposalHandler, w http.ResponseWriter, r *http.Request) { h.FundMilestone(w, r) },
		boundaryCases(proposalID, milestoneID, clientID),
	)
}

// ---------------------------------------------------------------------------
// SubmitMilestone
// ---------------------------------------------------------------------------

func TestProposalHandler_SubmitMilestone(t *testing.T) {
	proposalID := uuid.New()
	milestoneID := uuid.New()
	providerID := uuid.New()

	runMilestoneHandlerTests(
		t,
		func(h *ProposalHandler, w http.ResponseWriter, r *http.Request) { h.SubmitMilestone(w, r) },
		boundaryCases(proposalID, milestoneID, providerID),
	)
}

// ---------------------------------------------------------------------------
// ApproveMilestone
// ---------------------------------------------------------------------------

func TestProposalHandler_ApproveMilestone(t *testing.T) {
	proposalID := uuid.New()
	milestoneID := uuid.New()
	clientID := uuid.New()

	runMilestoneHandlerTests(
		t,
		func(h *ProposalHandler, w http.ResponseWriter, r *http.Request) { h.ApproveMilestone(w, r) },
		boundaryCases(proposalID, milestoneID, clientID),
	)
}

// ---------------------------------------------------------------------------
// RejectMilestone
// ---------------------------------------------------------------------------

func TestProposalHandler_RejectMilestone(t *testing.T) {
	proposalID := uuid.New()
	milestoneID := uuid.New()
	clientID := uuid.New()

	runMilestoneHandlerTests(
		t,
		func(h *ProposalHandler, w http.ResponseWriter, r *http.Request) { h.RejectMilestone(w, r) },
		boundaryCases(proposalID, milestoneID, clientID),
	)
}

// ---------------------------------------------------------------------------
// CancelProposal — no milestone id in the URL, so its boundary checks are
// a different (smaller) set.
// ---------------------------------------------------------------------------

func TestProposalHandler_CancelProposal(t *testing.T) {
	proposalID := uuid.New()
	userID := uuid.New()

	tests := []struct {
		name       string
		userID     *uuid.UUID
		urlParam   string
		setupMocks func(*mockProposalRepo, *mockMilestoneRepo)
		wantStatus int
		wantCode   string
	}{
		{
			name:       "unauthenticated returns 401",
			userID:     nil,
			urlParam:   proposalID.String(),
			wantStatus: http.StatusUnauthorized,
			wantCode:   "unauthorized",
		},
		{
			name:       "invalid uuid returns 400",
			userID:     &userID,
			urlParam:   "not-a-uuid",
			wantStatus: http.StatusBadRequest,
			wantCode:   "invalid_id",
		},
		{
			name:     "proposal not found returns 404",
			userID:   &userID,
			urlParam: proposalID.String(),
			setupMocks: func(pr *mockProposalRepo, _ *mockMilestoneRepo) {
				pr.getByIDFn = func(_ context.Context, _ uuid.UUID) (*proposaldomain.Proposal, error) {
					return nil, proposaldomain.ErrProposalNotFound
				}
			},
			wantStatus: http.StatusNotFound,
			wantCode:   "proposal_not_found",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ur := &mockUserRepo{}
			pr := &mockProposalRepo{}
			mr := &mockMilestoneRepo{}
			if tc.setupMocks != nil {
				tc.setupMocks(pr, mr)
			}
			h := newTestProposalHandlerForMilestones(ur, pr, mr)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/proposals/"+tc.urlParam+"/cancel", nil)
			// CancelProposal only reads {id}, no {mid}.
			req = milestoneCtx(req, tc.userID, tc.urlParam, "")
			rec := httptest.NewRecorder()

			h.CancelProposal(rec, req)

			assert.Equal(t, tc.wantStatus, rec.Code)
			assertProposalErrorCode(t, rec, tc.wantCode)
		})
	}
}
