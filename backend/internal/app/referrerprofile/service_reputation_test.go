package referrerprofile_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appreferrer "marketplace-backend/internal/app/referrerprofile"
	milestonedomain "marketplace-backend/internal/domain/milestone"
	proposaldomain "marketplace-backend/internal/domain/proposal"
	referraldomain "marketplace-backend/internal/domain/referral"
	reviewdomain "marketplace-backend/internal/domain/review"
	userdomain "marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/port/repository"
)

// ─── Fakes ──────────────────────────────────────────────────────────────

type callCounters struct {
	listReferrals           atomic.Int32
	listAttributionsByIDs   atomic.Int32
	getProposalsByIDs       atomic.Int32
	getUsersByIDs           atomic.Int32
	getReviewsByProposalIDs atomic.Int32
}

type fakeReferralRepo struct {
	counters  *callCounters
	referrals []*referraldomain.Referral
	attribs   []*referraldomain.Attribution
}

func (f *fakeReferralRepo) GetByID(context.Context, uuid.UUID) (*referraldomain.Referral, error) {
	return nil, nil
}
func (f *fakeReferralRepo) FindActiveByCouple(context.Context, uuid.UUID, uuid.UUID) (*referraldomain.Referral, error) {
	return nil, nil
}
func (f *fakeReferralRepo) ListByReferrer(_ context.Context, referrerID uuid.UUID, _ repository.ReferralListFilter) ([]*referraldomain.Referral, string, error) {
	if f.counters != nil {
		f.counters.listReferrals.Add(1)
	}
	var out []*referraldomain.Referral
	for _, r := range f.referrals {
		if r.ReferrerID == referrerID {
			out = append(out, r)
		}
	}
	return out, "", nil
}
func (f *fakeReferralRepo) ListIncomingForProvider(context.Context, uuid.UUID, repository.ReferralListFilter) ([]*referraldomain.Referral, string, error) {
	return nil, "", nil
}
func (f *fakeReferralRepo) ListIncomingForClient(context.Context, uuid.UUID, repository.ReferralListFilter) ([]*referraldomain.Referral, string, error) {
	return nil, "", nil
}
func (f *fakeReferralRepo) ListNegotiations(context.Context, uuid.UUID) ([]*referraldomain.Negotiation, error) {
	return nil, nil
}
func (f *fakeReferralRepo) CreateAttribution(context.Context, *referraldomain.Attribution) error {
	return nil
}
func (f *fakeReferralRepo) FindAttributionByProposal(context.Context, uuid.UUID) (*referraldomain.Attribution, error) {
	return nil, nil
}
func (f *fakeReferralRepo) FindAttributionByID(context.Context, uuid.UUID) (*referraldomain.Attribution, error) {
	return nil, nil
}
func (f *fakeReferralRepo) ListAttributionsByReferral(context.Context, uuid.UUID) ([]*referraldomain.Attribution, error) {
	return nil, nil
}
func (f *fakeReferralRepo) ListAttributionsByReferralIDs(_ context.Context, ids []uuid.UUID) ([]*referraldomain.Attribution, error) {
	if f.counters != nil {
		f.counters.listAttributionsByIDs.Add(1)
	}
	want := make(map[uuid.UUID]struct{}, len(ids))
	for _, id := range ids {
		want[id] = struct{}{}
	}
	var out []*referraldomain.Attribution
	for _, a := range f.attribs {
		if _, ok := want[a.ReferralID]; ok {
			out = append(out, a)
		}
	}
	return out, nil
}
func (f *fakeReferralRepo) ListExpiringIntros(context.Context, time.Time, int) ([]*referraldomain.Referral, error) {
	return nil, nil
}
func (f *fakeReferralRepo) ListExpiringActives(context.Context, time.Time, int) ([]*referraldomain.Referral, error) {
	return nil, nil
}
func (f *fakeReferralRepo) CountByReferrer(context.Context, uuid.UUID) (map[referraldomain.Status]int, error) {
	return nil, nil
}
func (f *fakeReferralRepo) SumCommissionsByReferrer(context.Context, uuid.UUID) (map[referraldomain.CommissionStatus]int64, error) {
	return nil, nil
}

