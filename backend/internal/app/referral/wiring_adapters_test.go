package referral_test

// Coverage for the wiring adapter helpers — every adapter must:
//   - return safe defaults when nil-wired (graceful degradation contract)
//   - produce the right shape when wired correctly
//
// These adapters are the boundary between the referral feature and
// other repositories (proposal, milestone, organization, freelance
// profile). They explicitly avoid importing the domains of the called
// features — only the repository ports — so a deletion of (e.g.) the
// proposal feature does not break the referral build.

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	referralapp "marketplace-backend/internal/app/referral"
	"marketplace-backend/internal/domain/milestone"
	"marketplace-backend/internal/domain/organization"
	"marketplace-backend/internal/domain/proposal"
	"marketplace-backend/internal/port/repository"
)

// ─── ThinSnapshotLoader ───────────────────────────────────────────────

func TestThinSnapshotLoader_NilSafe(t *testing.T) {
	loader := referralapp.NewThinSnapshotLoader(nil)
	require.NotNil(t, loader)
	prov, err := loader.LoadProvider(context.Background(), uuid.New())
	require.NoError(t, err)
	assert.Equal(t, "", prov.Region)
	cli, err := loader.LoadClient(context.Background(), uuid.New())
	require.NoError(t, err)
	assert.Equal(t, "", cli.Industry)
}

// ─── OrgStripeAccountResolver ─────────────────────────────────────────

func TestOrgStripeAccountResolver_NilSafe(t *testing.T) {
	r := referralapp.NewOrgStripeAccountResolver(nil)
	got, err := r.ResolveStripeAccountID(context.Background(), uuid.New())
	require.NoError(t, err)
	assert.Equal(t, "", got)
}

type stubOrgRepoForResolver struct {
	repository.OrganizationRepository
	accountID string
	err       error
}

func (s *stubOrgRepoForResolver) GetStripeAccountByUserID(_ context.Context, _ uuid.UUID) (string, string, error) {
	return s.accountID, "FR", s.err
}

func TestOrgStripeAccountResolver_ReturnsAccount(t *testing.T) {
	r := referralapp.NewOrgStripeAccountResolver(&stubOrgRepoForResolver{accountID: "acct_x"})
	got, err := r.ResolveStripeAccountID(context.Background(), uuid.New())
	require.NoError(t, err)
	assert.Equal(t, "acct_x", got)
}

func TestOrgStripeAccountResolver_RepoError_ReturnsEmptyNoError(t *testing.T) {
	r := referralapp.NewOrgStripeAccountResolver(&stubOrgRepoForResolver{err: errors.New("db")})
	got, err := r.ResolveStripeAccountID(context.Background(), uuid.New())
	require.NoError(t, err, "soft-failure: caller parks the commission instead of erroring")
	assert.Equal(t, "", got)
}

// ─── ProposalRepoSummaryResolver ──────────────────────────────────────

type stubProposalRepoForSummary struct {
	repository.ProposalRepository
	byID map[uuid.UUID]*proposal.Proposal
	err  error
}

func (s *stubProposalRepoForSummary) GetByID(_ context.Context, id uuid.UUID) (*proposal.Proposal, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.byID[id], nil
}

type stubMilestoneRepoForSummary struct {
	byProposal map[uuid.UUID][]*milestone.Milestone
	err        error
}

func (s *stubMilestoneRepoForSummary) ListByProposals(_ context.Context, ids []uuid.UUID) (map[uuid.UUID][]*milestone.Milestone, error) {
	if s.err != nil {
		return nil, s.err
	}
	out := map[uuid.UUID][]*milestone.Milestone{}
	for _, id := range ids {
		if list, ok := s.byProposal[id]; ok {
			out[id] = list
		}
	}
	return out, nil
}

// stubMilestoneRepoForSummary needs to satisfy MilestoneRepository — make
// the rest panic if called via embedding the interface.
type stubMilestoneFull struct {
	repository.MilestoneRepository
	*stubMilestoneRepoForSummary
}

func (s *stubMilestoneFull) ListByProposals(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID][]*milestone.Milestone, error) {
	return s.stubMilestoneRepoForSummary.ListByProposals(ctx, ids)
}

