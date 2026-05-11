package referral

// Tests for the pure helper functions in wallet_reader_projections.go.
// Lives in the `referral` (NOT `referral_test`) package so it can hit
// unexported helpers directly — the orchestrator and integration tests
// live in referral_test/wallet_reader_projections_test.go.

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"marketplace-backend/internal/domain/milestone"
	"marketplace-backend/internal/domain/referral"
)

// TestBuildProjections_NilAttributionSkipped pins the defensive
// nil-guard on attributions slice elements.
func TestBuildProjections_NilAttributionSkipped(t *testing.T) {
	out := buildProjections([]*referral.Attribution{nil}, nil, nil, nil)
	assert.Empty(t, out)
}

// TestBuildProjections_NilMilestoneSkipped — same for milestones.
func TestBuildProjections_NilMilestoneSkipped(t *testing.T) {
	proposalID := uuid.New()
	att := &referral.Attribution{
		ID:              uuid.New(),
		ProposalID:      proposalID,
		RatePctSnapshot: 5.0,
	}
	milestones := map[uuid.UUID][]*milestone.Milestone{
		proposalID: {nil},
	}
	out := buildProjections([]*referral.Attribution{att}, milestones, nil, nil)
	assert.Empty(t, out)
}

// TestProposalIDsOf_NilAndDedupe pins the empty-input + nil-element
// + dedupe paths in one go.
func TestProposalIDsOf_NilAndDedupe(t *testing.T) {
	pid := uuid.New()
	atts := []*referral.Attribution{
		nil,
		{ProposalID: pid},
		{ProposalID: pid}, // duplicate
	}
	out := proposalIDsOf(atts)
	assert.Equal(t, []uuid.UUID{pid}, out)
}

// TestReferralIDsOf_NilSkipped pins the defensive nil-guard.
func TestReferralIDsOf_NilSkipped(t *testing.T) {
	rid := uuid.New()
	refs := []*referral.Referral{
		nil,
		{ID: rid},
	}
	out := referralIDsOf(refs)
	assert.Equal(t, []uuid.UUID{rid}, out)
}

// TestTimeOrNow_PrimaryNil falls back when primary is nil.
func TestTimeOrNow_PrimaryNil(t *testing.T) {
	fallback := time.Now()
	got := timeOrNow(nil, fallback)
	assert.Equal(t, fallback, got)
}

// TestTimeOrNow_PrimaryNonNil prefers primary over fallback.
func TestTimeOrNow_PrimaryNonNil(t *testing.T) {
	primary := time.Now().Add(-time.Hour)
	fallback := time.Now()
	got := timeOrNow(&primary, fallback)
	assert.Equal(t, primary, got)
}

// TestProjectAmount_Math pins the formula across a range of inputs so
// a refactor of the divisor breaks the test loudly.
func TestProjectAmount_Math(t *testing.T) {
	cases := []struct {
		amount int64
		rate   float64
		want   int64
	}{
		{amount: 0, rate: 5.0, want: 0},
		{amount: 100_00, rate: 5.0, want: 5_00},
		{amount: 1000_00, rate: 5.0, want: 50_00},
		{amount: 1000_00, rate: 0.0, want: 0},
		{amount: 1000_00, rate: 10.5, want: 105_00},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, projectAmount(c.amount, c.rate),
			"amount=%d rate=%f", c.amount, c.rate)
	}
}

// TestDispatchMilestone_UnknownStatusSkipped pins the fall-through
// case in dispatchMilestone — a status outside the explicit switch
// arms must SKIP rather than emit garbage.
func TestDispatchMilestone_UnknownStatusSkipped(t *testing.T) {
	a := &referral.Attribution{ID: uuid.New(), RatePctSnapshot: 5.0}
	m := &milestone.Milestone{
		ID:     uuid.New(),
		Status: milestone.MilestoneStatus("unknown_status"),
		Amount: 100_00,
	}
	_, ok := dispatchMilestone(a, m, nil, "")
	assert.False(t, ok, "unknown milestone status must SKIP, never emit")
}