// fakeReferralRepo implements the narrowed reputation surface
// (ReferralReader + ReferralAttributionStore). The legacy 24-method
// stub was shrunk to 15 — every dropped method belonged to
// ReferralWriter or ReferralCommissionStore, neither of which is
// consumed by the reputation aggregate.
var (
	_ repository.ReferralReader           = (*fakeReferralRepo)(nil)
	_ repository.ReferralAttributionStore = (*fakeReferralRepo)(nil)
)

type fakeProposalRepo struct {
	counters  *callCounters
	proposals []*proposaldomain.Proposal
}

func (f *fakeProposalRepo) Create(context.Context, *proposaldomain.Proposal) error { return nil }
func (f *fakeProposalRepo) CreateWithDocuments(context.Context, *proposaldomain.Proposal, []*proposaldomain.ProposalDocument) error {
	return nil
}
func (f *fakeProposalRepo) CreateWithDocumentsAndMilestones(context.Context, *proposaldomain.Proposal, []*proposaldomain.ProposalDocument, []*milestonedomain.Milestone) error {
	return nil
}
func (f *fakeProposalRepo) GetByID(context.Context, uuid.UUID) (*proposaldomain.Proposal, error) {
	return nil, nil
}
func (f *fakeProposalRepo) GetByIDForOrg(context.Context, uuid.UUID, uuid.UUID) (*proposaldomain.Proposal, error) {
	return nil, nil
}
func (f *fakeProposalRepo) GetByIDs(_ context.Context, ids []uuid.UUID) ([]*proposaldomain.Proposal, error) {
	if f.counters != nil {
		f.counters.getProposalsByIDs.Add(1)
	}
	want := make(map[uuid.UUID]struct{}, len(ids))
	for _, id := range ids {
		want[id] = struct{}{}
	}
	out := make([]*proposaldomain.Proposal, 0, len(ids))
	for _, p := range f.proposals {
		if _, ok := want[p.ID]; ok {
			out = append(out, p)
		}
	}
	return out, nil
}
func (f *fakeProposalRepo) Update(context.Context, *proposaldomain.Proposal) error { return nil }
func (f *fakeProposalRepo) GetLatestVersion(context.Context, uuid.UUID) (*proposaldomain.Proposal, error) {
	return nil, nil
}
func (f *fakeProposalRepo) ListByConversation(context.Context, uuid.UUID) ([]*proposaldomain.Proposal, error) {
	return nil, nil
}
func (f *fakeProposalRepo) ListActiveProjectsByOrganization(context.Context, uuid.UUID, string, int) ([]*proposaldomain.Proposal, string, error) {
	return nil, "", nil
}
func (f *fakeProposalRepo) ListCompletedByOrganization(context.Context, uuid.UUID, string, int) ([]*proposaldomain.Proposal, string, error) {
	return nil, "", nil
}
func (f *fakeProposalRepo) GetDocuments(context.Context, uuid.UUID) ([]*proposaldomain.ProposalDocument, error) {
	return nil, nil
}
func (f *fakeProposalRepo) CreateDocument(context.Context, *proposaldomain.ProposalDocument) error {
	return nil
}
func (f *fakeProposalRepo) IsOrgAuthorizedForProposal(context.Context, uuid.UUID, uuid.UUID) (bool, error) {
	return true, nil
}
func (f *fakeProposalRepo) CountAll(context.Context) (int, int, error) { return 0, 0, nil }
func (f *fakeProposalRepo) SumPaidByClientOrganization(context.Context, uuid.UUID) (int64, error) {
	return 0, nil
}
func (f *fakeProposalRepo) ListCompletedByClientOrganization(context.Context, uuid.UUID, int) ([]*proposaldomain.Proposal, error) {
	return nil, nil
}

var _ repository.ProposalRepository = (*fakeProposalRepo)(nil)

