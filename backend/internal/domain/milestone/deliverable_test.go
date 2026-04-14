package milestone_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/milestone"
)

func validDeliverableInput() milestone.NewDeliverableInput {
	return milestone.NewDeliverableInput{
		MilestoneID: uuid.New(),
		Filename:    "spec.pdf",
		URL:         "https://cdn.example.com/deliv/spec.pdf",
		Size:        12345,
		MimeType:    "application/pdf",
		UploadedBy:  uuid.New(),
	}
}

func TestNewDeliverable_Happy(t *testing.T) {
	d, err := milestone.NewDeliverable(validDeliverableInput())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.ID == uuid.Nil {
		t.Error("expected non-nil UUID")
	}
	if d.CreatedAt.IsZero() {
		t.Error("CreatedAt must be set")
	}
}

func TestNewDeliverable_Validation(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*milestone.NewDeliverableInput)
		wantErr error
	}{
		{"empty filename", func(in *milestone.NewDeliverableInput) { in.Filename = "" }, milestone.ErrEmptyTitle},
		{"empty url", func(in *milestone.NewDeliverableInput) { in.URL = "" }, milestone.ErrEmptyDescription},
		{"zero size", func(in *milestone.NewDeliverableInput) { in.Size = 0 }, milestone.ErrInvalidAmount},
		{"negative size", func(in *milestone.NewDeliverableInput) { in.Size = -1 }, milestone.ErrInvalidAmount},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			in := validDeliverableInput()
			c.mutate(&in)
			_, err := milestone.NewDeliverable(in)
			if !errors.Is(err, c.wantErr) {
				t.Errorf("err = %v, want %v", err, c.wantErr)
			}
		})
	}
}

func TestIsMutableStatus(t *testing.T) {
	mutable := []milestone.MilestoneStatus{
		milestone.StatusPendingFunding,
		milestone.StatusFunded,
	}
	immutable := []milestone.MilestoneStatus{
		milestone.StatusSubmitted,
		milestone.StatusApproved,
		milestone.StatusReleased,
		milestone.StatusDisputed,
		milestone.StatusCancelled,
		milestone.StatusRefunded,
	}
	for _, s := range mutable {
		if !milestone.IsMutableStatus(s) {
			t.Errorf("IsMutableStatus(%q) = false, want true", s)
		}
	}
	for _, s := range immutable {
		if milestone.IsMutableStatus(s) {
			t.Errorf("IsMutableStatus(%q) = true, want false", s)
		}
	}
}
