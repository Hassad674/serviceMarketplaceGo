package report

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domain "marketplace-backend/internal/domain/report"
	userdomain "marketplace-backend/internal/domain/user"
)

// fakeAdminNotifier is a minimal AdminNotifierService that records
// each IncrementAll call so tests can assert the goroutine ran AND
// inspect the context it ran with.
type fakeAdminNotifier struct {
	called    atomic.Int32
	lastErr   atomic.Value // error
	lastCtxFn func(ctx context.Context)
}

func (f *fakeAdminNotifier) IncrementAll(ctx context.Context, _ string) error {
	if f.lastCtxFn != nil {
		f.lastCtxFn(ctx)
	}
	f.called.Add(1)
	if v := f.lastErr.Load(); v != nil {
		if e, ok := v.(error); ok {
			return e
		}
	}
	return nil
}

func (f *fakeAdminNotifier) GetAll(_ context.Context, _ uuid.UUID) (map[string]int64, error) {
	return nil, nil
}

func (f *fakeAdminNotifier) Reset(_ context.Context, _ uuid.UUID, _ string) error {
	return nil
}

// TestService_CreateReport_AdminNotifier_GoroutineDetachesFromCtx is
// the regression guard for gosec G118: even when the request context
// is canceled immediately after CreateReport returns, the admin
// notifier increment goroutine must still execute. The fix uses
// context.WithoutCancel(ctx) so trace identifiers survive while
// request cancellation does not propagate.
func TestService_CreateReport_AdminNotifier_GoroutineDetachesFromCtx(t *testing.T) {
	notifier := &fakeAdminNotifier{}
	svc := NewService(ServiceDeps{
		Reports: &mockReportRepo{
			hasPendingReportFn: func(_ context.Context, _ uuid.UUID, _ string, _ uuid.UUID) (bool, error) {
				return false, nil
			},
			createFn: func(_ context.Context, _ *domain.Report) error { return nil },
		},
		Users:    &mockUserRepo{},
		Messages: &mockMessageRepo{},
	})
	svc.SetAdminNotifier(notifier)

	// Cancellable context — cancel as soon as CreateReport returns.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, err := svc.CreateReport(ctx, CreateReportInput{
		ReporterID: uuid.New(),
		TargetType: "message",
		TargetID:   uuid.New(),
		Reason:     "spam",
	})
	require.NoError(t, err)
	cancel() // simulate the HTTP handler returning

	// Wait for the detached goroutine to land — should still run
	// because of WithoutCancel.
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if notifier.called.Load() >= 1 {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("admin notifier goroutine did not run within deadline (called=%d)", notifier.called.Load())
}

// TestService_CreateReport_AdminNotifier_HasTimeout proves that the
// detached goroutine still has a finite deadline. The notifier
// asserts the received context has a non-zero deadline so a stuck
// downstream cannot leak goroutines indefinitely.
func TestService_CreateReport_AdminNotifier_HasTimeout(t *testing.T) {
	var observedDeadline atomic.Value
	notifier := &fakeAdminNotifier{
		lastCtxFn: func(ctx context.Context) {
			if dl, ok := ctx.Deadline(); ok {
				observedDeadline.Store(dl)
			}
		},
	}
	svc := NewService(ServiceDeps{
		Reports: &mockReportRepo{
			hasPendingReportFn: func(_ context.Context, _ uuid.UUID, _ string, _ uuid.UUID) (bool, error) {
				return false, nil
			},
			createFn: func(_ context.Context, _ *domain.Report) error { return nil },
		},
		Users:    &mockUserRepo{},
		Messages: &mockMessageRepo{},
	})
	svc.SetAdminNotifier(notifier)

	_, err := svc.CreateReport(context.Background(), CreateReportInput{
		ReporterID: uuid.New(),
		TargetType: "message",
		TargetID:   uuid.New(),
		Reason:     "spam",
	})
	require.NoError(t, err)

	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if notifier.called.Load() >= 1 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	dl := observedDeadline.Load()
	require.NotNil(t, dl, "goroutine ctx must carry a deadline")
	assert.WithinDuration(t, time.Now().Add(5*time.Second), dl.(time.Time), 6*time.Second)
}

func TestService_CreateReport_ValidMessage(t *testing.T) {
	reporterID := uuid.New()
	targetID := uuid.New()

	svc := NewService(ServiceDeps{
		Reports: &mockReportRepo{
			hasPendingReportFn: func(_ context.Context, _ uuid.UUID, _ string, _ uuid.UUID) (bool, error) {
				return false, nil
			},
			createFn: func(_ context.Context, _ *domain.Report) error {
				return nil
			},
		},
		Users:    &mockUserRepo{},
		Messages: &mockMessageRepo{},
	})

	r, err := svc.CreateReport(context.Background(), CreateReportInput{
		ReporterID: reporterID,
		TargetType: "message",
		TargetID:   targetID,
		Reason:     "spam",
	})

	assert.NoError(t, err)
	assert.NotNil(t, r)
	assert.Equal(t, domain.TargetMessage, r.TargetType)
	assert.Equal(t, domain.ReasonSpam, r.Reason)
	assert.Equal(t, domain.StatusPending, r.Status)
}

