package job

import (
	"context"
	"testing"
	"time"

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
		stubOrg := uuid.New()
		return &user.User{ID: id, Role: user.RoleProvider, OrganizationID: &stubOrg}, nil
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
		return &user.User{ID: id, Role: user.RoleProvider, OrganizationID: &[]uuid.UUID{uuid.New()}[0]}, nil
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
		return &user.User{ID: id, Role: user.RoleProvider, OrganizationID: &[]uuid.UUID{uuid.New()}[0]}, nil // provider trying to apply
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
	pr.orgProfilesByUserIDsFn = func(_ context.Context, ids []uuid.UUID) (map[uuid.UUID]*profile.PublicProfile, error) {
		return map[uuid.UUID]*profile.PublicProfile{
			applicantID: {OrganizationID: uuid.New(), Name: "Test Org"},
		}, nil
	}

	items, _, err := svc.ListJobApplications(context.Background(), j.ID, creatorID, "", 20)
	assert.NoError(t, err)
	assert.Len(t, items, 1)
	assert.Equal(t, "Test Org", items[0].Profile.Name)
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

// --- Credit integration tests ---

func newTestApplyServiceWithCredits(cr *mockJobCreditRepo) (*Service, *mockJobRepo, *mockJobApplicationRepo, *mockUserRepo) {
	jr := &mockJobRepo{}
	ar := &mockJobApplicationRepo{}
	ur := &mockUserRepo{}
	svc := NewService(ServiceDeps{
		Jobs:         jr,
		Applications: ar,
		Users:        ur,
		Credits:      cr,
	})
	return svc, jr, ar, ur
}

func TestApplyToJob_NoCreditsLeft(t *testing.T) {
	cr := &mockJobCreditRepo{
		getOrCreateFn: func(_ context.Context, _ uuid.UUID) (int, error) {
			return 0, nil // zero credits
		},
	}
	svc, jr, _, ur := newTestApplyServiceWithCredits(cr)
	creatorID := uuid.New()
	applicantID := uuid.New()
	j := openJob(creatorID)

	jr.getByIDFn = func(_ context.Context, _ uuid.UUID) (*domain.Job, error) { return j, nil }
	ur.getByIDFn = func(_ context.Context, id uuid.UUID) (*user.User, error) {
		return &user.User{ID: id, Role: user.RoleProvider, OrganizationID: &[]uuid.UUID{uuid.New()}[0]}, nil
	}

	_, err := svc.ApplyToJob(context.Background(), ApplyToJobInput{
		JobID: j.ID, ApplicantID: applicantID, Message: "I want this job",
	})
	assert.ErrorIs(t, err, domain.ErrNoCreditsLeft)
}

func TestApplyToJob_CreditsDecremented(t *testing.T) {
	var decremented bool
	cr := &mockJobCreditRepo{
		getOrCreateFn: func(_ context.Context, _ uuid.UUID) (int, error) {
			return 5, nil // has credits
		},
		decrementFn: func(_ context.Context, _ uuid.UUID) error {
			decremented = true
			return nil
		},
	}
	svc, jr, _, ur := newTestApplyServiceWithCredits(cr)
	creatorID := uuid.New()
	applicantID := uuid.New()
	j := openJob(creatorID)

	jr.getByIDFn = func(_ context.Context, _ uuid.UUID) (*domain.Job, error) { return j, nil }
	ur.getByIDFn = func(_ context.Context, id uuid.UUID) (*user.User, error) {
		return &user.User{ID: id, Role: user.RoleProvider, OrganizationID: &[]uuid.UUID{uuid.New()}[0]}, nil
	}

	app, err := svc.ApplyToJob(context.Background(), ApplyToJobInput{
		JobID: j.ID, ApplicantID: applicantID, Message: "I want this job",
	})
	assert.NoError(t, err)
	assert.NotNil(t, app)
	assert.True(t, decremented, "credits should have been decremented after apply")
}

func TestApplyToJob_NilCreditsRepo(t *testing.T) {
	// When credits repo is nil, apply should work without credit checks.
	svc, jr, _, ur, _, _ := newTestApplyService()
	creatorID := uuid.New()
	applicantID := uuid.New()
	j := openJob(creatorID)

	jr.getByIDFn = func(_ context.Context, _ uuid.UUID) (*domain.Job, error) { return j, nil }
	ur.getByIDFn = func(_ context.Context, id uuid.UUID) (*user.User, error) {
		return &user.User{ID: id, Role: user.RoleProvider, OrganizationID: &[]uuid.UUID{uuid.New()}[0]}, nil
	}

	app, err := svc.ApplyToJob(context.Background(), ApplyToJobInput{
		JobID: j.ID, ApplicantID: applicantID, Message: "test",
	})
	assert.NoError(t, err)
	assert.NotNil(t, app)
}

func TestGetCredits_WithRepo(t *testing.T) {
	cr := &mockJobCreditRepo{
		getOrCreateFn: func(_ context.Context, _ uuid.UUID) (int, error) {
			return 7, nil
		},
	}
	svc, _, _, _ := newTestApplyServiceWithCredits(cr)

	credits, err := svc.GetCredits(context.Background(), uuid.New())
	assert.NoError(t, err)
	assert.Equal(t, 7, credits)
}

func TestGetCredits_NilRepo(t *testing.T) {
	svc, _, _, _, _, _ := newTestApplyService()

	credits, err := svc.GetCredits(context.Background(), uuid.New())
	assert.NoError(t, err)
	assert.Equal(t, domain.WeeklyQuota, credits)
}

// --- KYC enforcement tests ---

func TestApplyToJob_KYCBlocked(t *testing.T) {
	svc, jr, _, ur, _, _ := newTestApplyService()
	creatorID := uuid.New()
	applicantID := uuid.New()
	j := openJob(creatorID)
	past15 := time.Now().Add(-15 * 24 * time.Hour)

	jr.getByIDFn = func(_ context.Context, _ uuid.UUID) (*domain.Job, error) { return j, nil }
	ur.getByIDFn = func(_ context.Context, id uuid.UUID) (*user.User, error) {
		u := &user.User{ID: id, Role: user.RoleProvider}
		u.KYCFirstEarningAt = &past15
		return u, nil
	}

	_, err := svc.ApplyToJob(context.Background(), ApplyToJobInput{
		JobID: j.ID, ApplicantID: applicantID, Message: "test",
	})
	assert.ErrorIs(t, err, user.ErrKYCRestricted)
}

func TestApplyToJob_KYCNotBlocked_OK(t *testing.T) {
	svc, jr, _, ur, _, _ := newTestApplyService()
	creatorID := uuid.New()
	applicantID := uuid.New()
	j := openJob(creatorID)
	past5 := time.Now().Add(-5 * 24 * time.Hour)

	jr.getByIDFn = func(_ context.Context, _ uuid.UUID) (*domain.Job, error) { return j, nil }
	ur.getByIDFn = func(_ context.Context, id uuid.UUID) (*user.User, error) {
		stubOrg := uuid.New()
		u := &user.User{ID: id, Role: user.RoleProvider, OrganizationID: &stubOrg}
		u.KYCFirstEarningAt = &past5
		return u, nil
	}

	app, err := svc.ApplyToJob(context.Background(), ApplyToJobInput{
		JobID: j.ID, ApplicantID: applicantID, Message: "I am interested",
	})
	assert.NoError(t, err)
	assert.NotNil(t, app)
}
