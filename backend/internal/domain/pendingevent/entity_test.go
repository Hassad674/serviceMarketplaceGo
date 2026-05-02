package pendingevent_test

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"marketplace-backend/internal/domain/pendingevent"
)

func validInput() pendingevent.NewPendingEventInput {
	payload, _ := json.Marshal(map[string]string{"milestone_id": "abc"})
	return pendingevent.NewPendingEventInput{
		EventType: pendingevent.TypeMilestoneAutoApprove,
		Payload:   payload,
		FiresAt:   time.Now().Add(7 * 24 * time.Hour),
	}
}

func TestEventType_IsValid(t *testing.T) {
	cases := []struct {
		t    pendingevent.EventType
		want bool
	}{
		{pendingevent.TypeMilestoneAutoApprove, true},
		{pendingevent.TypeMilestoneFundReminder, true},
		{pendingevent.TypeProposalAutoClose, true},
		{pendingevent.TypeStripeTransfer, true},
		{pendingevent.TypeSearchReindex, true},
		{pendingevent.TypeSearchDelete, true},
		{pendingevent.TypeStripeWebhook, true},
		{"", false},
		{"unknown", false},
	}
	for _, c := range cases {
		if got := c.t.IsValid(); got != c.want {
			t.Errorf("IsValid(%q) = %v, want %v", c.t, got, c.want)
		}
	}
}

func TestStatus_IsValid(t *testing.T) {
	cases := []struct {
		s    pendingevent.Status
		want bool
	}{
		{pendingevent.StatusPending, true},
		{pendingevent.StatusProcessing, true},
		{pendingevent.StatusDone, true},
		{pendingevent.StatusFailed, true},
		{"", false},
		{"unknown", false},
	}
	for _, c := range cases {
		if got := c.s.IsValid(); got != c.want {
			t.Errorf("IsValid(%q) = %v, want %v", c.s, got, c.want)
		}
	}
}

func TestNewPendingEvent_Happy(t *testing.T) {
	e, err := pendingevent.NewPendingEvent(validInput())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if e.ID.String() == "00000000-0000-0000-0000-000000000000" {
		t.Error("ID should be non-nil")
	}
	if e.Status != pendingevent.StatusPending {
		t.Errorf("status = %q, want pending", e.Status)
	}
	if e.Attempts != 0 {
		t.Errorf("attempts = %d, want 0", e.Attempts)
	}
}

// TestNewPendingEvent_StripeWebhookHappy covers the new TypeStripeWebhook
// path: a valid event_id MUST round-trip into the entity so the
// repository can persist it on the partial unique index. Without
// this guard the constructor could silently drop the id and break
// dedup at the database layer.
func TestNewPendingEvent_StripeWebhookHappy(t *testing.T) {
	in := validInput()
	in.EventType = pendingevent.TypeStripeWebhook
	in.StripeEventID = "evt_1Q9Zxq2eZvKYlo2C123456"
	e, err := pendingevent.NewPendingEvent(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if e.StripeEventID != in.StripeEventID {
		t.Errorf("StripeEventID = %q, want %q", e.StripeEventID, in.StripeEventID)
	}
	if e.EventType != pendingevent.TypeStripeWebhook {
		t.Errorf("EventType = %q, want %q", e.EventType, pendingevent.TypeStripeWebhook)
	}
}

func TestNewPendingEvent_Validation(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*pendingevent.NewPendingEventInput)
		wantErr error
	}{
		{"invalid type", func(in *pendingevent.NewPendingEventInput) { in.EventType = "bogus" }, pendingevent.ErrInvalidEventType},
		{"empty payload", func(in *pendingevent.NewPendingEventInput) { in.Payload = nil }, pendingevent.ErrEmptyPayload},
		{"zero fires_at", func(in *pendingevent.NewPendingEventInput) { in.FiresAt = time.Time{} }, pendingevent.ErrZeroFiresAt},
		{"stripe webhook without event id", func(in *pendingevent.NewPendingEventInput) {
			in.EventType = pendingevent.TypeStripeWebhook
			in.StripeEventID = ""
		}, pendingevent.ErrMissingStripeEventID},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			in := validInput()
			c.mutate(&in)
			_, err := pendingevent.NewPendingEvent(in)
			if !errors.Is(err, c.wantErr) {
				t.Errorf("err = %v, want %v", err, c.wantErr)
			}
		})
	}
}

func TestMarkProcessing_FromPending(t *testing.T) {
	e, _ := pendingevent.NewPendingEvent(validInput())
	if err := e.MarkProcessing(); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if e.Status != pendingevent.StatusProcessing {
		t.Errorf("status = %q, want processing", e.Status)
	}
	if e.Attempts != 1 {
		t.Errorf("attempts = %d, want 1", e.Attempts)
	}
}

func TestMarkProcessing_FromFailed(t *testing.T) {
	e, _ := pendingevent.NewPendingEvent(validInput())
	e.Status = pendingevent.StatusFailed
	e.Attempts = 2
	if err := e.MarkProcessing(); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if e.Status != pendingevent.StatusProcessing {
		t.Errorf("status = %q, want processing", e.Status)
	}
	if e.Attempts != 3 {
		t.Errorf("attempts = %d, want 3", e.Attempts)
	}
}