// fakeReviewRepo keeps an explicit slice of reviews (NOT a map keyed
// by proposal_id) because proposals carry two rows in the double-blind
// model — one per side. Keying by proposal_id would collide and silently
// drop one side, which is exactly the regression the tests must guard
// against.
type fakeReviewRepo struct {
	counters *callCounters
	reviews  []*reviewdomain.Review
}

func (f *fakeReviewRepo) Create(context.Context, *reviewdomain.Review) error { return nil }
func (f *fakeReviewRepo) CreateAndMaybeReveal(_ context.Context, r *reviewdomain.Review) (*reviewdomain.Review, error) {
	return r, nil
}
func (f *fakeReviewRepo) GetByID(context.Context, uuid.UUID) (*reviewdomain.Review, error) {
	return nil, nil
}
func (f *fakeReviewRepo) GetByIDForOrg(context.Context, uuid.UUID, uuid.UUID) (*reviewdomain.Review, error) {
	return nil, nil
}
func (f *fakeReviewRepo) ListByReviewedOrganization(context.Context, uuid.UUID, string, int) ([]*reviewdomain.Review, string, error) {
	return nil, "", nil
}
func (f *fakeReviewRepo) GetAverageRatingByOrganization(context.Context, uuid.UUID) (*reviewdomain.AverageRating, error) {
	return &reviewdomain.AverageRating{}, nil
}
func (f *fakeReviewRepo) ListClientReviewsByOrganization(context.Context, uuid.UUID, int) ([]*reviewdomain.Review, error) {
	return nil, nil
}
func (f *fakeReviewRepo) GetClientAverageRating(context.Context, uuid.UUID) (*reviewdomain.AverageRating, error) {
	return &reviewdomain.AverageRating{}, nil
}
func (f *fakeReviewRepo) HasReviewed(context.Context, uuid.UUID, uuid.UUID) (bool, error) {
	return false, nil
}
func (f *fakeReviewRepo) GetByProposalIDs(_ context.Context, ids []uuid.UUID, side string) (map[uuid.UUID]*reviewdomain.Review, error) {
	if f.counters != nil {
		f.counters.getReviewsByProposalIDs.Add(1)
	}
	want := make(map[uuid.UUID]struct{}, len(ids))
	for _, id := range ids {
		want[id] = struct{}{}
	}
	out := make(map[uuid.UUID]*reviewdomain.Review, len(ids))
	for _, rv := range f.reviews {
		if rv == nil {
			continue
		}
		if _, ok := want[rv.ProposalID]; !ok {
			continue
		}
		// Honour the SQL-level side filter in the fake so callers that
		// set the wrong side (or no side at all) are caught by tests.
		if side != "" && rv.Side != side {
			continue
		}
		// Stable last-write-wins for callers that pass side="" on a
		// proposal with both sides — this intentionally reproduces the
		// pre-fix non-deterministic behaviour so regressions surface.
		out[rv.ProposalID] = rv
	}
	return out, nil
}
func (f *fakeReviewRepo) UpdateReviewModeration(context.Context, uuid.UUID, string, float64, []byte) error {
	return nil
}
func (f *fakeReviewRepo) ListAdmin(context.Context, repository.AdminReviewFilters) ([]repository.AdminReview, error) {
	return nil, nil
}
func (f *fakeReviewRepo) CountAdmin(context.Context, repository.AdminReviewFilters) (int, error) {
	return 0, nil
}
func (f *fakeReviewRepo) GetAdminByID(context.Context, uuid.UUID) (*repository.AdminReview, error) {
	return nil, nil
}
func (f *fakeReviewRepo) DeleteAdmin(context.Context, uuid.UUID) error { return nil }

var _ repository.ReviewRepository = (*fakeReviewRepo)(nil)

type fakeUserBatchReader struct {
	counters *callCounters
	users    map[uuid.UUID]*userdomain.User
}

func (f *fakeUserBatchReader) GetByIDs(_ context.Context, ids []uuid.UUID) ([]*userdomain.User, error) {
	if f.counters != nil {
		f.counters.getUsersByIDs.Add(1)
	}
	out := make([]*userdomain.User, 0, len(ids))
	for _, id := range ids {
		if u, ok := f.users[id]; ok {
			out = append(out, u)
		}
	}
	return out, nil
}

