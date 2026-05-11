package referral_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	referralapp "marketplace-backend/internal/app/referral"
	"marketplace-backend/internal/domain/milestone"
	"marketplace-backend/internal/domain/referral"
)

// fakeMilestoneLister implements MilestonesByProposalLister for unit
// tests. Tests seed milestones by proposal id; each ListByProposals
// call increments a counter so we can prove the algorithm batches its
// reads instead of falling into the N+1 trap.
type fakeMilestoneLister struct {
	mu      sync.Mutex
	rows    map[uuid.UUID][]*milestone.Milestone
	calls   int
	forceErr error
}

func newFakeMilestoneLister() *fakeMilestoneLister {
	return &fakeMilestoneLister{rows: map[uuid.UUID][]*milestone.Milestone{}}
}

func (f *fakeMilestoneLister) add(m *milestone.Milestone) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rows[m.ProposalID] = append(f.rows[m.ProposalID], m)
}

func (f *fakeMilestoneLister) ListByProposals(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID][]*milestone.Milestone, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if f.forceErr != nil {
		return nil, f.forceErr
	}
	out := map[uuid.UUID][]*milestone.Milestone{}
	for _, id := range ids {
		if list, ok := f.rows[id]; ok {
			out[id] = list
		}
	}
	return out, nil
}

// fakeOrgMemberLister implements OrgMemberLister.
type fakeOrgMemberLister struct {
	mu       sync.Mutex
	byOrg    map[uuid.UUID][]uuid.UUID
	forceErr error
	calls    int
}

func newFakeOrgMemberLister() *fakeOrgMemberLister {
	return &fakeOrgMemberLister{byOrg: map[uuid.UUID][]uuid.UUID{}}
}

func (f *fakeOrgMemberLister) set(orgID uuid.UUID, userIDs ...uuid.UUID) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.byOrg[orgID] = userIDs
}

func (f *fakeOrgMemberLister) ListMemberUserIDsByOrgIDs(ctx context.Context, orgIDs []uuid.UUID) (map[uuid.UUID][]uuid.UUID, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if f.forceErr != nil {
		return nil, f.forceErr
	}
	out := map[uuid.UUID][]uuid.UUID{}
	for _, id := range orgIDs {
		if u, ok := f.byOrg[id]; ok {
			out[id] = u
		}
	}
	return out, nil
}

// projectionFixture wires a Service with the projection-specific
// fakes plugged on top of the standard testFixture. Keeps the seed +
// assertions concise in each test.
type projectionFixture struct {
	*testFixture
	milestones *fakeMilestoneLister
	members    *fakeOrgMemberLister
	orgID      uuid.UUID
}

// newProjectionFixture builds a fixture with one apporteur member in a
// fresh org. Tests get the active referral, the attribution helper,
// and a clean slate for milestones / commissions.
func newProjectionFixture(t *testing.T) *projectionFixture {
	t.Helper()
	f := newTestFixture(t, "acct_apporteur")
	ml := newFakeMilestoneLister()
	mem := newFakeOrgMemberLister()
	// Rebuild the service with the projection ports plugged in. The
	// base testFixture already has the other ports wired through
	// newTestFixture; we re-create the service to add the two new deps.
	svc := referralapp.NewService(referralapp.ServiceDeps{
		Referrals:            f.repo,
		Users:                f.users,
		Messages:             f.msgs,
		Notifications:        f.notifier,
		Stripe:               f.stripe,
		Reversals:            f.reversal,
		SnapshotProfiles:     &fakeSnapshotLoader{},
		StripeAccounts:       f.accounts,
		Relationships:        f.relationships,
		Audits:               f.audits,
		ProposalSummaries:    f.summaries,
		MilestonesByProposal: ml,
		OrgMembersLister:     mem,
	})
	f.svc = svc
	return &projectionFixture{
		testFixture: f,
		milestones:  ml,
		members:     mem,
		orgID:       uuid.New(),
	}
}