func TestMarkProcessing_InvalidFrom(t *testing.T) {
	for _, s := range []pendingevent.Status{pendingevent.StatusProcessing, pendingevent.StatusDone} {
		t.Run(string(s), func(t *testing.T) {
			e, _ := pendingevent.NewPendingEvent(validInput())
			e.Status = s
			if err := e.MarkProcessing(); !errors.Is(err, pendingevent.ErrInvalidStatus) {
				t.Errorf("err = %v, want ErrInvalidStatus", err)
			}
		})
	}
}

func TestMarkDone_Happy(t *testing.T) {
	e, _ := pendingevent.NewPendingEvent(validInput())
	_ = e.MarkProcessing()
	if err := e.MarkDone(); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if e.Status != pendingevent.StatusDone {
		t.Errorf("status = %q, want done", e.Status)
	}
	if e.ProcessedAt == nil {
		t.Error("ProcessedAt not set")
	}
	if e.LastError != nil {
		t.Error("LastError should be cleared on done")
	}
}

func TestMarkDone_InvalidFrom(t *testing.T) {
	for _, s := range []pendingevent.Status{pendingevent.StatusPending, pendingevent.StatusFailed, pendingevent.StatusDone} {
		t.Run(string(s), func(t *testing.T) {
			e, _ := pendingevent.NewPendingEvent(validInput())
			e.Status = s
			if err := e.MarkDone(); !errors.Is(err, pendingevent.ErrInvalidStatus) {
				t.Errorf("err = %v, want ErrInvalidStatus", err)
			}
		})
	}
}

func TestMarkFailed_RecordsErrorAndBackoff(t *testing.T) {
	e, _ := pendingevent.NewPendingEvent(validInput())
	_ = e.MarkProcessing()
	now := time.Now()
	if err := e.MarkFailed(errors.New("transient stripe 500")); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if e.Status != pendingevent.StatusFailed {
		t.Errorf("status = %q, want failed", e.Status)
	}
	if e.LastError == nil || *e.LastError != "transient stripe 500" {
		t.Errorf("LastError = %v, want 'transient stripe 500'", e.LastError)
	}
	// First retry backoff is 1 minute — fires_at should be roughly
	// now + 1 minute.
	expected := now.Add(1 * time.Minute)
	if e.FiresAt.Before(expected.Add(-2*time.Second)) || e.FiresAt.After(expected.Add(2*time.Second)) {
		t.Errorf("FiresAt = %v, want ~%v", e.FiresAt, expected)
	}
}

func TestMarkFailed_ExponentialBackoff(t *testing.T) {
	cases := []struct {
		attempts int
		minDelay time.Duration
		maxDelay time.Duration
	}{
		{1, 1 * time.Minute, 1 * time.Minute},
		{2, 5 * time.Minute, 5 * time.Minute},
		{3, 15 * time.Minute, 15 * time.Minute},
		{4, 1 * time.Hour, 1 * time.Hour},
		{5, 6 * time.Hour, 6 * time.Hour},
	}
	for _, c := range cases {
		t.Run("", func(t *testing.T) {
			e, _ := pendingevent.NewPendingEvent(validInput())
			e.Attempts = c.attempts
			e.Status = pendingevent.StatusProcessing
			before := time.Now()
			_ = e.MarkFailed(errors.New("err"))
			delay := e.FiresAt.Sub(before)
			// ExceededMaxAttempts (attempt 5) does NOT bump fires_at
			// further — so for attempts==5 the test only checks the
			// helper, not the actual backoff (which is skipped).
			if c.attempts < pendingevent.MaxAttempts {
				if delay < c.minDelay || delay > c.maxDelay+time.Second {
					t.Errorf("attempts=%d delay=%v, want ~%v", c.attempts, delay, c.minDelay)
				}
			}
		})
	}
}

func TestMarkFailed_StopsRetryingPastMaxAttempts(t *testing.T) {
	e, _ := pendingevent.NewPendingEvent(validInput())
	originalFiresAt := e.FiresAt
	e.Attempts = pendingevent.MaxAttempts
	e.Status = pendingevent.StatusProcessing
	_ = e.MarkFailed(errors.New("forever broken"))
	if e.FiresAt != originalFiresAt {
		t.Errorf("expected fires_at unchanged when attempts >= MaxAttempts, got %v", e.FiresAt)
	}
	if !e.HasExceededMaxAttempts() {
		t.Error("HasExceededMaxAttempts should return true")
	}
}

func TestHasExceededMaxAttempts(t *testing.T) {
	e, _ := pendingevent.NewPendingEvent(validInput())
	if e.HasExceededMaxAttempts() {
		t.Error("fresh event should not be exceeded")
	}
	e.Attempts = pendingevent.MaxAttempts
	if !e.HasExceededMaxAttempts() {
		t.Error("event at MaxAttempts should be exceeded")
	}
}