var _ repository.UserBatchReader = (*fakeUserBatchReader)(nil)

// ─── Test helpers ────────────────────────────────────────────────────────

type setupInput struct {
	referrerID uuid.UUID
	referrals  []*referraldomain.Referral
	attribs    []*referraldomain.Attribution
	proposals  []*proposaldomain.Proposal
	// reviews is flattened by proposal_id → single review; convenient for
	// the majority of tests that only need one side per proposal. Tests
	// exercising the double-blind collision use reviewList instead.
	reviews     map[uuid.UUID]*reviewdomain.Review
	reviewList  []*reviewdomain.Review
	users       map[uuid.UUID]*userdomain.User
}

func newServiceForReputation(t *testing.T, in setupInput) (*appreferrer.Service, *callCounters) {
	t.Helper()
	counters := &callCounters{}
	allReviews := make([]*reviewdomain.Review, 0, len(in.reviews)+len(in.reviewList))
	for _, rv := range in.reviews {
		allReviews = append(allReviews, rv)
	}
	allReviews = append(allReviews, in.reviewList...)
	svc := appreferrer.NewService(&mockReferrerProfileRepo{}).WithReputationDeps(
		appreferrer.ReputationDeps{
			Referrals: &fakeReferralRepo{counters: counters, referrals: in.referrals, attribs: in.attribs},
			Proposals: &fakeProposalRepo{counters: counters, proposals: in.proposals},
			Reviews:   &fakeReviewRepo{counters: counters, reviews: allReviews},
			Users:     &fakeUserBatchReader{counters: counters, users: in.users},
		},
	)
	return svc, counters
}

func newCompletedProposal(id, clientID, providerID uuid.UUID, title string, completedAt time.Time) *proposaldomain.Proposal {
	ct := completedAt
	return &proposaldomain.Proposal{
		ID:          id,
		Title:       title,
		Status:      proposaldomain.StatusCompleted,
		ClientID:    clientID,
		ProviderID:  providerID,
		CompletedAt: &ct,
		CreatedAt:   completedAt.Add(-24 * time.Hour),
		UpdatedAt:   completedAt,
	}
}

func newProposal(id, clientID, providerID uuid.UUID, title string, status proposaldomain.ProposalStatus, createdAt time.Time) *proposaldomain.Proposal {
	return &proposaldomain.Proposal{
		ID:         id,
		Title:      title,
		Status:     status,
		ClientID:   clientID,
		ProviderID: providerID,
		CreatedAt:  createdAt,
		UpdatedAt:  createdAt,
	}
}

func newClientToProviderReview(proposalID, reviewerID, reviewedID uuid.UUID, rating int, comment string, createdAt time.Time) *reviewdomain.Review {
	publishedAt := createdAt
	return &reviewdomain.Review{
		ID:           uuid.New(),
		ProposalID:   proposalID,
		ReviewerID:   reviewerID,
		ReviewedID:   reviewedID,
		Side:         reviewdomain.SideClientToProvider,
		GlobalRating: rating,
		Comment:      comment,
		CreatedAt:    createdAt,
		UpdatedAt:    createdAt,
		PublishedAt:  &publishedAt,
	}
}

func newProviderToClientReview(proposalID, reviewerID, reviewedID uuid.UUID, rating int, createdAt time.Time) *reviewdomain.Review {
	publishedAt := createdAt
	return &reviewdomain.Review{
		ID:           uuid.New(),
		ProposalID:   proposalID,
		ReviewerID:   reviewerID,
		ReviewedID:   reviewedID,
		Side:         reviewdomain.SideProviderToClient,
		GlobalRating: rating,
		CreatedAt:    createdAt,
		UpdatedAt:    createdAt,
		PublishedAt:  &publishedAt,
	}
}

func newProviderUser(id uuid.UUID, displayName string) *userdomain.User {
	return &userdomain.User{
		ID:          id,
		Email:       "p@example.com",
		DisplayName: displayName,
		Role:        userdomain.RoleProvider,
	}
}

