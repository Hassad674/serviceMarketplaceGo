package job

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domain "marketplace-backend/internal/domain/job"
	"marketplace-backend/internal/domain/organization"
	"marketplace-backend/internal/domain/profile"
	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/port/repository"
)

func newTestApplyService() (*Service, *mockJobRepo, *mockJobApplicationRepo, *mockUserRepo, *mockProfileRepo, *mockMsgSender) {
	svc, jr, ar, ur, pr, ms, _ := newTestApplyServiceFull()
	return svc, jr, ar, ur, pr, ms
}

func newTestApplyServiceFull() (*Service, *mockJobRepo, *mockJobApplicationRepo, *mockUserRepo, *mockProfileRepo, *mockMsgSender, *mockOrgRepo) {
	jr := &mockJobRepo{}
	ar := &mockJobApplicationRepo{}
	ur := &mockUserRepo{}
	or := &mockOrgRepo{}
	pr := &mockProfileRepo{}
	ms := &mockMsgSender{}
	svc := NewService(ServiceDeps{
		Jobs:          jr,
		Applications:  ar,
		Users:         ur,
		Organizations: or,
		Profiles:      pr,
		Messages:      ms,
	})
	return svc, jr, ar, ur, pr, ms, or
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

	_, _, err := svc.ListJobApplications(context.Background(), j.ID, uuid.New(), "", 20, ListJobApplicationsFilter{})
	assert.ErrorIs(t, err, domain.ErrNotOwner)
}

