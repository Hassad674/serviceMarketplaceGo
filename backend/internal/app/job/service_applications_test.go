package job

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	domain "marketplace-backend/internal/domain/job"
	"marketplace-backend/internal/domain/profile"
	"marketplace-backend/internal/domain/user"
)

func newTestApplyService() (*Service, *mockJobRepo, *mockJobApplicationRepo, *mockUserRepo, *mockProfileRepo, *mockMsgSender) {
	jr := &mockJobRepo{}
	ar := &mockJobApplicationRepo{}
	ur := &mockUserRepo{}
	pr := &mockProfileRepo{}
	ms := &mockMsgSender{}
	svc := NewService(ServiceDeps{
		Jobs:         jr,
		Applications: ar,
		Users:        ur,
		Profiles:     pr,
		Messages:     ms,
	})
	return svc, jr, ar, ur, pr, ms
}

func openJob(creatorID uuid.UUID) *domain.Job {
	j, _ := domain.NewJob(domain.NewJobInput{
		CreatorID:     creatorID,
		Title:         "Test Job",
		Description:   "A test job",
		Skills:        []string{"Go"},
		ApplicantType: domain.ApplicantAll,
		BudgetType:    domain.BudgetOneShot,
		MinBudget:     1000,
		MaxBudget:     5000,
	})
	return j
}

func TestApplyToJob_Success(t *testing.T) {
	svc, jr, _, ur, _, _ := newTestApplyService()
	creatorID := uuid.New()
	applicantID := uuid.New()
	j := openJob(creatorID)

	jr.getByIDFn = func(_ context.Context, id uuid.UUID) (*domain.Job, error) { return j, nil }
	ur.getByIDFn = func(_ context.Context, id uuid.UUID) (*user.User, error) {
		return &user.User{ID: id, Role: user.RoleProvider}, nil
	}

	app, err := svc.ApplyToJob(context.Background(), ApplyToJobInput{
		JobID: j.ID, ApplicantID: applicantID, Message: "I am interested",
	})
	assert.NoError(t, err)
	assert.NotNil(t, app)
	assert.Equal(t, j.ID, app.JobID)
	assert.Equal(t, applicantID, app.ApplicantID)
}

func TestApplyToJob_ClosedJob(t *testing.T) {
	svc, jr, _, _, _, _ := newTestApplyService()
	j := openJob(uuid.New())
	_ = j.Close(j.CreatorID)

	jr.getByIDFn = func(_ context.Context, _ uuid.UUID) (*domain.Job, error) { return j, nil }

	_, err := svc.ApplyToJob(context.Background(), ApplyToJobInput{
		JobID: j.ID, ApplicantID: uuid.New(), Message: "test",
	})
	assert.ErrorIs(t, err, domain.ErrCannotApplyToClosed)
}

func TestApplyToJob_OwnJob(t *testing.T) {
	svc, jr, _, _, _, _ := newTestApplyService()
	creatorID := uuid.New()
	j := openJob(creatorID)

	jr.getByIDFn = func(_ context.Context, _ uuid.UUID) (*domain.Job, error) { return j, nil }

	_, err := svc.ApplyToJob(context.Background(), ApplyToJobInput{
		JobID: j.ID, ApplicantID: creatorID, Message: "test",
	})
	assert.ErrorIs(t, err, domain.ErrCannotApplyToOwnJob)
}

func TestApplyToJob_AlreadyApplied(t *testing.T) {
	svc, jr, ar, ur, _, _ := newTestApplyService()
	j := openJob(uuid.New())
	applicantID := uuid.New()

	jr.getByIDFn = func(_ context.Context, _ uuid.UUID) (*domain.Job, error) { return j, nil }
	ur.getByIDFn = func(_ context.Context, id uuid.UUID) (*user.User, error) {
		return &user.User{ID: id, Role: user.RoleProvider}, nil
	}
	ar.getByJobAndApplicantFn = func(_ context.Context, _, _ uuid.UUID) (*domain.JobApplication, error) {
		return &domain.JobApplication{}, nil // found — already applied
	}

	_, err := svc.ApplyToJob(context.Background(), ApplyToJobInput{
		JobID: j.ID, ApplicantID: applicantID, Message: "test",
	})
	assert.ErrorIs(t, err, domain.ErrAlreadyApplied)
}