func newActiveReferralRow(id, referrerID, providerID, clientID uuid.UUID) *referraldomain.Referral {
	return &referraldomain.Referral{
		ID:           id,
		ReferrerID:   referrerID,
		ProviderID:   providerID,
		ClientID:     clientID,
		Status:       referraldomain.StatusActive,
		CreatedAt:    time.Now().Add(-30 * 24 * time.Hour),
		UpdatedAt:    time.Now(),
		LastActionAt: time.Now(),
	}
}

func newAttribution(id, referralID, proposalID, providerID, clientID uuid.UUID, attributedAt time.Time) *referraldomain.Attribution {
	return &referraldomain.Attribution{
		ID:              id,
		ReferralID:      referralID,
		ProposalID:      proposalID,
		ProviderID:      providerID,
		ClientID:        clientID,
		RatePctSnapshot: 10.0,
		AttributedAt:    attributedAt,
	}
}

// ─── Tests ───────────────────────────────────────────────────────────────

func TestGetReferrerReputation_ZeroReferrals_ReturnsEmpty(t *testing.T) {
	svc, _ := newServiceForReputation(t, setupInput{referrerID: uuid.New()})
	rep, err := svc.GetReferrerReputation(context.Background(), uuid.New(), "", 20)
	require.NoError(t, err)
	assert.Equal(t, 0.0, rep.RatingAvg)
	assert.Equal(t, 0, rep.ReviewCount)
	assert.Empty(t, rep.History)
	assert.Empty(t, rep.NextCursor)
}

func TestGetReferrerReputation_OneCompletedReviewed_AveragesToTheReviewRating(t *testing.T) {
	referrerID := uuid.New()
	providerID := uuid.New()
	clientID := uuid.New()
	referralID := uuid.New()
	proposalID := uuid.New()
	completedAt := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)

	in := setupInput{
		referrerID: referrerID,
		referrals:  []*referraldomain.Referral{newActiveReferralRow(referralID, referrerID, providerID, clientID)},
		attribs: []*referraldomain.Attribution{
			newAttribution(uuid.New(), referralID, proposalID, providerID, clientID, completedAt.Add(-48*time.Hour)),
		},
		proposals: []*proposaldomain.Proposal{
			newCompletedProposal(proposalID, clientID, providerID, "Build a landing page", completedAt),
		},
		reviews: map[uuid.UUID]*reviewdomain.Review{
			proposalID: newClientToProviderReview(proposalID, clientID, providerID, 4, "Great work", completedAt.Add(time.Hour)),
		},
		users: map[uuid.UUID]*userdomain.User{
			providerID: newProviderUser(providerID, "Provider Name"),
		},
	}
	svc, counters := newServiceForReputation(t, in)

	rep, err := svc.GetReferrerReputation(context.Background(), referrerID, "", 20)
	require.NoError(t, err)

	assert.Equal(t, 4.0, rep.RatingAvg)
	assert.Equal(t, 1, rep.ReviewCount)
	require.Len(t, rep.History, 1)
	entry := rep.History[0]
	assert.Equal(t, "Build a landing page", entry.ProposalTitle)
	assert.Equal(t, string(proposaldomain.StatusCompleted), entry.ProposalStatus)
	require.NotNil(t, entry.Review)
	assert.Equal(t, 4, entry.Review.GlobalRating)
	assert.Equal(t, reviewdomain.SideClientToProvider, entry.Review.Side)
	assert.Equal(t, "Great work", entry.Review.Comment)
	assert.Empty(t, rep.NextCursor)

	// Query budget: 1 list referrals, 1 list attributions, 1 proposals,
	// 1 reviews. User lookup is gone — the public apporteur surface no
	// longer resolves provider names (Modèle A confidentiality).
	assert.Equal(t, int32(1), counters.listReferrals.Load())
	assert.Equal(t, int32(1), counters.listAttributionsByIDs.Load())
	assert.Equal(t, int32(1), counters.getProposalsByIDs.Load())
	assert.Equal(t, int32(1), counters.getReviewsByProposalIDs.Load())
	assert.Equal(t, int32(0), counters.getUsersByIDs.Load(), "user batch reader must not be called anymore")
}

