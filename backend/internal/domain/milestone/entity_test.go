package milestone_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/milestone"
)

func validInput() milestone.NewMilestoneInput {
	return milestone.NewMilestoneInput{
		ProposalID:  uuid.New(),
		Sequence:    1,
		Title:       "Design phase",
		Description: "Wireframes + moodboard",
		Amount:      50000,
	}
}

func mustNew(t *testing.T) *milestone.Milestone {
	t.Helper()
	m, err := milestone.NewMilestone(validInput())
	if err != nil {
		t.Fatalf("NewMilestone failed: %v", err)
	}
	return m
}

func TestNewMilestone_Happy(t *testing.T) {
	m, err := milestone.NewMilestone(validInput())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.ID == uuid.Nil {
		t.Error("expected non-nil UUID")
	}
	if m.Status != milestone.StatusPendingFunding {
		t.Errorf("status = %q, want %q", m.Status, milestone.StatusPendingFunding)
	}
	if m.Version != 0 {
		t.Errorf("version = %d, want 0", m.Version)
	}
	if m.CreatedAt.IsZero() || m.UpdatedAt.IsZero() {
		t.Error("timestamps must be set")
	}
}

func TestNewMilestone_Validation(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(in *milestone.NewMilestoneInput)
		wantErr error
	}{
		{"empty title", func(in *milestone.NewMilestoneInput) { in.Title = "" }, milestone.ErrEmptyTitle},
		{"empty description", func(in *milestone.NewMilestoneInput) { in.Description = "" }, milestone.ErrEmptyDescription},
		{"zero amount", func(in *milestone.NewMilestoneInput) { in.Amount = 0 }, milestone.ErrInvalidAmount},
		{"negative amount", func(in *milestone.NewMilestoneInput) { in.Amount = -100 }, milestone.ErrInvalidAmount},
		{"sequence zero", func(in *milestone.NewMilestoneInput) { in.Sequence = 0 }, milestone.ErrInvalidSequence},
		{"sequence negative", func(in *milestone.NewMilestoneInput) { in.Sequence = -5 }, milestone.ErrInvalidSequence},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			in := validInput()
			c.mutate(&in)
			_, err := milestone.NewMilestone(in)
			if !errors.Is(err, c.wantErr) {
				t.Errorf("err = %v, want %v", err, c.wantErr)
			}
		})
	}
}

func TestNewMilestone_NoMinimumAmount(t *testing.T) {
	// The user explicitly chose no minimum amount at the domain level.
	// A 1-centime milestone must be accepted (the credit fraud check
	// lives elsewhere and will simply not award a bonus below 30 EUR).
	in := validInput()
	in.Amount = 1
	if _, err := milestone.NewMilestone(in); err != nil {
		t.Errorf("1-centime milestone should be valid, got %v", err)
	}
}