func TestApplyToJob_TypeMismatch(t *testing.T) {
	svc, jr, _, ur, _, _ := newTestApplyService()
	creatorID := uuid.New()
	j := openJob(creatorID)
	j.ApplicantType = domain.ApplicantAgencies // agencies only

	jr.getByIDFn = func(_ context.Context, _ uuid.UUID) (*domain.Job, error) { return j, nil }
	ur.getByIDFn = func(_ context.Context, id uuid.UUID) (*user.User, error) {
		return &user.User{ID: id, Role: user.RoleProvider}, nil // provider trying to apply
	}

	_, err := svc.ApplyToJob(context.Background(), ApplyToJobInput{
		JobID: j.ID, ApplicantID: uuid.New(), Message: "test",
	})
	assert.ErrorIs(t, err, domain.ErrApplicantTypeMismatch)
}

func TestWithdrawApplication_Success(t *testing.T) {
	svc, _, ar, _, _, _ := newTestApplyService()
	applicantID := uuid.New()
	appID := uuid.New()

	ar.getByIDFn = func(_ context.Context, _ uuid.UUID) (*domain.JobApplication, error) {
		return &domain.JobApplication{ID: appID, ApplicantID: applicantID}, nil
	}

	err := svc.WithdrawApplication(context.Background(), appID, applicantID)
	assert.NoError(t, err)
}

func TestWithdrawApplication_NotApplicant(t *testing.T) {
	svc, _, ar, _, _, _ := newTestApplyService()
	appID := uuid.New()

	ar.getByIDFn = func(_ context.Context, _ uuid.UUID) (*domain.JobApplication, error) {
		return &domain.JobApplication{ID: appID, ApplicantID: uuid.New()}, nil
	}

	err := svc.WithdrawApplication(context.Background(), appID, uuid.New())
	assert.ErrorIs(t, err, domain.ErrNotApplicant)
}

func TestListJobApplications_NotOwner(t *testing.T) {
	svc, jr, _, _, _, _ := newTestApplyService()
	j := openJob(uuid.New())
	jr.getByIDFn = func(_ context.Context, _ uuid.UUID) (*domain.Job, error) { return j, nil }

	_, _, err := svc.ListJobApplications(context.Background(), j.ID, uuid.New(), "", 20)
	assert.ErrorIs(t, err, domain.ErrNotOwner)
}

func TestListJobApplications_WithProfiles(t *testing.T) {
	svc, jr, ar, _, pr, _ := newTestApplyService()
	creatorID := uuid.New()
	j := openJob(creatorID)
	applicantID := uuid.New()

	jr.getByIDFn = func(_ context.Context, _ uuid.UUID) (*domain.Job, error) { return j, nil }
	ar.listByJobFn = func(_ context.Context, _ uuid.UUID, _ string, _ int) ([]*domain.JobApplication, string, error) {
		return []*domain.JobApplication{{ID: uuid.New(), JobID: j.ID, ApplicantID: applicantID, Message: "hi"}}, "", nil
	}
	pr.getPublicProfilesByUserIDsFn = func(_ context.Context, ids []uuid.UUID) ([]*profile.PublicProfile, error) {
		return []*profile.PublicProfile{{UserID: applicantID, DisplayName: "Test"}}, nil
	}

	items, _, err := svc.ListJobApplications(context.Background(), j.ID, creatorID, "", 20)
	assert.NoError(t, err)
	assert.Len(t, items, 1)
	assert.Equal(t, "Test", items[0].Profile.DisplayName)
}

func TestHasApplied_True(t *testing.T) {
	svc, _, ar, _, _, _ := newTestApplyService()
	ar.getByJobAndApplicantFn = func(_ context.Context, _, _ uuid.UUID) (*domain.JobApplication, error) {
		return &domain.JobApplication{}, nil
	}
	applied, err := svc.HasApplied(context.Background(), uuid.New(), uuid.New())
	assert.NoError(t, err)
	assert.True(t, applied)
}

func TestHasApplied_False(t *testing.T) {
	svc, _, _, _, _, _ := newTestApplyService()
	// default mock returns ErrApplicationNotFound
	applied, err := svc.HasApplied(context.Background(), uuid.New(), uuid.New())
	assert.NoError(t, err)
	assert.False(t, applied)
}