func TestGetReferrerReputation_DisputedMissionDoesNotContributeToRating(t *testing.T) {
	referrerID := uuid.New()
	providerID := uuid.New()
	clientID := uuid.New()
	referralID := uuid.New()
	reviewedProposalID := uuid.New()
	disputedProposalID := uuid.New()
	t0 := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)

	in := setupInput{
		referrerID: referrerID,
		referrals:  []*referraldomain.Referral{newActiveReferralRow(referralID, referrerID, providerID, clientID)},
		attribs: []*referraldomain.Attribution{
			newAttribution(uuid.New(), referralID, reviewedProposalID, providerID, clientID, t0.Add(-96*time.Hour)),
			newAttribution(uuid.New(), referralID, disputedProposalID, providerID, clientID, t0.Add(-48*time.Hour)),
		},
		proposals: []*proposaldomain.Proposal{
			newCompletedProposal(reviewedProposalID, clientID, providerID, "Completed mission", t0),
			newProposal(disputedProposalID, clientID, providerID, "Disputed mission", proposaldomain.StatusDisputed, t0.Add(-24*time.Hour)),
		},
		reviews: map[uuid.UUID]*reviewdomain.Review{
			reviewedProposalID: newClientToProviderReview(reviewedProposalID, clientID, providerID, 5, "Excellent", t0.Add(time.Hour)),
		},
		users: map[uuid.UUID]*userdomain.User{
			providerID: newProviderUser(providerID, "Provider Name"),
		},
	}
	svc, _ := newServiceForReputation(t, in)

	rep, err := svc.GetReferrerReputation(context.Background(), referrerID, "", 20)
	require.NoError(t, err)

	assert.Equal(t, 5.0, rep.RatingAvg)
	assert.Equal(t, 1, rep.ReviewCount)
	require.Len(t, rep.History, 2)
	// Completed, reviewed proposal comes first because completed_at > null.
	assert.Equal(t, "Completed mission", rep.History[0].ProposalTitle)
	assert.Equal(t, string(proposaldomain.StatusDisputed), rep.History[1].ProposalStatus)
	assert.Nil(t, rep.History[1].Review, "disputed mission must not carry a review on the public surface")
}

func TestGetReferrerReputation_CompletedWithoutReview_CountsInHistoryButNotInRating(t *testing.T) {
	referrerID := uuid.New()
	providerID := uuid.New()
	clientID := uuid.New()
	referralID := uuid.New()
	proposalID := uuid.New()
	completedAt := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)

	in := setupInput{
		referrerID: referrerID,
		referrals:  []*referraldomain.Referral{newActiveReferralRow(referralID, referrerID, providerID, clientID)},
		attribs: []*referraldomain.Attribution{
			newAttribution(uuid.New(), referralID, proposalID, providerID, clientID, completedAt.Add(-48*time.Hour)),
		},
		proposals: []*proposaldomain.Proposal{
			newCompletedProposal(proposalID, clientID, providerID, "Waiting mission", completedAt),
		},
		reviews: map[uuid.UUID]*reviewdomain.Review{},
		users: map[uuid.UUID]*userdomain.User{
			providerID: newProviderUser(providerID, "Provider Name"),
		},
	}
	svc, _ := newServiceForReputation(t, in)

	rep, err := svc.GetReferrerReputation(context.Background(), referrerID, "", 20)
	require.NoError(t, err)
	assert.Equal(t, 0.0, rep.RatingAvg)
	assert.Equal(t, 0, rep.ReviewCount)
	require.Len(t, rep.History, 1)
	assert.Nil(t, rep.History[0].Review)
}