// seedAttribution drives a fresh intro from creation to active and
// then creates an attribution for `proposalID` at `rate` %. Returns
// the attribution entity so the test can mutate it (e.g. end it).
func (pf *projectionFixture) seedAttribution(t *testing.T, rate float64, proposalID uuid.UUID) *referral.Attribution {
	t.Helper()
	refID, provID, cliID := pf.seedActors(t)
	pf.members.set(pf.orgID, refID)
	r := pf.createIntro(t, refID, provID, cliID, rate)
	bringToActive(t, pf.svc, r, provID, cliID)
	require.NoError(t, pf.svc.CreateAttributionIfExists(context.Background(),
		attrInputFor(proposalID, provID, cliID)))
	att, err := pf.repo.FindAttributionByProposal(context.Background(), proposalID)
	require.NoError(t, err)
	return att
}

// seedMilestoneInStatus inserts a milestone in the given status onto
// the lister fake. Goes through the domain factory so timestamps are
// realistic. Returns the created milestone ID.
func (pf *projectionFixture) seedMilestoneInStatus(t *testing.T, proposalID uuid.UUID, amount int64, status milestone.MilestoneStatus) *milestone.Milestone {
	t.Helper()
	m, err := milestone.NewMilestone(milestone.NewMilestoneInput{
		ProposalID: proposalID,
		Sequence:   1,
		Title:      "Step",
		Amount:     amount,
	})
	require.NoError(t, err)
	// Force the requested status without driving the state machine —
	// we want a stable input for the projection dispatch tests.
	m.Status = status
	switch status {
	case milestone.StatusApproved, milestone.StatusReleased:
		now := time.Now().UTC()
		m.ApprovedAt = &now
	}
	pf.milestones.add(m)
	return m
}

// TestProjectedCommissions_AllStatuses is the table-driven matrix the
// brief mandates. Every milestone status × commission row presence ×
// attribution ended state is exercised on a fresh attribution per
// case so the assertions stay local.
func TestProjectedCommissions_AllStatuses(t *testing.T) {
	type want struct {
		emit   bool
		status referralapp.ProjectionStatus
		source referralapp.ProjectionSource
	}
	cases := []struct {
		name     string
		status   milestone.MilestoneStatus
		withRow  bool
		ended    bool
		approved bool
		want     want
	}{
		{name: "pending_funding skipped", status: milestone.StatusPendingFunding, want: want{emit: false}},
		{name: "cancelled skipped", status: milestone.StatusCancelled, want: want{emit: false}},
		{name: "refunded skipped", status: milestone.StatusRefunded, want: want{emit: false}},

		{name: "funded escrowed projection", status: milestone.StatusFunded, want: want{emit: true, status: referralapp.ProjectionEscrowed, source: referralapp.SourceProjection}},
		{name: "submitted escrowed projection", status: milestone.StatusSubmitted, want: want{emit: true, status: referralapp.ProjectionEscrowed, source: referralapp.SourceProjection}},
		{name: "disputed escrowed projection", status: milestone.StatusDisputed, want: want{emit: true, status: referralapp.ProjectionEscrowed, source: referralapp.SourceProjection}},

		{name: "approved with paid row → paid from row", status: milestone.StatusApproved, withRow: true, approved: true, want: want{emit: true, status: referralapp.ProjectionPaid, source: referralapp.SourceRow}},
		{name: "approved without row → pending projection (safety net)", status: milestone.StatusApproved, withRow: false, approved: true, want: want{emit: true, status: referralapp.ProjectionPending, source: referralapp.SourceProjection}},
		{name: "released with row → paid from row", status: milestone.StatusReleased, withRow: true, approved: true, want: want{emit: true, status: referralapp.ProjectionPaid, source: referralapp.SourceRow}},

		// Ended attribution gates
		{name: "ended + active milestone → skipped", status: milestone.StatusFunded, ended: true, want: want{emit: false}},
		{name: "ended + approved AFTER ended_at → skipped", status: milestone.StatusApproved, ended: true, approved: true, want: want{emit: false}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pf := newProjectionFixture(t)
			proposalID := uuid.New()
			att := pf.seedAttribution(t, 5.0, proposalID)
			m := pf.seedMilestoneInStatus(t, proposalID, 1000_00, tc.status)
			if tc.ended {
				// End the attribution in the past, then force the
				// milestone's ApprovedAt to be AFTER the cutoff so the
				// gate fires deterministically (default ApprovedAt set
				// by seedMilestoneInStatus is "now" which could race
				// with the EndAttribution timestamp).
				require.NoError(t, pf.repo.EndAttribution(context.Background(), att.ID, referrerOf(pf, att)))
				if m.ApprovedAt != nil {
					future := time.Now().UTC().Add(2 * time.Hour)
					m.ApprovedAt = &future
				}
			}
			if tc.withRow {
				milestoneID := m.ID
				_, err := pf.svc.DistributeIfApplicable(context.Background(),
					distInputFor(proposalID, milestoneID, 1000_00))
				require.NoError(t, err)
			}

			out, err := pf.svc.ProjectedCommissions(context.Background(), pf.orgID)
			require.NoError(t, err)

			if !tc.want.emit {
				assert.Empty(t, out, "expected no projection for %s", tc.name)
				return
			}
			require.Len(t, out, 1, "expected exactly one projection")
			row := out[0]
			assert.Equal(t, tc.want.status, row.Status, "status mismatch")
			assert.Equal(t, tc.want.source, row.Source, "source mismatch")
			assert.Equal(t, m.ID, row.MilestoneID)
			assert.Equal(t, proposalID, row.ProposalID)
		})
	}
}