func TestProposalRepoSummaryResolver_NilProposals_ReturnsEmpty(t *testing.T) {
	r := referralapp.NewProposalRepoSummaryResolver(nil, nil)
	out, err := r.ResolveProposalSummaries(context.Background(), []uuid.UUID{uuid.New()})
	require.NoError(t, err)
	assert.Empty(t, out)
}

func TestProposalRepoSummaryResolver_EmptyIDs(t *testing.T) {
	r := referralapp.NewProposalRepoSummaryResolver(&stubProposalRepoForSummary{}, nil)
	out, err := r.ResolveProposalSummaries(context.Background(), nil)
	require.NoError(t, err)
	assert.Empty(t, out)
}

func TestProposalRepoSummaryResolver_HappyPath_AggregatesFundedMilestones(t *testing.T) {
	pid := uuid.New()
	proposalRepo := &stubProposalRepoForSummary{
		byID: map[uuid.UUID]*proposal.Proposal{
			pid: {ID: pid, Title: "T", Status: proposal.StatusActive},
		},
	}
	mileRepo := &stubMilestoneFull{
		stubMilestoneRepoForSummary: &stubMilestoneRepoForSummary{
			byProposal: map[uuid.UUID][]*milestone.Milestone{
				pid: {
					{Status: milestone.StatusFunded, Amount: 1000},
					{Status: milestone.StatusSubmitted, Amount: 500},
					{Status: milestone.StatusReleased, Amount: 9999},   // released → not counted
					{Status: milestone.StatusCancelled, Amount: 7777},  // cancelled → not counted
				},
			},
		},
	}
	r := referralapp.NewProposalRepoSummaryResolver(proposalRepo, mileRepo)
	out, err := r.ResolveProposalSummaries(context.Background(), []uuid.UUID{pid})
	require.NoError(t, err)
	require.Contains(t, out, pid)
	summary := out[pid]
	assert.Equal(t, "T", summary.Title)
	assert.Equal(t, 4, summary.MilestonesTotal)
	assert.Equal(t, 2, summary.MilestonesFunded)
	assert.Equal(t, int64(1500), summary.FundedAmountCents,
		"only Funded+Submitted milestones contribute to the escrow bucket")
}

func TestProposalRepoSummaryResolver_ProposalLookupFailure_SkipsMissing(t *testing.T) {
	pid := uuid.New()
	missing := uuid.New()
	proposalRepo := &stubProposalRepoForSummary{
		byID: map[uuid.UUID]*proposal.Proposal{
			pid: {ID: pid, Title: "T"},
		},
	}
	r := referralapp.NewProposalRepoSummaryResolver(proposalRepo, nil)
	out, err := r.ResolveProposalSummaries(context.Background(), []uuid.UUID{pid, missing})
	require.NoError(t, err)
	assert.Len(t, out, 1, "missing rows are silently dropped — UI degrades gracefully")
	assert.Contains(t, out, pid)
}

func TestProposalRepoSummaryResolver_MilestoneError_DegradesToCountsZero(t *testing.T) {
	pid := uuid.New()
	proposalRepo := &stubProposalRepoForSummary{
		byID: map[uuid.UUID]*proposal.Proposal{pid: {ID: pid, Title: "T"}},
	}
	mileRepo := &stubMilestoneFull{
		stubMilestoneRepoForSummary: &stubMilestoneRepoForSummary{err: errors.New("ms err")},
	}
	r := referralapp.NewProposalRepoSummaryResolver(proposalRepo, mileRepo)
	out, err := r.ResolveProposalSummaries(context.Background(), []uuid.UUID{pid})
	require.NoError(t, err, "milestone failure must NOT bubble — UI shows zero counts")
	assert.Equal(t, 0, out[pid].MilestonesTotal)
}

// ─── OrgDirectoryMemberResolver ───────────────────────────────────────

type stubOrgRepoForMembers struct {
	repository.OrganizationRepository
	org *organization.Organization
	err error
}