func TestGetReferrerReputation_PaginationWalksAllPages(t *testing.T) {
	referrerID := uuid.New()
	providerID := uuid.New()
	clientID := uuid.New()
	referralID := uuid.New()
	base := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)

	const total = 30
	attribs := make([]*referraldomain.Attribution, 0, total)
	proposals := make([]*proposaldomain.Proposal, 0, total)
	for i := 0; i < total; i++ {
		proposalID := uuid.New()
		// Each completedAt is distinct so the sort order is stable.
		completedAt := base.Add(time.Duration(-i) * time.Hour)
		attribs = append(attribs, newAttribution(uuid.New(), referralID, proposalID, providerID, clientID, completedAt.Add(-24*time.Hour)))
		proposals = append(proposals, newCompletedProposal(proposalID, clientID, providerID, "m", completedAt))
	}

	in := setupInput{
		referrerID: referrerID,
		referrals:  []*referraldomain.Referral{newActiveReferralRow(referralID, referrerID, providerID, clientID)},
		attribs:    attribs,
		proposals:  proposals,
		reviews:    map[uuid.UUID]*reviewdomain.Review{},
		users: map[uuid.UUID]*userdomain.User{
			providerID: newProviderUser(providerID, "Provider Name"),
		},
	}
	svc, _ := newServiceForReputation(t, in)

	var cursor string
	var seen []uuid.UUID
	pages := 0
	for {
		rep, err := svc.GetReferrerReputation(context.Background(), referrerID, cursor, 10)
		require.NoError(t, err)
		for _, e := range rep.History {
			seen = append(seen, e.ProposalID)
		}
		pages++
		if rep.NextCursor == "" {
			break
		}
		cursor = rep.NextCursor
		require.LessOrEqual(t, pages, 5, "pagination must terminate within a few pages")
	}
	assert.Equal(t, 3, pages)
	assert.Len(t, seen, total)
	// No duplicates across pages.
	uniq := make(map[uuid.UUID]struct{}, total)
	for _, id := range seen {
		_, dup := uniq[id]
		assert.False(t, dup, "proposal %s appeared twice across pages", id)
		uniq[id] = struct{}{}
	}
}

// TestGetReferrerReputation_BothSidesSubmitted_SurfaceClientToProvider
// is the regression guard for the double-blind collision bug: a
// proposal carries BOTH a client→provider review (5★) AND a
// provider→client review (3★). Pre-fix, GetByProposalIDs used a map
// keyed on proposal_id without a side filter, so whichever side
// scanned last silently replaced the other — the apporteur's rating
// became non-deterministic and sub-criteria could vanish. Post-fix,
// the service requests the client→provider side explicitly at the
// SQL level and the aggregate stays stable at the client→provider
// rating (5), with the embedded review pointing at that side.
func TestGetReferrerReputation_BothSidesSubmitted_SurfaceClientToProvider(t *testing.T) {
	referrerID := uuid.New()
	providerID := uuid.New()
	clientID := uuid.New()
	referralID := uuid.New()
	proposalID := uuid.New()
	completedAt := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)

	clientSide := newClientToProviderReview(proposalID, clientID, providerID, 5, "Great work on the client side", completedAt.Add(time.Hour))
	providerSide := newProviderToClientReview(proposalID, providerID, clientID, 3, completedAt.Add(2*time.Hour))

	in := setupInput{
		referrerID: referrerID,
		referrals:  []*referraldomain.Referral{newActiveReferralRow(referralID, referrerID, providerID, clientID)},
		attribs: []*referraldomain.Attribution{
			newAttribution(uuid.New(), referralID, proposalID, providerID, clientID, completedAt.Add(-48*time.Hour)),
		},
		proposals: []*proposaldomain.Proposal{
			newCompletedProposal(proposalID, clientID, providerID, "Double-blind mission", completedAt),
		},
		// Both sides of the double-blind pair exist on the same proposal.
		reviewList: []*reviewdomain.Review{clientSide, providerSide},
		users: map[uuid.UUID]*userdomain.User{
			providerID: newProviderUser(providerID, "Provider Name"),
		},
	}
	svc, _ := newServiceForReputation(t, in)

	rep, err := svc.GetReferrerReputation(context.Background(), referrerID, "", 20)
	require.NoError(t, err)

	// Rating aggregate must reflect the CLIENT→PROVIDER side only (5★),
	// not the provider→client side (3★) — flaky before the fix.
	assert.Equal(t, 5.0, rep.RatingAvg, "apporteur rating must surface the client→provider side only")
	assert.Equal(t, 1, rep.ReviewCount)

	require.Len(t, rep.History, 1)
	entry := rep.History[0]
	require.NotNil(t, entry.Review, "review must be embedded on the apporteur surface")
	assert.Equal(t, reviewdomain.SideClientToProvider, entry.Review.Side, "embedded review must point at the client→provider side")
	assert.Equal(t, 5, entry.Review.GlobalRating)
	assert.Equal(t, "Great work on the client side", entry.Review.Comment)
	// ID must match the client→provider review, not the provider→client one.
	assert.Equal(t, clientSide.ID, entry.Review.ID)
}