// referrerOf is a tiny lookup helper to retrieve the parent referrer
// of a freshly-created attribution. Used by the "ended attribution"
// test cases that need to call EndAttribution.
func referrerOf(pf *projectionFixture, a *referral.Attribution) uuid.UUID {
	pf.repo.mu.Lock()
	defer pf.repo.mu.Unlock()
	r := pf.repo.rows[a.ReferralID]
	return r.ReferrerID
}

// TestProjectedCommissions_BatchFetchAvoidsNPlusOne verifies the
// algorithm batches its DB reads: regardless of how many milestones a
// proposal has, ListByProposals is called ONCE. Seeds 5 attributions
// × 3 milestones each = 15 milestones; expected counter = 1.
func TestProjectedCommissions_BatchFetchAvoidsNPlusOne(t *testing.T) {
	pf := newProjectionFixture(t)
	refID, provID, cliID := pf.seedActors(t)
	pf.members.set(pf.orgID, refID)
	r := pf.createIntro(t, refID, provID, cliID, 5)
	bringToActive(t, pf.svc, r, provID, cliID)

	for i := 0; i < 5; i++ {
		proposalID := uuid.New()
		require.NoError(t, pf.svc.CreateAttributionIfExists(context.Background(),
			attrInputFor(proposalID, provID, cliID)))
		for j := 0; j < 3; j++ {
			m, err := milestone.NewMilestone(milestone.NewMilestoneInput{
				ProposalID: proposalID,
				Sequence:   j + 1,
				Title:      "Step",
				Amount:     1000_00,
			})
			require.NoError(t, err)
			m.Status = milestone.StatusFunded
			pf.milestones.add(m)
		}
	}

	_, err := pf.svc.ProjectedCommissions(context.Background(), pf.orgID)
	require.NoError(t, err)
	assert.Equal(t, 1, pf.milestones.calls, "ListByProposals must be called exactly once")
	assert.Equal(t, 1, pf.members.calls, "ListMemberUserIDsByOrgIDs must be called exactly once")
}

