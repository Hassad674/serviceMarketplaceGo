package milestone_test

import (
	"testing"

	"marketplace-backend/internal/domain/milestone"
)

func TestMilestoneStatus_IsValid(t *testing.T) {
	cases := []struct {
		status milestone.MilestoneStatus
		want   bool
	}{
		{milestone.StatusPendingFunding, true},
		{milestone.StatusFunded, true},
		{milestone.StatusSubmitted, true},
		{milestone.StatusApproved, true},
		{milestone.StatusReleased, true},
		{milestone.StatusDisputed, true},
		{milestone.StatusCancelled, true},
		{milestone.StatusRefunded, true},
		{"", false},
		{"unknown", false},
		{"COMPLETED", false},
	}
	for _, c := range cases {
		if got := c.status.IsValid(); got != c.want {
			t.Errorf("IsValid(%q) = %v, want %v", c.status, got, c.want)
		}
	}
}

func TestMilestoneStatus_IsTerminal(t *testing.T) {
	terminals := []milestone.MilestoneStatus{
		milestone.StatusReleased,
		milestone.StatusCancelled,
		milestone.StatusRefunded,
	}
	nonTerminals := []milestone.MilestoneStatus{
		milestone.StatusPendingFunding,
		milestone.StatusFunded,
		milestone.StatusSubmitted,
		milestone.StatusApproved,
		milestone.StatusDisputed,
	}
	for _, s := range terminals {
		if !s.IsTerminal() {
			t.Errorf("IsTerminal(%q) = false, want true", s)
		}
	}
	for _, s := range nonTerminals {
		if s.IsTerminal() {
			t.Errorf("IsTerminal(%q) = true, want false", s)
		}
	}
}

func TestMilestoneStatus_IsActive(t *testing.T) {
	active := []milestone.MilestoneStatus{
		milestone.StatusFunded,
		milestone.StatusSubmitted,
		milestone.StatusApproved,
		milestone.StatusDisputed,
	}
	inactive := []milestone.MilestoneStatus{
		milestone.StatusPendingFunding,
		milestone.StatusReleased,
		milestone.StatusCancelled,
		milestone.StatusRefunded,
	}
	for _, s := range active {
		if !s.IsActive() {
			t.Errorf("IsActive(%q) = false, want true", s)
		}
	}
	for _, s := range inactive {
		if s.IsActive() {
			t.Errorf("IsActive(%q) = true, want false", s)
		}
	}
}

// TestCanTransitionTo_Exhaustive enforces the full transition table by
// walking every (from, to) pair and comparing against the legal set below.
// If you change the state machine you MUST update both the code and this
// table, otherwise this test will catch the drift.
func TestCanTransitionTo_Exhaustive(t *testing.T) {
	allStatuses := []milestone.MilestoneStatus{
		milestone.StatusPendingFunding,
		milestone.StatusFunded,
		milestone.StatusSubmitted,
		milestone.StatusApproved,
		milestone.StatusReleased,
		milestone.StatusDisputed,
		milestone.StatusCancelled,
		milestone.StatusRefunded,
	}

	// legal is the canonical truth: from -> set of allowed targets.
	legal := map[milestone.MilestoneStatus]map[milestone.MilestoneStatus]bool{
		milestone.StatusPendingFunding: {
			milestone.StatusFunded:    true,
			milestone.StatusCancelled: true,
		},
		milestone.StatusFunded: {
			milestone.StatusSubmitted: true,
			milestone.StatusDisputed:  true,
		},
		milestone.StatusSubmitted: {
			milestone.StatusApproved: true,
			milestone.StatusFunded:   true,
			milestone.StatusDisputed: true,
		},
		milestone.StatusApproved: {
			milestone.StatusReleased: true,
		},
		milestone.StatusDisputed: {
			milestone.StatusFunded:   true,
			milestone.StatusReleased: true,
			milestone.StatusRefunded: true,
		},
		// Terminal states: no outgoing transitions.
		milestone.StatusReleased:  {},
		milestone.StatusCancelled: {},
		milestone.StatusRefunded:  {},
	}

	for _, from := range allStatuses {
		for _, to := range allStatuses {
			want := legal[from][to]
			got := from.CanTransitionTo(to)
			if got != want {
				t.Errorf("CanTransitionTo: from=%q to=%q got=%v want=%v", from, to, got, want)
			}
		}
	}
}
