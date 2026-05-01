package gdpr

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestExportValidate_NilExportFails(t *testing.T) {
	var e *Export
	assert.ErrorIs(t, e.Validate(), ErrEmptyExport)
}

func TestExportValidate_EmptyProfileFails(t *testing.T) {
	e := &Export{
		UserID:    uuid.New(),
		Email:     "u@example.com",
		Timestamp: time.Now(),
	}
	assert.ErrorIs(t, e.Validate(), ErrEmptyExport)
}

func TestExportValidate_OK(t *testing.T) {
	e := &Export{
		UserID:    uuid.New(),
		Email:     "u@example.com",
		Timestamp: time.Now(),
		Profile:   []map[string]any{{"id": "u-1"}},
	}
	assert.NoError(t, e.Validate())
}

func TestExportFileNames_StableOrder(t *testing.T) {
	e := &Export{}
	files := e.FileNames()
	want := []string{
		"profile.json",
		"proposals.json",
		"messages.json",
		"invoices.json",
		"reviews.json",
		"notifications.json",
		"jobs.json",
		"portfolios.json",
		"reports.json",
		"audit_logs.json",
	}
	assert.Equal(t, want, files, "manifest order must be stable across runs")
}

func TestExportSectionFor_ReturnsCorrectSlice(t *testing.T) {
	profile := []map[string]any{{"id": "u-1"}}
	proposals := []map[string]any{{"id": "p-1"}}
	e := &Export{
		Profile:   profile,
		Proposals: proposals,
	}

	assert.Equal(t, profile, e.SectionFor("profile.json"))
	assert.Equal(t, proposals, e.SectionFor("proposals.json"))
	assert.Nil(t, e.SectionFor("unknown.json"))
}

func TestScheduledHardDeleteAt_AddsThirtyDays(t *testing.T) {
	deletedAt := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	got := ScheduledHardDeleteAt(deletedAt)
	want := time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC)
	assert.Equal(t, want, got)
}

func TestIsPurgeable(t *testing.T) {
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	assert.False(t, IsPurgeable(now.Add(-29*24*time.Hour), now), "29 days is too soon")
	assert.True(t, IsPurgeable(now.Add(-30*24*time.Hour), now), "30 days exactly is purgeable")
	assert.True(t, IsPurgeable(now.Add(-60*24*time.Hour), now), "well past 30 days is purgeable")
	assert.False(t, IsPurgeable(time.Time{}, now), "zero deletedAt is not purgeable")
}

func TestOwnerBlockedError_UnwrapsToSentinel(t *testing.T) {
	err := NewOwnerBlockedError([]BlockedOrg{
		{
			OrgID:       uuid.New(),
			OrgName:     "Acme",
			MemberCount: 3,
			Actions:     []RemediationAction{ActionTransferOwnership, ActionDissolveOrg},
		},
	})
	assert.ErrorIs(t, err, ErrOrgOwnerHasMembers)
	assert.Len(t, err.Orgs, 1)
	assert.Equal(t, "Acme", err.Orgs[0].OrgName)
}