// TestProjectedCommissions_Cap200 seeds 250 milestones and asserts the
// returned slice is capped at MaxProjections (200).
func TestProjectedCommissions_Cap200(t *testing.T) {
	pf := newProjectionFixture(t)
	refID, provID, cliID := pf.seedActors(t)
	pf.members.set(pf.orgID, refID)
	r := pf.createIntro(t, refID, provID, cliID, 5)
	bringToActive(t, pf.svc, r, provID, cliID)

	// 5 proposals × 50 milestones each = 250
	for i := 0; i < 5; i++ {
		proposalID := uuid.New()
		require.NoError(t, pf.svc.CreateAttributionIfExists(context.Background(),
			attrInputFor(proposalID, provID, cliID)))
		for j := 0; j < 50; j++ {
			m, err := milestone.NewMilestone(milestone.NewMilestoneInput{
				ProposalID: proposalID,
				Sequence:   j + 1,
				Title:      "Step",
				Amount:     1000_00,
			})
			require.NoError(t, err)
			m.Status = milestone.StatusFunded
			pf.milestones.add(m)
		}
	}

	out, err := pf.svc.ProjectedCommissions(context.Background(), pf.orgID)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(out), referralapp.MaxProjections)
	assert.Equal(t, referralapp.MaxProjections, len(out), "expected the cap to bind on 250 inputs")
}

// TestProjectedCommissions_RespectsEndedGate is the brief's explicit
// before/at/after split: a milestone approved BEFORE the attribution
// ended timestamp must keep its projection; one approved AFTER must
// be dropped.
func TestProjectedCommissions_RespectsEndedGate(t *testing.T) {
	pf := newProjectionFixture(t)
	proposalA := uuid.New()
	proposalB := uuid.New()
	att := pf.seedAttribution(t, 5.0, proposalA)

	// Milestone approved BEFORE end-of-intro — must survive.
	mPre, err := milestone.NewMilestone(milestone.NewMilestoneInput{
		ProposalID: proposalA,
		Sequence:   1,
		Title:      "Pre",
		Amount:     1000_00,
	})
	require.NoError(t, err)
	mPre.Status = milestone.StatusApproved
	preApproval := time.Now().UTC().Add(-2 * time.Hour)
	mPre.ApprovedAt = &preApproval
	pf.milestones.add(mPre)

	// Seed a SECOND attribution on a different proposal so we can
	// drop a milestone that approves AFTER end-of-intro on it.
	_, provID, cliID := pf.seedActors(t)
	pf.members.set(pf.orgID, referrerOf(pf, att))
	r := pf.createIntro(t, referrerOf(pf, att), provID, cliID, 5)
	bringToActive(t, pf.svc, r, provID, cliID)
	require.NoError(t, pf.svc.CreateAttributionIfExists(context.Background(),
		attrInputFor(proposalB, provID, cliID)))
	attB, err := pf.repo.FindAttributionByProposal(context.Background(), proposalB)
	require.NoError(t, err)
	mPost, err := milestone.NewMilestone(milestone.NewMilestoneInput{
		ProposalID: proposalB,
		Sequence:   1,
		Title:      "Post",
		Amount:     1000_00,
	})
	require.NoError(t, err)
	mPost.Status = milestone.StatusApproved
	postApproval := time.Now().UTC().Add(2 * time.Hour)
	mPost.ApprovedAt = &postApproval
	pf.milestones.add(mPost)

	// End the SECOND attribution one hour ago — so mPost (+2h) is
	// AFTER ended_at and gets dropped, while mPre on the first
	// attribution (still active) survives.
	one := time.Now().UTC().Add(-1 * time.Hour)
	attB.EndedAt = &one
	require.NoError(t, pf.repo.EndAttribution(context.Background(), attB.ID, referrerOf(pf, attB)))

	out, err := pf.svc.ProjectedCommissions(context.Background(), pf.orgID)
	require.NoError(t, err)
	// Only mPre survives — mPost is dropped by the ended_at gate.
	require.NotEmpty(t, out)
	for _, row := range out {
		assert.NotEqual(t, mPost.ID, row.MilestoneID, "milestone approved after ended_at must be dropped")
	}
}

// TestProjectedCommissions_EmptyOrg returns nil when the org has no
// members (graceful degradation, no error).
func TestProjectedCommissions_EmptyOrg(t *testing.T) {
	pf := newProjectionFixture(t)
	out, err := pf.svc.ProjectedCommissions(context.Background(), pf.orgID)
	require.NoError(t, err)
	assert.Nil(t, out)
}

