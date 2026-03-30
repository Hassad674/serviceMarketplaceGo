package report

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	domain "marketplace-backend/internal/domain/report"
	userdomain "marketplace-backend/internal/domain/user"
)

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