func TestNewMilestoneBatch_Happy(t *testing.T) {
	propID := uuid.New()
	inputs := []milestone.NewMilestoneInput{
		{Sequence: 1, Title: "A", Description: "a", Amount: 1000},
		{Sequence: 2, Title: "B", Description: "b", Amount: 2000},
		{Sequence: 3, Title: "C", Description: "c", Amount: 3000},
	}
	batch, err := milestone.NewMilestoneBatch(propID, inputs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(batch) != 3 {
		t.Fatalf("len = %d, want 3", len(batch))
	}
	for i, m := range batch {
		if m.ProposalID != propID {
			t.Errorf("milestone[%d].ProposalID = %v, want %v", i, m.ProposalID, propID)
		}
		if m.Sequence != i+1 {
			t.Errorf("milestone[%d].Sequence = %d, want %d", i, m.Sequence, i+1)
		}
		if m.Status != milestone.StatusPendingFunding {
			t.Errorf("milestone[%d].Status = %q, want pending_funding", i, m.Status)
		}
	}
}

func TestNewMilestoneBatch_Empty(t *testing.T) {
	_, err := milestone.NewMilestoneBatch(uuid.New(), nil)
	if !errors.Is(err, milestone.ErrEmptyBatch) {
		t.Errorf("err = %v, want ErrEmptyBatch", err)
	}
}

func TestNewMilestoneBatch_TooMany(t *testing.T) {
	inputs := make([]milestone.NewMilestoneInput, milestone.MaxMilestonesPerProposal+1)
	for i := range inputs {
		inputs[i] = milestone.NewMilestoneInput{
			Sequence: i + 1, Title: "t", Description: "d", Amount: 100,
		}
	}
	_, err := milestone.NewMilestoneBatch(uuid.New(), inputs)
	if !errors.Is(err, milestone.ErrTooManyMilestones) {
		t.Errorf("err = %v, want ErrTooManyMilestones", err)
	}
}

func TestNewMilestoneBatch_ExactlyMax(t *testing.T) {
	inputs := make([]milestone.NewMilestoneInput, milestone.MaxMilestonesPerProposal)
	for i := range inputs {
		inputs[i] = milestone.NewMilestoneInput{
			Sequence: i + 1, Title: "t", Description: "d", Amount: 100,
		}
	}
	batch, err := milestone.NewMilestoneBatch(uuid.New(), inputs)
	if err != nil {
		t.Errorf("20 milestones should be valid, got %v", err)
	}
	if len(batch) != milestone.MaxMilestonesPerProposal {
		t.Errorf("len = %d, want %d", len(batch), milestone.MaxMilestonesPerProposal)
	}
}

func TestNewMilestoneBatch_NonConsecutive(t *testing.T) {
	cases := []struct {
		name   string
		inputs []milestone.NewMilestoneInput
	}{
		{
			"gap",
			[]milestone.NewMilestoneInput{
				{Sequence: 1, Title: "a", Description: "a", Amount: 1},
				{Sequence: 3, Title: "c", Description: "c", Amount: 1},
			},
		},
		{
			"starts-at-2",
			[]milestone.NewMilestoneInput{
				{Sequence: 2, Title: "a", Description: "a", Amount: 1},
				{Sequence: 3, Title: "b", Description: "b", Amount: 1},
			},
		},
		{
			"duplicate",
			[]milestone.NewMilestoneInput{
				{Sequence: 1, Title: "a", Description: "a", Amount: 1},
				{Sequence: 1, Title: "b", Description: "b", Amount: 1},
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := milestone.NewMilestoneBatch(uuid.New(), c.inputs)
			if !errors.Is(err, milestone.ErrNonConsecutiveSequence) {
				t.Errorf("err = %v, want ErrNonConsecutiveSequence", err)
			}
		})
	}
}

func TestNewMilestoneBatch_PropagatesItemError(t *testing.T) {
	inputs := []milestone.NewMilestoneInput{
		{Sequence: 1, Title: "a", Description: "a", Amount: 100},
		{Sequence: 2, Title: "", Description: "d", Amount: 100},
	}
	_, err := milestone.NewMilestoneBatch(uuid.New(), inputs)
	if !errors.Is(err, milestone.ErrEmptyTitle) {
		t.Errorf("err = %v, want ErrEmptyTitle", err)
	}
}

// TestHappyPath walks the full fund -> submit -> approve -> release path
// and verifies each transition sets the matching timestamp and advances the status.
func TestHappyPath(t *testing.T) {
	m := mustNew(t)

	assertStatus := func(want milestone.MilestoneStatus) {
		t.Helper()
		if m.Status != want {
			t.Fatalf("status = %q, want %q", m.Status, want)
		}
	}

	// Fund
	beforeFund := time.Now()
	if err := m.Fund(); err != nil {
		t.Fatal(err)
	}
	assertStatus(milestone.StatusFunded)
	if m.FundedAt == nil || m.FundedAt.Before(beforeFund) {
		t.Error("FundedAt not set properly")
	}

	// Submit
	if err := m.Submit(); err != nil {
		t.Fatal(err)
	}
	assertStatus(milestone.StatusSubmitted)
	if m.SubmittedAt == nil {
		t.Error("SubmittedAt not set")
	}

	// Approve
	if err := m.Approve(); err != nil {
		t.Fatal(err)
	}
	assertStatus(milestone.StatusApproved)
	if m.ApprovedAt == nil {
		t.Error("ApprovedAt not set")
	}

	// Release
	if err := m.Release(); err != nil {
		t.Fatal(err)
	}
	assertStatus(milestone.StatusReleased)
	if m.ReleasedAt == nil {
		t.Error("ReleasedAt not set")
	}
	if !m.IsTerminal() {
		t.Error("released milestone should be terminal")
	}
}

func TestFund_Invalid(t *testing.T) {
	for _, s := range []milestone.MilestoneStatus{
		milestone.StatusFunded, milestone.StatusSubmitted, milestone.StatusApproved,
		milestone.StatusReleased, milestone.StatusDisputed, milestone.StatusCancelled,
		milestone.StatusRefunded,
	} {
		t.Run(string(s), func(t *testing.T) {
			m := mustNew(t)
			m.Status = s
			if err := m.Fund(); !errors.Is(err, milestone.ErrInvalidStatus) {
				t.Errorf("err = %v, want ErrInvalidStatus", err)
			}
		})
	}
}

func TestSubmit_Invalid(t *testing.T) {
	for _, s := range []milestone.MilestoneStatus{
		milestone.StatusPendingFunding, milestone.StatusSubmitted, milestone.StatusApproved,
		milestone.StatusReleased, milestone.StatusDisputed, milestone.StatusCancelled,
		milestone.StatusRefunded,
	} {
		t.Run(string(s), func(t *testing.T) {
			m := mustNew(t)
			m.Status = s
			if err := m.Submit(); !errors.Is(err, milestone.ErrInvalidStatus) {
				t.Errorf("err = %v, want ErrInvalidStatus", err)
			}
		})
	}
}

func TestApprove_Invalid(t *testing.T) {
	for _, s := range []milestone.MilestoneStatus{
		milestone.StatusPendingFunding, milestone.StatusFunded, milestone.StatusApproved,
		milestone.StatusReleased, milestone.StatusDisputed, milestone.StatusCancelled,
		milestone.StatusRefunded,
	} {
		t.Run(string(s), func(t *testing.T) {
			m := mustNew(t)
			m.Status = s
			if err := m.Approve(); !errors.Is(err, milestone.ErrInvalidStatus) {
				t.Errorf("err = %v, want ErrInvalidStatus", err)
			}
		})
	}
}

func TestReject_ResetsSubmittedAt(t *testing.T) {
	m := mustNew(t)
	_ = m.Fund()
	_ = m.Submit()
	if m.SubmittedAt == nil {
		t.Fatal("setup: SubmittedAt should be set")
	}
	if err := m.Reject(); err != nil {
		t.Fatal(err)
	}
	if m.Status != milestone.StatusFunded {
		t.Errorf("status = %q, want funded", m.Status)
	}
	if m.SubmittedAt != nil {
		t.Error("Reject must clear SubmittedAt so the next submit restarts the timer")
	}

	// Second submit should set a new timestamp and work normally.
	if err := m.Submit(); err != nil {
		t.Fatal(err)
	}
	if m.SubmittedAt == nil {
		t.Error("resubmit must set SubmittedAt")
	}
}

func TestReject_InvalidFrom(t *testing.T) {
	for _, s := range []milestone.MilestoneStatus{
		milestone.StatusPendingFunding, milestone.StatusFunded, milestone.StatusApproved,
		milestone.StatusReleased, milestone.StatusDisputed, milestone.StatusCancelled,
		milestone.StatusRefunded,
	} {
		t.Run(string(s), func(t *testing.T) {
			m := mustNew(t)
			m.Status = s
			if err := m.Reject(); !errors.Is(err, milestone.ErrInvalidStatus) {
				t.Errorf("err = %v, want ErrInvalidStatus", err)
			}
		})
	}
}

func TestRelease_InvalidFrom(t *testing.T) {
	for _, s := range []milestone.MilestoneStatus{
		milestone.StatusPendingFunding, milestone.StatusFunded, milestone.StatusSubmitted,
		milestone.StatusReleased, milestone.StatusDisputed, milestone.StatusCancelled,
		milestone.StatusRefunded,
	} {
		t.Run(string(s), func(t *testing.T) {
			m := mustNew(t)
			m.Status = s
			if err := m.Release(); !errors.Is(err, milestone.ErrInvalidStatus) {
				t.Errorf("err = %v, want ErrInvalidStatus", err)
			}
		})
	}
}

func TestOpenDispute_FromFunded(t *testing.T) {
	m := mustNew(t)
	_ = m.Fund()
	did := uuid.New()
	if err := m.OpenDispute(did); err != nil {
		t.Fatal(err)
	}
	if m.Status != milestone.StatusDisputed {
		t.Errorf("status = %q, want disputed", m.Status)
	}
	if m.ActiveDisputeID == nil || *m.ActiveDisputeID != did {
		t.Error("ActiveDisputeID not set")
	}
	if m.LastDisputeID == nil || *m.LastDisputeID != did {
		t.Error("LastDisputeID not set")
	}
	if m.DisputedAt == nil {
		t.Error("DisputedAt not set")
	}
}

func TestOpenDispute_FromSubmitted(t *testing.T) {
	m := mustNew(t)
	_ = m.Fund()
	_ = m.Submit()
	if err := m.OpenDispute(uuid.New()); err != nil {
		t.Fatal(err)
	}
	if m.Status != milestone.StatusDisputed {
		t.Errorf("status = %q, want disputed", m.Status)
	}
}

func TestOpenDispute_InvalidFrom(t *testing.T) {
	for _, s := range []milestone.MilestoneStatus{
		milestone.StatusPendingFunding, milestone.StatusApproved, milestone.StatusReleased,
		milestone.StatusDisputed, milestone.StatusCancelled, milestone.StatusRefunded,
	} {
		t.Run(string(s), func(t *testing.T) {
			m := mustNew(t)
			m.Status = s
			if err := m.OpenDispute(uuid.New()); !errors.Is(err, milestone.ErrInvalidStatus) {
				t.Errorf("err = %v, want ErrInvalidStatus", err)
			}
		})
	}
}

func TestRestoreFromDispute_AllValidTargets(t *testing.T) {
	targets := []milestone.MilestoneStatus{
		milestone.StatusFunded,
		milestone.StatusReleased,
		milestone.StatusRefunded,
	}
	for _, target := range targets {
		t.Run(string(target), func(t *testing.T) {
			m := mustNew(t)
			_ = m.Fund()
			did := uuid.New()
			_ = m.OpenDispute(did)

			if err := m.RestoreFromDispute(target); err != nil {
				t.Fatalf("err = %v", err)
			}
			if m.Status != target {
				t.Errorf("status = %q, want %q", m.Status, target)
			}
			if m.ActiveDisputeID != nil {
				t.Error("ActiveDisputeID should be cleared after restore")
			}
			if m.LastDisputeID == nil || *m.LastDisputeID != did {
				t.Error("LastDisputeID must be preserved after restore")
			}
			if target == milestone.StatusReleased && m.ReleasedAt == nil {
				t.Error("ReleasedAt must be set when restoring to released")
			}
		})
	}
}

func TestRestoreFromDispute_InvalidTarget(t *testing.T) {
	m := mustNew(t)
	_ = m.Fund()
	_ = m.OpenDispute(uuid.New())

	for _, target := range []milestone.MilestoneStatus{
		milestone.StatusPendingFunding,
		milestone.StatusSubmitted,
		milestone.StatusApproved,
		milestone.StatusDisputed,
		milestone.StatusCancelled,
	} {
		t.Run(string(target), func(t *testing.T) {
			mm := *m
			if err := mm.RestoreFromDispute(target); !errors.Is(err, milestone.ErrInvalidRestoreTarget) {
				t.Errorf("err = %v, want ErrInvalidRestoreTarget", err)
			}
		})
	}
}

func TestRestoreFromDispute_FromNonDisputed(t *testing.T) {
	m := mustNew(t)
	// Still in pending_funding
	if err := m.RestoreFromDispute(milestone.StatusFunded); !errors.Is(err, milestone.ErrInvalidStatus) {
		t.Errorf("err = %v, want ErrInvalidStatus", err)
	}
}

func TestCancel_FromPendingFunding(t *testing.T) {
	m := mustNew(t)
	if err := m.Cancel(); err != nil {
		t.Fatal(err)
	}
	if m.Status != milestone.StatusCancelled {
		t.Errorf("status = %q, want cancelled", m.Status)
	}
	if m.CancelledAt == nil {
		t.Error("CancelledAt not set")
	}
	if !m.IsTerminal() {
		t.Error("cancelled milestone should be terminal")
	}
}

func TestCancel_InvalidFrom(t *testing.T) {
	for _, s := range []milestone.MilestoneStatus{
		milestone.StatusFunded, milestone.StatusSubmitted, milestone.StatusApproved,
		milestone.StatusReleased, milestone.StatusDisputed, milestone.StatusCancelled,
		milestone.StatusRefunded,
	} {
		t.Run(string(s), func(t *testing.T) {
			m := mustNew(t)
			m.Status = s
			if err := m.Cancel(); !errors.Is(err, milestone.ErrInvalidStatus) {
				t.Errorf("err = %v, want ErrInvalidStatus", err)
			}
		})
	}
}

func TestSumAmount(t *testing.T) {
	m1 := mustNew(t)
	m1.Amount = 1000
	m2 := mustNew(t)
	m2.Amount = 2500
	m3 := mustNew(t)
	m3.Amount = 500

	if got := milestone.SumAmount([]*milestone.Milestone{m1, m2, m3}); got != 4000 {
		t.Errorf("SumAmount = %d, want 4000", got)
	}
	if got := milestone.SumAmount(nil); got != 0 {
		t.Errorf("SumAmount(nil) = %d, want 0", got)
	}
}

func TestFindCurrentActive(t *testing.T) {
	mk := func(seq int, status milestone.MilestoneStatus) *milestone.Milestone {
		m := mustNew(t)
		m.Sequence = seq
		m.Status = status
		return m
	}

	t.Run("released+released+pending returns pending", func(t *testing.T) {
		milestones := []*milestone.Milestone{
			mk(1, milestone.StatusReleased),
			mk(2, milestone.StatusReleased),
			mk(3, milestone.StatusPendingFunding),
		}
		current := milestone.FindCurrentActive(milestones)
		if current == nil || current.Sequence != 3 {
			t.Errorf("expected sequence 3, got %+v", current)
		}
	})

	t.Run("picks lowest non-terminal sequence", func(t *testing.T) {
		milestones := []*milestone.Milestone{
			mk(3, milestone.StatusPendingFunding),
			mk(1, milestone.StatusReleased),
			mk(2, milestone.StatusFunded),
		}
		current := milestone.FindCurrentActive(milestones)
		if current == nil || current.Sequence != 2 {
			t.Errorf("expected sequence 2, got %+v", current)
		}
	})

	t.Run("all terminal returns nil", func(t *testing.T) {
		milestones := []*milestone.Milestone{
			mk(1, milestone.StatusReleased),
			mk(2, milestone.StatusReleased),
		}
		if milestone.FindCurrentActive(milestones) != nil {
			t.Error("expected nil for all-terminal batch")
		}
	})

	t.Run("mix with cancelled", func(t *testing.T) {
		milestones := []*milestone.Milestone{
			mk(1, milestone.StatusReleased),
			mk(2, milestone.StatusCancelled),
			mk(3, milestone.StatusFunded),
		}
		current := milestone.FindCurrentActive(milestones)
		if current == nil || current.Sequence != 3 {
			t.Errorf("expected sequence 3, got %+v", current)
		}
	})
}

func TestAllReleased(t *testing.T) {
	m := mustNew(t)
	m.Status = milestone.StatusReleased
	n := mustNew(t)
	n.Status = milestone.StatusReleased

	if !milestone.AllReleased([]*milestone.Milestone{m, n}) {
		t.Error("expected true for all-released batch")
	}

	n.Status = milestone.StatusFunded
	if milestone.AllReleased([]*milestone.Milestone{m, n}) {
		t.Error("expected false when one is not released")
	}

	if milestone.AllReleased(nil) {
		t.Error("empty batch must not be considered fully released")
	}
}

func TestAnyFunded(t *testing.T) {
	mk := func(s milestone.MilestoneStatus) *milestone.Milestone {
		m := mustNew(t)
		m.Status = s
		return m
	}

	t.Run("none", func(t *testing.T) {
		if milestone.AnyFunded([]*milestone.Milestone{
			mk(milestone.StatusPendingFunding),
			mk(milestone.StatusCancelled),
		}) {
			t.Error("expected false")
		}
	})

	for _, s := range []milestone.MilestoneStatus{
		milestone.StatusFunded, milestone.StatusSubmitted,
		milestone.StatusApproved, milestone.StatusReleased,
		milestone.StatusDisputed, milestone.StatusRefunded,
	} {
		t.Run(string(s), func(t *testing.T) {
			if !milestone.AnyFunded([]*milestone.Milestone{mk(s)}) {
				t.Errorf("expected true for %q", s)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Deadline ordering tests (bug fix: milestone N+1 must come strictly after N)
// ---------------------------------------------------------------------------

// dl is a tiny helper to build a *time.Time for a given YYYY-MM-DD.
func dl(t *testing.T, ymd string) *time.Time {
	t.Helper()
	parsed, err := time.Parse("2006-01-02", ymd)
	if err != nil {
		t.Fatalf("dl(%q): %v", ymd, err)
	}
	return &parsed
}

func TestValidateMilestoneDeadlineOrder_TableDriven(t *testing.T) {
	cases := []struct {
		name    string
		inputs  []milestone.NewMilestoneInput
		wantErr error
	}{
		{
			name:    "empty list is allowed",
			inputs:  nil,
			wantErr: nil,
		},
		{
			name: "single milestone is allowed",
			inputs: []milestone.NewMilestoneInput{
				{Sequence: 1, Title: "a", Description: "a", Amount: 100, Deadline: dl(t, "2026-05-07")},
			},
			wantErr: nil,
		},
		{
			name: "single milestone without deadline is allowed",
			inputs: []milestone.NewMilestoneInput{
				{Sequence: 1, Title: "a", Description: "a", Amount: 100},
			},
			wantErr: nil,
		},
		{
			name: "valid increasing sequence",
			inputs: []milestone.NewMilestoneInput{
				{Sequence: 1, Title: "a", Description: "a", Amount: 100, Deadline: dl(t, "2026-05-07")},
				{Sequence: 2, Title: "b", Description: "b", Amount: 100, Deadline: dl(t, "2026-05-14")},
				{Sequence: 3, Title: "c", Description: "c", Amount: 100, Deadline: dl(t, "2026-05-28")},
			},
			wantErr: nil,
		},
		{
			name: "two milestones on the same day are rejected (strict-after)",
			inputs: []milestone.NewMilestoneInput{
				{Sequence: 1, Title: "a", Description: "a", Amount: 100, Deadline: dl(t, "2026-05-07")},
				{Sequence: 2, Title: "b", Description: "b", Amount: 100, Deadline: dl(t, "2026-05-07")},
			},
			wantErr: milestone.ErrMilestonesNotSequential,
		},
		{
			name: "out-of-order pair (the reported bug)",
			inputs: []milestone.NewMilestoneInput{
				{Sequence: 1, Title: "a", Description: "a", Amount: 100, Deadline: dl(t, "2026-05-07")},
				{Sequence: 2, Title: "b", Description: "b", Amount: 100, Deadline: dl(t, "2026-05-06")},
			},
			wantErr: milestone.ErrMilestonesNotSequential,
		},
		{
			name: "three milestones with middle out of order",
			inputs: []milestone.NewMilestoneInput{
				{Sequence: 1, Title: "a", Description: "a", Amount: 100, Deadline: dl(t, "2026-05-01")},
				{Sequence: 2, Title: "b", Description: "b", Amount: 100, Deadline: dl(t, "2026-04-25")},
				{Sequence: 3, Title: "c", Description: "c", Amount: 100, Deadline: dl(t, "2026-05-10")},
			},
			wantErr: milestone.ErrMilestonesNotSequential,
		},
		{
			name: "nil deadlines are skipped, the set ones must still be ordered",
			inputs: []milestone.NewMilestoneInput{
				{Sequence: 1, Title: "a", Description: "a", Amount: 100, Deadline: dl(t, "2026-05-01")},
				{Sequence: 2, Title: "b", Description: "b", Amount: 100, Deadline: nil},
				{Sequence: 3, Title: "c", Description: "c", Amount: 100, Deadline: dl(t, "2026-05-10")},
			},
			wantErr: nil,
		},
		{
			name: "all nil deadlines pass",
			inputs: []milestone.NewMilestoneInput{
				{Sequence: 1, Title: "a", Description: "a", Amount: 100},
				{Sequence: 2, Title: "b", Description: "b", Amount: 100},
				{Sequence: 3, Title: "c", Description: "c", Amount: 100},
			},
			wantErr: nil,
		},
		{
			name: "ordering enforced even when caller passes inputs out of sequence order",
			inputs: []milestone.NewMilestoneInput{
				{Sequence: 2, Title: "b", Description: "b", Amount: 100, Deadline: dl(t, "2026-05-06")},
				{Sequence: 1, Title: "a", Description: "a", Amount: 100, Deadline: dl(t, "2026-05-07")},
			},
			wantErr: milestone.ErrMilestonesNotSequential,
		},
		{
			name: "sparse deadlines: nil between two strictly-decreasing should still fail",
			inputs: []milestone.NewMilestoneInput{
				{Sequence: 1, Title: "a", Description: "a", Amount: 100, Deadline: dl(t, "2026-05-10")},
				{Sequence: 2, Title: "b", Description: "b", Amount: 100, Deadline: nil},
				{Sequence: 3, Title: "c", Description: "c", Amount: 100, Deadline: dl(t, "2026-05-05")},
			},
			wantErr: milestone.ErrMilestonesNotSequential,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := milestone.ValidateMilestoneDeadlineOrder(c.inputs)
			if c.wantErr == nil {
				if err != nil {
					t.Errorf("err = %v, want nil", err)
				}
				return
			}
			if !errors.Is(err, c.wantErr) {
				t.Errorf("err = %v, want %v", err, c.wantErr)
			}
		})
	}
}

func TestNewMilestoneBatch_RejectsBadDeadlineOrder(t *testing.T) {
	// Verify the validation is wired through NewMilestoneBatch — the
	// realistic call site that the proposal app service uses.
	inputs := []milestone.NewMilestoneInput{
		{Sequence: 1, Title: "a", Description: "a", Amount: 100, Deadline: dl(t, "2026-05-07")},
		{Sequence: 2, Title: "b", Description: "b", Amount: 100, Deadline: dl(t, "2026-05-06")},
	}
	_, err := milestone.NewMilestoneBatch(uuid.New(), inputs)
	if !errors.Is(err, milestone.ErrMilestonesNotSequential) {
		t.Errorf("err = %v, want ErrMilestonesNotSequential", err)
	}
}

func TestValidateMilestonesAgainstProjectDeadline(t *testing.T) {
	cases := []struct {
		name        string
		inputs      []milestone.NewMilestoneInput
		projectDead *time.Time
		wantErr     error
	}{
		{
			name:        "no project deadline -> always passes",
			inputs:      []milestone.NewMilestoneInput{{Sequence: 1, Deadline: dl(t, "2030-01-01")}},
			projectDead: nil,
			wantErr:     nil,
		},
		{
			name: "all milestones before project deadline pass",
			inputs: []milestone.NewMilestoneInput{
				{Sequence: 1, Deadline: dl(t, "2026-05-07")},
				{Sequence: 2, Deadline: dl(t, "2026-05-14")},
			},
			projectDead: dl(t, "2026-06-01"),
			wantErr:     nil,
		},
		{
			name: "milestone equal to project deadline is allowed",
			inputs: []milestone.NewMilestoneInput{
				{Sequence: 1, Deadline: dl(t, "2026-05-07")},
				{Sequence: 2, Deadline: dl(t, "2026-06-01")},
			},
			projectDead: dl(t, "2026-06-01"),
			wantErr:     nil,
		},
		{
			name: "milestone after project deadline is rejected",
			inputs: []milestone.NewMilestoneInput{
				{Sequence: 1, Deadline: dl(t, "2026-05-07")},
				{Sequence: 2, Deadline: dl(t, "2026-07-01")},
			},
			projectDead: dl(t, "2026-06-01"),
			wantErr:     milestone.ErrMilestoneDeadlineAfterProject,
		},
		{
			name: "milestone with nil deadline is skipped",
			inputs: []milestone.NewMilestoneInput{
				{Sequence: 1, Deadline: nil},
				{Sequence: 2, Deadline: dl(t, "2026-05-30")},
			},
			projectDead: dl(t, "2026-06-01"),
			wantErr:     nil,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := milestone.ValidateMilestonesAgainstProjectDeadline(c.inputs, c.projectDead)
			if c.wantErr == nil {
				if err != nil {
					t.Errorf("err = %v, want nil", err)
				}
				return
			}
			if !errors.Is(err, c.wantErr) {
				t.Errorf("err = %v, want %v", err, c.wantErr)
			}
		})
	}
}