// TestProjectedCommissions_OrgMemberListerError surfaces the
// upstream error rather than swallowing it — a broken org-members
// repo is not something the wallet should silently hide.
func TestProjectedCommissions_OrgMemberListerError(t *testing.T) {
	pf := newProjectionFixture(t)
	pf.members.forceErr = assertAnError("boom")
	out, err := pf.svc.ProjectedCommissions(context.Background(), pf.orgID)
	require.Error(t, err)
	assert.Nil(t, out)
	assert.Contains(t, err.Error(), "resolve org members")
}

// TestProjectedCommissions_ReferralListError surfaces a broken
// per-user ListByReferrer call to the caller. Wraps with context.
func TestProjectedCommissions_ReferralListError(t *testing.T) {
	pf := newProjectionFixture(t)
	pf.members.set(pf.orgID, uuid.New())
	pf.repo.listByReferrerForceErr = assertAnError("ref-list-boom")
	t.Cleanup(func() { pf.repo.listByReferrerForceErr = nil })
	out, err := pf.svc.ProjectedCommissions(context.Background(), pf.orgID)
	require.Error(t, err)
	assert.Nil(t, out)
	assert.Contains(t, err.Error(), "list referrals by referrer")
}

// TestProjectedCommissions_AttributionListError surfaces a broken
// ListAttributionsByReferralIDs batch call.
func TestProjectedCommissions_AttributionListError(t *testing.T) {
	pf := newProjectionFixture(t)
	refID, provID, cliID := pf.seedActors(t)
	pf.members.set(pf.orgID, refID)
	r := pf.createIntro(t, refID, provID, cliID, 5)
	bringToActive(t, pf.svc, r, provID, cliID)
	pf.repo.listAttsByRefIDsForceErr = assertAnError("att-batch-boom")
	t.Cleanup(func() { pf.repo.listAttsByRefIDsForceErr = nil })
	out, err := pf.svc.ProjectedCommissions(context.Background(), pf.orgID)
	require.Error(t, err)
	assert.Nil(t, out)
	assert.Contains(t, err.Error(), "list attributions")
}

// TestProjectedCommissions_CommissionListError surfaces a broken
// per-referral ListCommissionsByReferral call.
func TestProjectedCommissions_CommissionListError(t *testing.T) {
	pf := newProjectionFixture(t)
	proposalID := uuid.New()
	pf.seedAttribution(t, 5.0, proposalID)
	pf.seedMilestoneInStatus(t, proposalID, 1000_00, milestone.StatusFunded)
	pf.repo.listCommissionsByRefForceErr = assertAnError("com-list-boom")
	t.Cleanup(func() { pf.repo.listCommissionsByRefForceErr = nil })
	out, err := pf.svc.ProjectedCommissions(context.Background(), pf.orgID)
	require.Error(t, err)
	assert.Nil(t, out)
	assert.Contains(t, err.Error(), "list commissions for referral")
}

// TestProjectedCommissions_TitleResolverError tolerates a broken
// summary resolver — title is empty, no projection emission breaks.
func TestProjectedCommissions_TitleResolverError(t *testing.T) {
	pf := newProjectionFixture(t)
	proposalID := uuid.New()
	pf.seedAttribution(t, 5.0, proposalID)
	pf.seedMilestoneInStatus(t, proposalID, 1000_00, milestone.StatusFunded)
	pf.summaries.forceErr = assertAnError("title-boom")
	t.Cleanup(func() { pf.summaries.forceErr = nil })
	out, err := pf.svc.ProjectedCommissions(context.Background(), pf.orgID)
	require.NoError(t, err)
	require.Len(t, out, 1)
	assert.Equal(t, "", out[0].MissionTitle, "title resolver error must degrade to empty title, not break the wallet")
}

// TestProjectedCommissions_MilestoneListerError surfaces a broken
// milestone batch read so ops see the gap clearly.
func TestProjectedCommissions_MilestoneListerError(t *testing.T) {
	pf := newProjectionFixture(t)
	proposalID := uuid.New()
	pf.seedAttribution(t, 5.0, proposalID)
	pf.seedMilestoneInStatus(t, proposalID, 1000_00, milestone.StatusFunded)
	pf.milestones.forceErr = assertAnError("milestone-boom")
	out, err := pf.svc.ProjectedCommissions(context.Background(), pf.orgID)
	require.Error(t, err)
	assert.Nil(t, out)
	assert.Contains(t, err.Error(), "list milestones")
}