func TestService_CreateReport_ValidUser(t *testing.T) {
	reporterID := uuid.New()
	targetID := uuid.New()

	svc := NewService(ServiceDeps{
		Reports: &mockReportRepo{
			hasPendingReportFn: func(_ context.Context, _ uuid.UUID, _ string, _ uuid.UUID) (bool, error) {
				return false, nil
			},
			createFn: func(_ context.Context, _ *domain.Report) error {
				return nil
			},
		},
		Users: &mockUserRepo{
			getByIDFn: func(_ context.Context, id uuid.UUID) (*userdomain.User, error) {
				return &userdomain.User{ID: id}, nil
			},
		},
		Messages: &mockMessageRepo{},
	})

	r, err := svc.CreateReport(context.Background(), CreateReportInput{
		ReporterID: reporterID,
		TargetType: "user",
		TargetID:   targetID,
		Reason:     "fake_profile",
	})

	assert.NoError(t, err)
	assert.NotNil(t, r)
	assert.Equal(t, domain.TargetUser, r.TargetType)
	assert.Equal(t, domain.ReasonFakeProfile, r.Reason)
}

func TestService_CreateReport_SelfReport(t *testing.T) {
	userID := uuid.New()

	svc := NewService(ServiceDeps{
		Reports: &mockReportRepo{
			hasPendingReportFn: func(_ context.Context, _ uuid.UUID, _ string, _ uuid.UUID) (bool, error) {
				return false, nil
			},
		},
		Users: &mockUserRepo{
			getByIDFn: func(_ context.Context, id uuid.UUID) (*userdomain.User, error) {
				return &userdomain.User{ID: id}, nil
			},
		},
		Messages: &mockMessageRepo{},
	})

	_, err := svc.CreateReport(context.Background(), CreateReportInput{
		ReporterID: userID,
		TargetType: "user",
		TargetID:   userID,
		Reason:     "spam",
	})

	assert.ErrorIs(t, err, domain.ErrSelfReport)
}

func TestService_CreateReport_AlreadyReported(t *testing.T) {
	svc := NewService(ServiceDeps{
		Reports: &mockReportRepo{
			hasPendingReportFn: func(_ context.Context, _ uuid.UUID, _ string, _ uuid.UUID) (bool, error) {
				return true, nil
			},
		},
		Users:    &mockUserRepo{},
		Messages: &mockMessageRepo{},
	})

	_, err := svc.CreateReport(context.Background(), CreateReportInput{
		ReporterID: uuid.New(),
		TargetType: "message",
		TargetID:   uuid.New(),
		Reason:     "spam",
	})

	assert.ErrorIs(t, err, domain.ErrAlreadyReported)
}

func TestService_CreateReport_InvalidReason(t *testing.T) {
	svc := NewService(ServiceDeps{
		Reports: &mockReportRepo{
			hasPendingReportFn: func(_ context.Context, _ uuid.UUID, _ string, _ uuid.UUID) (bool, error) {
				return false, nil
			},
		},
		Users:    &mockUserRepo{},
		Messages: &mockMessageRepo{},
	})

	_, err := svc.CreateReport(context.Background(), CreateReportInput{
		ReporterID: uuid.New(),
		TargetType: "message",
		TargetID:   uuid.New(),
		Reason:     "fake_profile", // not valid for message
	})

	assert.ErrorIs(t, err, domain.ErrReasonNotAllowedForType)
}

func TestService_ListMyReports(t *testing.T) {
	reporterID := uuid.New()
	expected := []*domain.Report{
		{ID: uuid.New(), ReporterID: reporterID, TargetType: domain.TargetMessage},
		{ID: uuid.New(), ReporterID: reporterID, TargetType: domain.TargetUser},
	}

	svc := NewService(ServiceDeps{
		Reports: &mockReportRepo{
			listByReporterFn: func(_ context.Context, _ uuid.UUID, _ string, _ int) ([]*domain.Report, string, error) {
				return expected, "next_cursor", nil
			},
		},
		Users:    &mockUserRepo{},
		Messages: &mockMessageRepo{},
	})

	reports, cursor, err := svc.ListMyReports(context.Background(), reporterID, "", 20)

	assert.NoError(t, err)
	assert.Len(t, reports, 2)
	assert.Equal(t, "next_cursor", cursor)
}