func TestListJobApplications_WithProfiles(t *testing.T) {
	svc, jr, ar, _, pr, _ := newTestApplyService()
	creatorID := uuid.New()
	j := openJob(creatorID)
	applicantID := uuid.New()

	jr.getByIDFn = func(_ context.Context, _ uuid.UUID) (*domain.Job, error) { return j, nil }
	ar.listByJobFn = func(_ context.Context, _ uuid.UUID, _ string, _ int, _ domain.ApplicantKind) ([]*domain.JobApplication, string, error) {
		return []*domain.JobApplication{{ID: uuid.New(), JobID: j.ID, ApplicantID: applicantID, Message: "hi"}}, "", nil
	}
	pr.orgProfilesByUserIDsFn = func(_ context.Context, ids []uuid.UUID) (map[uuid.UUID]*profile.PublicProfile, error) {
		return map[uuid.UUID]*profile.PublicProfile{
			applicantID: {OrganizationID: uuid.New(), Name: "Test Org"},
		}, nil
	}

	items, _, err := svc.ListJobApplications(context.Background(), j.ID, creatorID, "", 20, ListJobApplicationsFilter{})
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
	// R12 — the atomic decrement is the authoritative gate. Under the
	// new model, Decrement returns ErrNoCreditsLeft directly when the
	// org pool is exhausted.
	cr := &mockJobCreditRepo{
		decrementFn: func(_ context.Context, _ uuid.UUID) error {
			return domain.ErrNoCreditsLeft
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
	cr := &mockJobCreditRepo{}
	svc, jr, _, ur := newTestApplyServiceWithCredits(cr)
	creatorID := uuid.New()
	applicantID := uuid.New()
	orgID := uuid.New()
	j := openJob(creatorID)

	jr.getByIDFn = func(_ context.Context, _ uuid.UUID) (*domain.Job, error) { return j, nil }
	ur.getByIDFn = func(_ context.Context, id uuid.UUID) (*user.User, error) {
		return &user.User{ID: id, Role: user.RoleProvider, OrganizationID: &orgID}, nil
	}

	app, err := svc.ApplyToJob(context.Background(), ApplyToJobInput{
		JobID: j.ID, ApplicantID: applicantID, Message: "I want this job",
	})
	assert.NoError(t, err)
	assert.NotNil(t, app)
	// R12 — the decrement must target the applicant's ORG, not the user.
	require.Len(t, cr.decrementCalls, 1)
	assert.Equal(t, orgID, cr.decrementCalls[0],
		"decrement must hit the applicant's org, not the user")
	assert.Empty(t, cr.refundCalls, "no refund on a successful apply")
}

// R12 — a user with zero user-credits (pre-migration notion) but whose
// org has credits CAN apply. Conversely, a user whose org is exhausted
// CANNOT apply even if they personally had credits historically. The
// mock org id encodes this via the Decrement stub.
func TestApplyToJob_SharedOrgPool_MembersShareCredits(t *testing.T) {
	orgID := uuid.New()
	var pool = 2 // two credits shared between the whole org
	cr := &mockJobCreditRepo{
		getOrCreateFn: func(_ context.Context, gotOrg uuid.UUID) (int, error) {
			assert.Equal(t, orgID, gotOrg)
			return pool, nil
		},
		decrementFn: func(_ context.Context, gotOrg uuid.UUID) error {
			assert.Equal(t, orgID, gotOrg)
			if pool <= 0 {
				return domain.ErrNoCreditsLeft
			}
			pool--
			return nil
		},
	}
	svc, jr, _, ur := newTestApplyServiceWithCredits(cr)

	// Two different applicant users, same org.
	aliceID := uuid.New()
	bobID := uuid.New()
	creatorID := uuid.New()

	jr.getByIDFn = func(_ context.Context, _ uuid.UUID) (*domain.Job, error) {
		return openJob(creatorID), nil
	}
	ur.getByIDFn = func(_ context.Context, id uuid.UUID) (*user.User, error) {
		return &user.User{ID: id, Role: user.RoleProvider, OrganizationID: &orgID}, nil
	}

	// Alice burns credit #1
	_, err := svc.ApplyToJob(context.Background(), ApplyToJobInput{
		JobID: uuid.New(), ApplicantID: aliceID, Message: "first"})
	require.NoError(t, err)

	// Bob burns credit #2
	_, err = svc.ApplyToJob(context.Background(), ApplyToJobInput{
		JobID: uuid.New(), ApplicantID: bobID, Message: "second"})
	require.NoError(t, err)

	// Alice again — pool is empty, must fail even though she is a
	// different user than Bob.
	_, err = svc.ApplyToJob(context.Background(), ApplyToJobInput{
		JobID: uuid.New(), ApplicantID: aliceID, Message: "third"})
	assert.ErrorIs(t, err, domain.ErrNoCreditsLeft)

	assert.Equal(t, 0, pool, "shared pool must be fully drained")
}

// R12 — when the INSERT step fails after a successful debit, the
// credit is refunded so the shared pool stays consistent.
func TestApplyToJob_RefundsOnInsertFailure(t *testing.T) {
	cr := &mockJobCreditRepo{}
	svc, jr, ar, ur := newTestApplyServiceWithCredits(cr)
	creatorID := uuid.New()
	applicantID := uuid.New()
	orgID := uuid.New()

	jr.getByIDFn = func(_ context.Context, _ uuid.UUID) (*domain.Job, error) {
		return openJob(creatorID), nil
	}
	ur.getByIDFn = func(_ context.Context, id uuid.UUID) (*user.User, error) {
		return &user.User{ID: id, Role: user.RoleProvider, OrganizationID: &orgID}, nil
	}
	// Force the INSERT to fail after the credit was spent.
	ar.createFn = func(_ context.Context, _ *domain.JobApplication) error {
		return assertErr("db boom")
	}

	_, err := svc.ApplyToJob(context.Background(), ApplyToJobInput{
		JobID: uuid.New(), ApplicantID: applicantID, Message: "oops",
	})
	assert.Error(t, err)
	require.Len(t, cr.decrementCalls, 1, "credit should have been debited")
	require.Len(t, cr.refundCalls, 1, "credit should have been refunded after insert failure")
	assert.Equal(t, orgID, cr.refundCalls[0])
}

// assertErr is a tiny helper so the test file does not need to import
// "errors" just for one stub.
type assertErr string

func (e assertErr) Error() string { return string(e) }

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
	svc, jr, _, ur, _, _, or := newTestApplyServiceFull()
	creatorID := uuid.New()
	applicantID := uuid.New()
	j := openJob(creatorID)
	past15 := time.Now().Add(-15 * 24 * time.Hour)

	jr.getByIDFn = func(_ context.Context, _ uuid.UUID) (*domain.Job, error) { return j, nil }
	ur.getByIDFn = func(_ context.Context, id uuid.UUID) (*user.User, error) {
		return &user.User{ID: id, Role: user.RoleProvider, OrganizationID: &[]uuid.UUID{uuid.New()}[0]}, nil
	}
	or.findByUserIDFn = func(_ context.Context, _ uuid.UUID) (*organization.Organization, error) {
		return &organization.Organization{
			ID:                uuid.New(),
			Type:              organization.OrgTypeProviderPersonal,
			KYCFirstEarningAt: &past15,
		}, nil
	}

	_, err := svc.ApplyToJob(context.Background(), ApplyToJobInput{
		JobID: j.ID, ApplicantID: applicantID, Message: "test",
	})
	assert.ErrorIs(t, err, user.ErrKYCRestricted)
}

// --- ListOpenJobs (public marketplace feed with social-proof counts) ---

func newTestListOpenService(jr *mockJobRepo, jv *mockJobViewRepo) *Service {
	return NewService(ServiceDeps{
		Jobs:         jr,
		Applications: &mockJobApplicationRepo{},
		Users:        &mockUserRepo{},
		JobViews:     jv,
	})
}

func TestListOpenJobs_EmptyResult(t *testing.T) {
	jr := &mockJobRepo{listOpenFn: func(_ context.Context, _ repository.JobListFilters, _ string, _ int) ([]*domain.Job, string, error) {
		return []*domain.Job{}, "", nil
	}}
	jv := &mockJobViewRepo{}
	svc := newTestListOpenService(jr, jv)

	jobs, cursor, err := svc.ListOpenJobs(context.Background(), repository.JobListFilters{}, "", 20)
	require.NoError(t, err)
	assert.Empty(t, jobs)
	assert.Empty(t, cursor)
	// No batch call when the page is empty — the batch helper would
	// otherwise fire with an empty slice and waste a query.
	assert.Equal(t, 0, jv.batchCalls, "batch must not fire on empty page")
}

func TestListOpenJobs_WithCounts(t *testing.T) {
	creatorID := uuid.New()
	j1 := openJob(creatorID)
	j2 := openJob(creatorID)
	j3 := openJob(creatorID) // no applicants — must default to 0

	jr := &mockJobRepo{listOpenFn: func(_ context.Context, _ repository.JobListFilters, _ string, _ int) ([]*domain.Job, string, error) {
		return []*domain.Job{j1, j2, j3}, "next_cursor_xyz", nil
	}}
	jv := &mockJobViewRepo{
		getApplicationCountsBatchFn: func(_ context.Context, _ []uuid.UUID, _ uuid.UUID) (map[uuid.UUID]repository.ApplicationCounts, error) {
			return map[uuid.UUID]repository.ApplicationCounts{
				j1.ID: {Total: 12, NewCount: 4},
				j2.ID: {Total: 1, NewCount: 1},
				// j3 absent on purpose — service must surface 0, not panic.
			}, nil
		},
	}
	svc := newTestListOpenService(jr, jv)

	jobs, cursor, err := svc.ListOpenJobs(context.Background(), repository.JobListFilters{}, "", 20)
	require.NoError(t, err)
	assert.Equal(t, "next_cursor_xyz", cursor)
	require.Len(t, jobs, 3)

	// Counts mapped per job.
	assert.Equal(t, j1.ID, jobs[0].Job.ID)
	assert.Equal(t, 12, jobs[0].TotalApplicants)
	assert.Equal(t, j2.ID, jobs[1].Job.ID)
	assert.Equal(t, 1, jobs[1].TotalApplicants)

	// Edge case: missing entry in the count map → 0, not panic.
	assert.Equal(t, j3.ID, jobs[2].Job.ID)
	assert.Equal(t, 0, jobs[2].TotalApplicants)

	// new_applicants must NEVER be exposed on the public feed.
	for _, jc := range jobs {
		assert.Equal(t, 0, jc.NewApplicants, "new_applicants must stay zero on /jobs/open")
	}
}

func TestListOpenJobs_BatchedNotN1(t *testing.T) {
	// The whole point of swapping per-job count queries for the batch
	// helper is to keep the public feed N+1-free. This test pins that
	// invariant: regardless of the page size, exactly ONE batch call
	// fires — and it carries every job ID on the page.
	creatorID := uuid.New()
	page := make([]*domain.Job, 0, 25)
	for i := 0; i < 25; i++ {
		page = append(page, openJob(creatorID))
	}

	jr := &mockJobRepo{listOpenFn: func(_ context.Context, _ repository.JobListFilters, _ string, _ int) ([]*domain.Job, string, error) {
		return page, "", nil
	}}
	jv := &mockJobViewRepo{}
	svc := newTestListOpenService(jr, jv)

	_, _, err := svc.ListOpenJobs(context.Background(), repository.JobListFilters{}, "", 25)
	require.NoError(t, err)

	assert.Equal(t, 1, jv.batchCalls, "must fire exactly one batch call (no N+1)")
	assert.Len(t, jv.lastBatchIDs, 25, "batch must cover every job on the page")
	// Public feed has no per-user "since I last looked" semantic — we
	// pass uuid.Nil so the LEFT JOIN sentinel kicks in and the total
	// count stays accurate.
	assert.Equal(t, uuid.Nil, jv.lastViewerID, "public feed batch must use uuid.Nil viewer")
}

func TestListOpenJobs_NilJobViewRepo_GracefulZero(t *testing.T) {
	// Legacy test setups that don't wire JobView repo must still get a
	// usable list — counts gracefully default to 0 instead of crashing.
	creatorID := uuid.New()
	j := openJob(creatorID)
	jr := &mockJobRepo{listOpenFn: func(_ context.Context, _ repository.JobListFilters, _ string, _ int) ([]*domain.Job, string, error) {
		return []*domain.Job{j}, "", nil
	}}
	svc := NewService(ServiceDeps{
		Jobs:         jr,
		Applications: &mockJobApplicationRepo{},
		Users:        &mockUserRepo{},
		// JobViews intentionally nil
	})

	jobs, _, err := svc.ListOpenJobs(context.Background(), repository.JobListFilters{}, "", 20)
	require.NoError(t, err)
	require.Len(t, jobs, 1)
	assert.Equal(t, 0, jobs[0].TotalApplicants)
}

func TestListOpenJobs_RepoErrorPropagated(t *testing.T) {
	wantErr := assert.AnError
	jr := &mockJobRepo{listOpenFn: func(_ context.Context, _ repository.JobListFilters, _ string, _ int) ([]*domain.Job, string, error) {
		return nil, "", wantErr
	}}
	svc := newTestListOpenService(jr, &mockJobViewRepo{})

	jobs, cursor, err := svc.ListOpenJobs(context.Background(), repository.JobListFilters{}, "", 20)
	assert.ErrorIs(t, err, wantErr)
	assert.Nil(t, jobs)
	assert.Empty(t, cursor)
}

// --- Applicant kind tests (radio: freelance / agency / referrer) ---

func TestApplyToJob_DefaultKindForProvider_IsFreelance(t *testing.T) {
	svc, jr, ar, ur, _, _ := newTestApplyService()
	j := openJob(uuid.New())
	jr.getByIDFn = func(_ context.Context, _ uuid.UUID) (*domain.Job, error) { return j, nil }
	ur.getByIDFn = func(_ context.Context, id uuid.UUID) (*user.User, error) {
		stubOrg := uuid.New()
		return &user.User{ID: id, Role: user.RoleProvider, OrganizationID: &stubOrg}, nil
	}
	var captured *domain.JobApplication
	ar.createFn = func(_ context.Context, app *domain.JobApplication) error {
		captured = app
		return nil
	}

	_, err := svc.ApplyToJob(context.Background(), ApplyToJobInput{
		JobID: j.ID, ApplicantID: uuid.New(), Message: "default kind",
	})
	require.NoError(t, err)
	require.NotNil(t, captured)
	assert.Equal(t, domain.ApplicantKindFreelance, captured.ApplicantKind)
}

func TestApplyToJob_DefaultKindForAgency_IsAgency(t *testing.T) {
	svc, jr, ar, ur, _, _ := newTestApplyService()
	j := openJob(uuid.New())
	j.ApplicantType = domain.ApplicantAll
	jr.getByIDFn = func(_ context.Context, _ uuid.UUID) (*domain.Job, error) { return j, nil }
	ur.getByIDFn = func(_ context.Context, id uuid.UUID) (*user.User, error) {
		stubOrg := uuid.New()
		return &user.User{ID: id, Role: user.RoleAgency, OrganizationID: &stubOrg}, nil
	}
	var captured *domain.JobApplication
	ar.createFn = func(_ context.Context, app *domain.JobApplication) error {
		captured = app
		return nil
	}

	_, err := svc.ApplyToJob(context.Background(), ApplyToJobInput{
		JobID: j.ID, ApplicantID: uuid.New(), Message: "agency apply",
	})
	require.NoError(t, err)
	require.NotNil(t, captured)
	assert.Equal(t, domain.ApplicantKindAgency, captured.ApplicantKind)
}

func TestApplyToJob_ReferrerEnabledProvider_AcceptsReferrerKind(t *testing.T) {
	svc, jr, ar, ur, _, _ := newTestApplyService()
	j := openJob(uuid.New())
	jr.getByIDFn = func(_ context.Context, _ uuid.UUID) (*domain.Job, error) { return j, nil }
	ur.getByIDFn = func(_ context.Context, id uuid.UUID) (*user.User, error) {
		stubOrg := uuid.New()
		return &user.User{ID: id, Role: user.RoleProvider, OrganizationID: &stubOrg, ReferrerEnabled: true}, nil
	}
	var captured *domain.JobApplication
	ar.createFn = func(_ context.Context, app *domain.JobApplication) error {
		captured = app
		return nil
	}

	_, err := svc.ApplyToJob(context.Background(), ApplyToJobInput{
		JobID:       j.ID,
		ApplicantID: uuid.New(),
		Kind:        domain.ApplicantKindReferrer,
		Message:     "I bring you a great freelance",
	})
	require.NoError(t, err)
	require.NotNil(t, captured)
	assert.Equal(t, domain.ApplicantKindReferrer, captured.ApplicantKind)
}

func TestApplyToJob_ProviderWithoutReferrer_RejectsReferrerKind(t *testing.T) {
	svc, jr, _, ur, _, _ := newTestApplyService()
	j := openJob(uuid.New())
	jr.getByIDFn = func(_ context.Context, _ uuid.UUID) (*domain.Job, error) { return j, nil }
	ur.getByIDFn = func(_ context.Context, id uuid.UUID) (*user.User, error) {
		stubOrg := uuid.New()
		return &user.User{ID: id, Role: user.RoleProvider, OrganizationID: &stubOrg, ReferrerEnabled: false}, nil
	}

	_, err := svc.ApplyToJob(context.Background(), ApplyToJobInput{
		JobID:       j.ID,
		ApplicantID: uuid.New(),
		Kind:        domain.ApplicantKindReferrer,
		Message:     "should fail",
	})
	assert.ErrorIs(t, err, domain.ErrInvalidApplicantKind)
}

func TestApplyToJob_AgencyCannotApplyAsReferrer(t *testing.T) {
	svc, jr, _, ur, _, _ := newTestApplyService()
	j := openJob(uuid.New())
	jr.getByIDFn = func(_ context.Context, _ uuid.UUID) (*domain.Job, error) { return j, nil }
	ur.getByIDFn = func(_ context.Context, id uuid.UUID) (*user.User, error) {
		stubOrg := uuid.New()
		return &user.User{ID: id, Role: user.RoleAgency, OrganizationID: &stubOrg}, nil
	}

	_, err := svc.ApplyToJob(context.Background(), ApplyToJobInput{
		JobID:       j.ID,
		ApplicantID: uuid.New(),
		Kind:        domain.ApplicantKindReferrer,
		Message:     "should fail",
	})
	assert.ErrorIs(t, err, domain.ErrInvalidApplicantKind)
}

func TestApplyToJob_ProviderCannotApplyAsAgency(t *testing.T) {
	svc, jr, _, ur, _, _ := newTestApplyService()
	j := openJob(uuid.New())
	jr.getByIDFn = func(_ context.Context, _ uuid.UUID) (*domain.Job, error) { return j, nil }
	ur.getByIDFn = func(_ context.Context, id uuid.UUID) (*user.User, error) {
		stubOrg := uuid.New()
		return &user.User{ID: id, Role: user.RoleProvider, OrganizationID: &stubOrg, ReferrerEnabled: true}, nil
	}

	_, err := svc.ApplyToJob(context.Background(), ApplyToJobInput{
		JobID:       j.ID,
		ApplicantID: uuid.New(),
		Kind:        domain.ApplicantKindAgency,
		Message:     "should fail",
	})
	assert.ErrorIs(t, err, domain.ErrInvalidApplicantKind)
}

// --- ListJobApplications filter tests ---

func TestListJobApplications_FilterByKind(t *testing.T) {
	svc, jr, ar, _, _, _ := newTestApplyService()
	creatorID := uuid.New()
	j := openJob(creatorID)
	jr.getByIDFn = func(_ context.Context, _ uuid.UUID) (*domain.Job, error) { return j, nil }
	ar.listByJobFn = func(_ context.Context, _ uuid.UUID, _ string, _ int, kindFilter domain.ApplicantKind) ([]*domain.JobApplication, string, error) {
		// Echo the filter back as a marker — we assert it on the mock.
		return []*domain.JobApplication{}, "", nil
	}

	_, _, err := svc.ListJobApplications(context.Background(), j.ID, creatorID, "", 20, ListJobApplicationsFilter{Kind: domain.ApplicantKindReferrer})
	require.NoError(t, err)
	assert.Equal(t, domain.ApplicantKindReferrer, ar.listByJobLastKind, "filter should be forwarded to the repository")
}

func TestListJobApplications_RejectsInvalidKind(t *testing.T) {
	svc, jr, _, _, _, _ := newTestApplyService()
	creatorID := uuid.New()
	j := openJob(creatorID)
	jr.getByIDFn = func(_ context.Context, _ uuid.UUID) (*domain.Job, error) { return j, nil }

	_, _, err := svc.ListJobApplications(context.Background(), j.ID, creatorID, "", 20, ListJobApplicationsFilter{Kind: domain.ApplicantKind("hacker")})
	assert.ErrorIs(t, err, domain.ErrInvalidApplicantKind)
}

func TestListJobApplications_EmptyKind_NoFilter(t *testing.T) {
	svc, jr, ar, _, _, _ := newTestApplyService()
	creatorID := uuid.New()
	j := openJob(creatorID)
	jr.getByIDFn = func(_ context.Context, _ uuid.UUID) (*domain.Job, error) { return j, nil }

	_, _, err := svc.ListJobApplications(context.Background(), j.ID, creatorID, "", 20, ListJobApplicationsFilter{})
	require.NoError(t, err)
	assert.Equal(t, domain.ApplicantKind(""), ar.listByJobLastKind, "empty filter must not add a SQL clause")
}

func TestApplyToJob_KYCNotBlocked_OK(t *testing.T) {
	svc, jr, _, ur, _, _, or := newTestApplyServiceFull()
	creatorID := uuid.New()
	applicantID := uuid.New()
	j := openJob(creatorID)
	past5 := time.Now().Add(-5 * 24 * time.Hour)

	jr.getByIDFn = func(_ context.Context, _ uuid.UUID) (*domain.Job, error) { return j, nil }
	ur.getByIDFn = func(_ context.Context, id uuid.UUID) (*user.User, error) {
		stubOrg := uuid.New()
		return &user.User{ID: id, Role: user.RoleProvider, OrganizationID: &stubOrg}, nil
	}
	or.findByUserIDFn = func(_ context.Context, _ uuid.UUID) (*organization.Organization, error) {
		// 5 days elapsed — below the 14-day deadline, should still pass.
		return &organization.Organization{
			ID:                uuid.New(),
			Type:              organization.OrgTypeProviderPersonal,
			KYCFirstEarningAt: &past5,
		}, nil
	}

	app, err := svc.ApplyToJob(context.Background(), ApplyToJobInput{
		JobID: j.ID, ApplicantID: applicantID, Message: "I am interested",
	})
	assert.NoError(t, err)
	assert.NotNil(t, app)
}