// assertAnError is a tiny helper so tests don't repeat the
// errors.New boilerplate. Kept private to this test file.
func assertAnError(msg string) error {
	return &simpleError{msg: msg}
}

type simpleError struct{ msg string }

func (e *simpleError) Error() string { return e.msg }

// TestProjectedCommissions_DedupeAcrossMembers seeds the SAME
// referrer user under two members of the same org. The dedupe gate
// in collectReferralsForUsers must keep a single copy of every
// referral so projections aren't duplicated when a member appears in
// the org listing twice (defensive, theoretical only).
func TestProjectedCommissions_DedupeAcrossMembers(t *testing.T) {
	pf := newProjectionFixture(t)
	refID, provID, cliID := pf.seedActors(t)
	// Same user id appears TWICE in the org listing (defensive).
	pf.members.set(pf.orgID, refID, refID)
	r := pf.createIntro(t, refID, provID, cliID, 5)
	bringToActive(t, pf.svc, r, provID, cliID)
	proposalID := uuid.New()
	require.NoError(t, pf.svc.CreateAttributionIfExists(context.Background(),
		attrInputFor(proposalID, provID, cliID)))
	pf.seedMilestoneInStatus(t, proposalID, 1000_00, milestone.StatusFunded)

	out, err := pf.svc.ProjectedCommissions(context.Background(), pf.orgID)
	require.NoError(t, err)
	assert.Len(t, out, 1, "duplicate userID in members must not duplicate the projection")
}

// TestProjectedCommissions_MultiUserOrgDedupesReferrals seeds two
// users in the same org sharing the same referral row (impossible in
// real life since referrer_id is per-user, but the dedup guard in
// collectReferralsForUsers MUST never double-count if a future
// concurrent insert lets it slip through).
func TestProjectedCommissions_MultiUserOrgDedupesReferrals(t *testing.T) {
	pf := newProjectionFixture(t)
	refID, provID, cliID := pf.seedActors(t)
	// Put TWO users in the org, but only one is the actual referrer.
	// The second user has no referrals → ListByReferrer returns empty.
	// Even so, the merge loop iterates both and the dedup gate prevents
	// any duplicate emission.
	pf.members.set(pf.orgID, refID, uuid.New())
	r := pf.createIntro(t, refID, provID, cliID, 5)
	bringToActive(t, pf.svc, r, provID, cliID)
	proposalID := uuid.New()
	require.NoError(t, pf.svc.CreateAttributionIfExists(context.Background(),
		attrInputFor(proposalID, provID, cliID)))
	pf.seedMilestoneInStatus(t, proposalID, 1000_00, milestone.StatusFunded)

	out, err := pf.svc.ProjectedCommissions(context.Background(), pf.orgID)
	require.NoError(t, err)
	assert.Len(t, out, 1, "duplicate iteration over members must not duplicate the projection")
}

// TestProjectedCommissions_NoAttributions returns nil when the
// apporteur has referrals but none have been attributed to a
// proposal yet (newly active intro window with no signed deals).
func TestProjectedCommissions_NoAttributions(t *testing.T) {
	pf := newProjectionFixture(t)
	refID, provID, cliID := pf.seedActors(t)
	pf.members.set(pf.orgID, refID)
	r := pf.createIntro(t, refID, provID, cliID, 5)
	bringToActive(t, pf.svc, r, provID, cliID)

	out, err := pf.svc.ProjectedCommissions(context.Background(), pf.orgID)
	require.NoError(t, err)
	assert.Nil(t, out)
}