func (s *stubOrgRepoForMembers) FindByUserID(_ context.Context, _ uuid.UUID) (*organization.Organization, error) {
	return s.org, s.err
}

type stubMembersRepo struct {
	repository.OrganizationMemberRepository
	byOrgFn func(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID][]uuid.UUID, error)
}

func (s *stubMembersRepo) ListMemberUserIDsByOrgIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID][]uuid.UUID, error) {
	return s.byOrgFn(ctx, ids)
}

func TestOrgDirectoryMemberResolver_NilSafe_FallbackToSelf(t *testing.T) {
	r := referralapp.NewOrgDirectoryMemberResolver(nil, nil)
	uid := uuid.New()
	got, err := r.ResolveMemberUserIDs(context.Background(), uid)
	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{uid}, got, "no org wiring → caller is the only recipient")
}

func TestOrgDirectoryMemberResolver_NoOrg_FallbackToSelf(t *testing.T) {
	orgs := &stubOrgRepoForMembers{org: nil}
	members := &stubMembersRepo{}
	r := referralapp.NewOrgDirectoryMemberResolver(orgs, members)
	uid := uuid.New()
	got, err := r.ResolveMemberUserIDs(context.Background(), uid)
	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{uid}, got)
}

func TestOrgDirectoryMemberResolver_AllMembers(t *testing.T) {
	orgID := uuid.New()
	uid := uuid.New()
	other1 := uuid.New()
	other2 := uuid.New()
	orgs := &stubOrgRepoForMembers{org: &organization.Organization{ID: orgID}}
	members := &stubMembersRepo{
		byOrgFn: func(_ context.Context, _ []uuid.UUID) (map[uuid.UUID][]uuid.UUID, error) {
			return map[uuid.UUID][]uuid.UUID{
				orgID: {uid, other1, other2, other1}, // duplicate to test dedup
			}, nil
		},
	}
	r := referralapp.NewOrgDirectoryMemberResolver(orgs, members)
	got, err := r.ResolveMemberUserIDs(context.Background(), uid)
	require.NoError(t, err)
	require.Len(t, got, 3, "duplicates must be deduped")
	// Verify all expected IDs present.
	idSet := map[uuid.UUID]bool{}
	for _, id := range got {
		idSet[id] = true
	}
	assert.True(t, idSet[uid])
	assert.True(t, idSet[other1])
	assert.True(t, idSet[other2])
}

func TestOrgDirectoryMemberResolver_AnchorAlwaysIncluded(t *testing.T) {
	orgID := uuid.New()
	uid := uuid.New()
	other := uuid.New()
	orgs := &stubOrgRepoForMembers{org: &organization.Organization{ID: orgID}}
	members := &stubMembersRepo{
		byOrgFn: func(_ context.Context, _ []uuid.UUID) (map[uuid.UUID][]uuid.UUID, error) {
			// Membership query returns OTHER members but not the anchor —
			// the resolver must still include the anchor.
			return map[uuid.UUID][]uuid.UUID{orgID: {other}}, nil
		},
	}
	r := referralapp.NewOrgDirectoryMemberResolver(orgs, members)
	got, err := r.ResolveMemberUserIDs(context.Background(), uid)
	require.NoError(t, err)
	require.Len(t, got, 2)
	idSet := map[uuid.UUID]bool{}
	for _, id := range got {
		idSet[id] = true
	}
	assert.True(t, idSet[uid], "anchor user must always be included even if membership query missed them")
}

func TestOrgDirectoryMemberResolver_MembersError_FallbackToSelf(t *testing.T) {
	orgs := &stubOrgRepoForMembers{org: &organization.Organization{ID: uuid.New()}}
	members := &stubMembersRepo{
		byOrgFn: func(_ context.Context, _ []uuid.UUID) (map[uuid.UUID][]uuid.UUID, error) {
			return nil, errors.New("db down")
		},
	}
	r := referralapp.NewOrgDirectoryMemberResolver(orgs, members)
	uid := uuid.New()
	got, err := r.ResolveMemberUserIDs(context.Background(), uid)
	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{uid}, got)
}

// quick silence on unused imports (some tests don't trigger every block)
var _ = time.Now