func TestGetReferrerReputation_ProviderToClientReview_DoesNotCount(t *testing.T) {
	referrerID := uuid.New()
	providerID := uuid.New()
	clientID := uuid.New()
	referralID := uuid.New()
	proposalID := uuid.New()
	completedAt := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)

	in := setupInput{
		referrerID: referrerID,
		referrals:  []*referraldomain.Referral{newActiveReferralRow(referralID, referrerID, providerID, clientID)},
		attribs: []*referraldomain.Attribution{
			newAttribution(uuid.New(), referralID, proposalID, providerID, clientID, completedAt.Add(-48*time.Hour)),
		},
		proposals: []*proposaldomain.Proposal{
			newCompletedProposal(proposalID, clientID, providerID, "Mission", completedAt),
		},
		reviews: map[uuid.UUID]*reviewdomain.Review{
			// Wrong direction — provider rating the client. MUST not count.
			proposalID: newProviderToClientReview(proposalID, providerID, clientID, 5, completedAt.Add(time.Hour)),
		},
		users: map[uuid.UUID]*userdomain.User{
			providerID: newProviderUser(providerID, "Provider Name"),
		},
	}
	svc, _ := newServiceForReputation(t, in)

	rep, err := svc.GetReferrerReputation(context.Background(), referrerID, "", 20)
	require.NoError(t, err)
	assert.Equal(t, 0.0, rep.RatingAvg)
	assert.Equal(t, 0, rep.ReviewCount)
	require.Len(t, rep.History, 1)
	assert.Nil(t, rep.History[0].Review, "provider→client review must never be surfaced as the apporteur's score")
}

func TestGetReferrerReputation_InvalidCursor_ReturnsError(t *testing.T) {
	referrerID := uuid.New()
	providerID := uuid.New()
	clientID := uuid.New()
	referralID := uuid.New()
	proposalID := uuid.New()
	completedAt := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)
	in := setupInput{
		referrerID: referrerID,
		referrals:  []*referraldomain.Referral{newActiveReferralRow(referralID, referrerID, providerID, clientID)},
		attribs: []*referraldomain.Attribution{
			newAttribution(uuid.New(), referralID, proposalID, providerID, clientID, completedAt.Add(-24*time.Hour)),
		},
		proposals: []*proposaldomain.Proposal{newCompletedProposal(proposalID, clientID, providerID, "m", completedAt)},
		users:     map[uuid.UUID]*userdomain.User{providerID: newProviderUser(providerID, "P")},
	}
	svc, _ := newServiceForReputation(t, in)

	_, err := svc.GetReferrerReputation(context.Background(), referrerID, "not-a-valid-cursor!!!", 20)
	assert.Error(t, err)
}

// Sanity: the reputation surface stays nil-safe when deps are missing.
func TestGetReferrerReputation_MissingDeps_ReturnsEmpty(t *testing.T) {
	svc := appreferrer.NewService(&mockReferrerProfileRepo{})
	rep, err := svc.GetReferrerReputation(context.Background(), uuid.New(), "", 20)
	require.NoError(t, err)
	assert.Empty(t, rep.History)
	assert.Equal(t, 0, rep.ReviewCount)
}

// Compile-time guard: errors package must still be reachable from the
// test file since other assertions rely on it indirectly via ErrorIs.
var _ = errors.New