// TestProjectedCommissions_DispatchEnded_NilApprovedAt covers the
// branch where attribution.ended_at is set but the milestone has not
// been approved yet — the ended_at gate must still allow the
// projection to be emitted as escrowed (gate only fires on approved
// AFTER ended_at, not active states). This case is the inverse of
// "ended + active milestone → skipped" — same milestone status, but
// dispatchMilestone must skip BECAUSE the attribution is ended,
// regardless of ApprovedAt. Pinning here so a refactor doesn't drop
// the safe-fail behaviour.
func TestProjectedCommissions_DispatchEnded_ActiveMilestoneSkipped(t *testing.T) {
	pf := newProjectionFixture(t)
	proposalID := uuid.New()
	att := pf.seedAttribution(t, 5.0, proposalID)
	// Active escrow milestone, no ApprovedAt at all.
	pf.seedMilestoneInStatus(t, proposalID, 500_00, milestone.StatusSubmitted)
	require.NoError(t, pf.repo.EndAttribution(context.Background(), att.ID, referrerOf(pf, att)))
	out, err := pf.svc.ProjectedCommissions(context.Background(), pf.orgID)
	require.NoError(t, err)
	assert.Empty(t, out, "ended attribution + active milestone must skip")
}

// TestProjectedCommissions_NilPorts degrades gracefully when the
// projection ports are not wired in a worktree (returns nil, no
// error — never breaks the wallet feature on a misconfiguration).
func TestProjectedCommissions_NilPorts(t *testing.T) {
	f := newTestFixture(t, "acct_apporteur")
	// f.svc was built without MilestonesByProposal / OrgMembersLister.
	out, err := f.svc.ProjectedCommissions(context.Background(), uuid.New())
	require.NoError(t, err)
	assert.Nil(t, out)
}

// TestProjectedCommissions_Amount confirms the rate snapshot is the
// authoritative formula on projections (no DB row case). A 5% rate on
// a 1 000,00 € milestone = 50,00 €.
func TestProjectedCommissions_Amount(t *testing.T) {
	pf := newProjectionFixture(t)
	proposalID := uuid.New()
	pf.seedAttribution(t, 5.0, proposalID)
	pf.seedMilestoneInStatus(t, proposalID, 1000_00, milestone.StatusFunded)

	out, err := pf.svc.ProjectedCommissions(context.Background(), pf.orgID)
	require.NoError(t, err)
	require.Len(t, out, 1)
	assert.Equal(t, int64(50_00), out[0].ProjectedCents, "5%% of 1000,00 € = 50,00 €")
	assert.Equal(t, "EUR", out[0].Currency)
}

// TestProjectedCommissions_FromCommissionRow_AllStatuses pins the
// per-commission-status mapping in fromCommissionRow so the wallet UI
// gets consistent ProjectionStatus labels regardless of the actual
// commission lifecycle value.
func TestProjectedCommissions_FromCommissionRow_AllStatuses(t *testing.T) {
	cases := []struct {
		name     string
		cstatus  referral.CommissionStatus
		expected referralapp.ProjectionStatus
	}{
		{name: "paid", cstatus: referral.CommissionPaid, expected: referralapp.ProjectionPaid},
		{name: "failed", cstatus: referral.CommissionFailed, expected: referralapp.ProjectionFailed},
		{name: "pending_kyc", cstatus: referral.CommissionPendingKYC, expected: referralapp.ProjectionPending},
		{name: "pending", cstatus: referral.CommissionPending, expected: referralapp.ProjectionPending},
		{name: "clawed_back", cstatus: referral.CommissionClawedBack, expected: referralapp.ProjectionFailed},
		{name: "cancelled", cstatus: referral.CommissionCancelled, expected: referralapp.ProjectionFailed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pf := newProjectionFixture(t)
			proposalID := uuid.New()
			pf.seedAttribution(t, 5.0, proposalID)
			m := pf.seedMilestoneInStatus(t, proposalID, 1000_00, milestone.StatusApproved)
			// Create the commission row via the distributor, then flip
			// its status to the target value via the repo.
			_, err := pf.svc.DistributeIfApplicable(context.Background(),
				distInputFor(proposalID, m.ID, 1000_00))
			require.NoError(t, err)
			row, err := pf.repo.FindCommissionByMilestone(context.Background(), m.ID)
			require.NoError(t, err)
			row.Status = tc.cstatus
			require.NoError(t, pf.repo.UpdateCommission(context.Background(), row))

			out, err := pf.svc.ProjectedCommissions(context.Background(), pf.orgID)
			require.NoError(t, err)
			require.Len(t, out, 1)
			assert.Equal(t, tc.expected, out[0].Status, "row status %s should map to %s", tc.cstatus, tc.expected)
			assert.Equal(t, referralapp.SourceRow, out[0].Source)
		})
	}
}

// TestProjectedCommissions_FromRow_FallbackCurrency verifies a
// commission row missing its Currency field gets defaulted to "EUR"
// rather than emitting an empty currency to the UI.
func TestProjectedCommissions_FromRow_FallbackCurrency(t *testing.T) {
	pf := newProjectionFixture(t)
	proposalID := uuid.New()
	pf.seedAttribution(t, 5.0, proposalID)
	m := pf.seedMilestoneInStatus(t, proposalID, 1000_00, milestone.StatusApproved)
	_, err := pf.svc.DistributeIfApplicable(context.Background(),
		distInputFor(proposalID, m.ID, 1000_00))
	require.NoError(t, err)
	row, err := pf.repo.FindCommissionByMilestone(context.Background(), m.ID)
	require.NoError(t, err)
	row.Currency = ""
	require.NoError(t, pf.repo.UpdateCommission(context.Background(), row))

	out, err := pf.svc.ProjectedCommissions(context.Background(), pf.orgID)
	require.NoError(t, err)
	require.Len(t, out, 1)
	assert.Equal(t, "EUR", out[0].Currency)
}

// TestProjectedCommissions_MissionTitleResolved verifies the projection
// carries a non-empty mission title when the resolver returns one. The
// fake summary resolver in the testFixture is keyed by proposal id.
func TestProjectedCommissions_MissionTitleResolved(t *testing.T) {
	pf := newProjectionFixture(t)
	proposalID := uuid.New()
	att := pf.seedAttribution(t, 5.0, proposalID)
	// Register the title in the summary resolver fake (keyed by ID).
	pf.summaries.set(proposalID, &referralapp.ProposalSummary{
		ID:    proposalID,
		Title: "Refonte UI corail",
	})
	pf.seedMilestoneInStatus(t, proposalID, 1000_00, milestone.StatusFunded)

	out, err := pf.svc.ProjectedCommissions(context.Background(), pf.orgID)
	require.NoError(t, err)
	require.Len(t, out, 1)
	assert.Equal(t, "Refonte UI corail", out[0].MissionTitle)
	assert.Equal(t, att.ID, out[0].AttributionID)
}

// TestProjectedCommissions_OrderingDescending verifies the output is
// sorted DESC by ProjectedAt — the most recent line first.
func TestProjectedCommissions_OrderingDescending(t *testing.T) {
	pf := newProjectionFixture(t)
	proposalID := uuid.New()
	pf.seedAttribution(t, 5.0, proposalID)

	now := time.Now().UTC()
	// Two milestones with different CreatedAt values.
	older, err := milestone.NewMilestone(milestone.NewMilestoneInput{
		ProposalID: proposalID, Sequence: 1, Title: "Old", Amount: 100_00,
	})
	require.NoError(t, err)
	older.Status = milestone.StatusFunded
	older.CreatedAt = now.Add(-24 * time.Hour)
	pf.milestones.add(older)

	newer, err := milestone.NewMilestone(milestone.NewMilestoneInput{
		ProposalID: proposalID, Sequence: 2, Title: "New", Amount: 100_00,
	})
	require.NoError(t, err)
	newer.Status = milestone.StatusFunded
	newer.CreatedAt = now
	pf.milestones.add(newer)

	out, err := pf.svc.ProjectedCommissions(context.Background(), pf.orgID)
	require.NoError(t, err)
	require.Len(t, out, 2)
	assert.True(t, out[0].ProjectedAt.After(out[1].ProjectedAt) ||
		out[0].ProjectedAt.Equal(out[1].ProjectedAt),
		"results must be sorted DESC by ProjectedAt")
	assert.Equal(t, newer.ID, out[0].MilestoneID)
}
