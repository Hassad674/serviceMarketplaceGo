package admin

// Extended admin service tests covering the surfaces that were left
// at 0% coverage by the BUG-NEW-09 / SEC-05 round:
//   - GetDashboardStats (counters + degraded paths)
//   - GetNotificationCounters / ResetNotificationCounter
//   - ListUsers / GetUser
//   - ListConversations / GetConversation / GetConversationMessages
//   - ListConversationReports / ListUserReports / ResolveReport
//   - ListJobs / GetJob / DeleteJob / ListJobApplications / DeleteJobApplication / ListJobReports
//   - ListMedia / GetMedia / ApproveMedia / RejectMedia / DeleteMedia
//   - ListReviews / GetReview / DeleteReview / ListReviewReports
//   - ListModerationItems / ModerationPendingCount
//   - ApproveMessageModeration / HideMessage / RestoreMessageModeration
//   - ApproveReviewModeration / RestoreReviewModeration / RestoreModeration

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/domain/audit"
	"marketplace-backend/internal/domain/media"
	"marketplace-backend/internal/domain/moderation"
	"marketplace-backend/internal/domain/report"
	"marketplace-backend/internal/domain/review"
	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/port/repository"
)

// extraFixture wraps the full set of mocks used by the extended tests.
type extraFixture struct {
	users         *mockUserRepo
	reports       *mockReportRepo
	jobs          *mockJobsRepo
	apps          *mockApplicationsRepo
	proposals     *mockProposalRepo
	convs         *mockAdminConversationsRepo
	medias        *mockMediaRepo
	reviews       *mockReviewRepo
	moderation    *mockModerationRepo
	modResults    *mockModerationResultsRepo
	storage       *mockStorageService
	adminNotifier *mockAdminNotifier
	audit         *mockAuditRepo
	svc           *Service
}

func newExtraFixture() *extraFixture {
	users := &mockUserRepo{}
	reports := &mockReportRepo{}
	jobs := &mockJobsRepo{}
	apps := &mockApplicationsRepo{}
	proposals := &mockProposalRepo{}
	convs := &mockAdminConversationsRepo{}
	medias := &mockMediaRepo{}
	reviews := &mockReviewRepo{}
	moderationRepo := &mockModerationRepo{}
	modResults := &mockModerationResultsRepo{}
	storage := &mockStorageService{}
	adminNotifier := &mockAdminNotifier{}
	audits := &mockAuditRepo{}
	svc := NewService(ServiceDeps{
		Users:              users,
		Reports:            reports,
		Reviews:            reviews,
		Jobs:               jobs,
		Applications:       apps,
		Proposals:          proposals,
		AdminConversations: convs,
		MediaRepo:          medias,
		ModerationRepo:     moderationRepo,
		ModerationResults:  modResults,
		Audit:              audits,
		StorageSvc:         storage,
		AdminNotifier:      adminNotifier,
	})
	return &extraFixture{
		users:         users,
		reports:       reports,
		jobs:          jobs,
		apps:          apps,
		proposals:     proposals,
		convs:         convs,
		medias:        medias,
		reviews:       reviews,
		moderation:    moderationRepo,
		modResults:    modResults,
		storage:       storage,
		adminNotifier: adminNotifier,
		audit:         audits,
		svc:           svc,
	}
}

// ─── notification counters ────────────────────────────────────────────

func TestService_GetNotificationCounters_NilNotifier_ReturnsEmpty(t *testing.T) {
	svc := NewService(ServiceDeps{})
	counters, err := svc.GetNotificationCounters(context.Background(), uuid.New())
	require.NoError(t, err)
	assert.Empty(t, counters)
}

func TestService_GetNotificationCounters_WithNotifier(t *testing.T) {
	f := newExtraFixture()
	f.adminNotifier.getAllFn = func(_ context.Context, _ uuid.UUID) (map[string]int64, error) {
		return map[string]int64{"reports": 7}, nil
	}
	counters, err := f.svc.GetNotificationCounters(context.Background(), uuid.New())
	require.NoError(t, err)
	assert.Equal(t, int64(7), counters["reports"])
}

func TestService_GetNotificationCounters_NotifierError_PropagatesWrapped(t *testing.T) {
	f := newExtraFixture()
	f.adminNotifier.getAllFn = func(_ context.Context, _ uuid.UUID) (map[string]int64, error) {
		return nil, errors.New("redis down")
	}
	_, err := f.svc.GetNotificationCounters(context.Background(), uuid.New())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "get admin notification counters")
}

func TestService_ResetNotificationCounter_NilNotifier_NoOp(t *testing.T) {
	svc := NewService(ServiceDeps{})
	err := svc.ResetNotificationCounter(context.Background(), uuid.New(), "reports")
	require.NoError(t, err)
}

func TestService_ResetNotificationCounter_PropagatesError(t *testing.T) {
	f := newExtraFixture()
	f.adminNotifier.resetFn = func(_ context.Context, _ uuid.UUID, _ string) error {
		return errors.New("boom")
	}
	err := f.svc.ResetNotificationCounter(context.Background(), uuid.New(), "x")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reset admin notification counter")
}

// ─── dashboard stats ──────────────────────────────────────────────────

// stubUserRepoForDashboard adds the methods GetDashboardStats relies on.
type stubUserRepoForDashboard struct {
	mockUserRepo
	roleCount   map[string]int
	statusCount map[string]int
	recent      []*user.User
	roleErr     error
	statusErr   error
	recentErr   error
}

func (s *stubUserRepoForDashboard) CountByRole(_ context.Context) (map[string]int, error) {
	return s.roleCount, s.roleErr
}
func (s *stubUserRepoForDashboard) CountByStatus(_ context.Context) (map[string]int, error) {
	return s.statusCount, s.statusErr
}
func (s *stubUserRepoForDashboard) RecentSignups(_ context.Context, _ int) ([]*user.User, error) {
	return s.recent, s.recentErr
}

func TestService_GetDashboardStats_HappyPath(t *testing.T) {
	users := &stubUserRepoForDashboard{
		roleCount:   map[string]int{"agency": 3, "provider": 5, "enterprise": 2},
		statusCount: map[string]int{"active": 8, "suspended": 1, "banned": 1},
		recent:      []*user.User{{ID: uuid.New(), Email: "u@example.com"}},
	}
	proposals := &mockProposalRepo{
		countAllFn: func(_ context.Context) (int, int, error) { return 10, 5, nil },
	}
	jobs := &mockJobsRepo{
		countAllFn: func(_ context.Context) (int, int, error) { return 4, 2, nil },
	}
	svc := NewService(ServiceDeps{
		Users:     users,
		Proposals: proposals,
		Jobs:      jobs,
	})

	stats, err := svc.GetDashboardStats(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 10, stats.TotalUsers, "10 = 3+5+2")
	assert.Equal(t, 8, stats.ActiveUsers)
	assert.Equal(t, 1, stats.SuspendedUsers)
	assert.Equal(t, 1, stats.BannedUsers)
	assert.Equal(t, 10, stats.TotalProposals)
	assert.Equal(t, 5, stats.ActiveProposals)
	assert.Equal(t, 4, stats.TotalJobs)
	assert.Equal(t, 2, stats.OpenJobs)
	assert.Equal(t, 0, stats.TotalOrganizations, "no orgs repo wired → 0")
	assert.Equal(t, 0, stats.PendingInvitations, "no invitations repo wired → 0")
	require.Len(t, stats.RecentSignups, 1)
}

func TestService_GetDashboardStats_RoleCountError(t *testing.T) {
	users := &stubUserRepoForDashboard{roleErr: errors.New("db down")}
	svc := NewService(ServiceDeps{Users: users})
	_, err := svc.GetDashboardStats(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "count by role")
}

func TestService_GetDashboardStats_StatusCountError(t *testing.T) {
	users := &stubUserRepoForDashboard{
		roleCount: map[string]int{}, statusErr: errors.New("status err"),
	}
	svc := NewService(ServiceDeps{Users: users})
	_, err := svc.GetDashboardStats(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "count by status")
}

func TestService_GetDashboardStats_RecentSignupsError(t *testing.T) {
	users := &stubUserRepoForDashboard{
		roleCount:   map[string]int{},
		statusCount: map[string]int{},
		recentErr:   errors.New("recent err"),
	}
	svc := NewService(ServiceDeps{Users: users})
	_, err := svc.GetDashboardStats(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "recent signups")
}

func TestService_GetDashboardStats_ProposalsError(t *testing.T) {
	users := &stubUserRepoForDashboard{
		roleCount: map[string]int{}, statusCount: map[string]int{},
	}
	proposals := &mockProposalRepo{
		countAllFn: func(_ context.Context) (int, int, error) { return 0, 0, errors.New("prop err") },
	}
	svc := NewService(ServiceDeps{Users: users, Proposals: proposals})
	_, err := svc.GetDashboardStats(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "proposals")
}

func TestService_GetDashboardStats_JobsError(t *testing.T) {
	users := &stubUserRepoForDashboard{
		roleCount: map[string]int{}, statusCount: map[string]int{},
	}
	proposals := &mockProposalRepo{}
	jobs := &mockJobsRepo{
		countAllFn: func(_ context.Context) (int, int, error) { return 0, 0, errors.New("jobs err") },
	}
	svc := NewService(ServiceDeps{Users: users, Proposals: proposals, Jobs: jobs})
	_, err := svc.GetDashboardStats(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "jobs")
}

// ─── ListUsers / GetUser ──────────────────────────────────────────────

// stubUserRepoForList provides the list/count admin methods.
type stubUserRepoForList struct {
	mockUserRepo
	listFn  func(ctx context.Context, f repository.AdminUserFilters) ([]*user.User, string, error)
	countFn func(ctx context.Context, f repository.AdminUserFilters) (int, error)
}

func (s *stubUserRepoForList) ListAdmin(ctx context.Context, f repository.AdminUserFilters) ([]*user.User, string, error) {
	return s.listFn(ctx, f)
}
func (s *stubUserRepoForList) CountAdmin(ctx context.Context, f repository.AdminUserFilters) (int, error) {
	return s.countFn(ctx, f)
}

func TestService_ListUsers_Success(t *testing.T) {
	uid := uuid.New()
	stub := &stubUserRepoForList{
		listFn: func(_ context.Context, _ repository.AdminUserFilters) ([]*user.User, string, error) {
			return []*user.User{{ID: uid, Email: "x"}}, "next", nil
		},
		countFn: func(_ context.Context, _ repository.AdminUserFilters) (int, error) { return 1, nil },
	}
	svc := NewService(ServiceDeps{Users: stub})
	users, next, total, err := svc.ListUsers(context.Background(), repository.AdminUserFilters{Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Equal(t, "next", next)
	require.Len(t, users, 1)
	assert.Equal(t, uid, users[0].ID)
}

func TestService_ListUsers_ListError(t *testing.T) {
	stub := &stubUserRepoForList{
		listFn: func(_ context.Context, _ repository.AdminUserFilters) ([]*user.User, string, error) {
			return nil, "", errors.New("query failed")
		},
	}
	svc := NewService(ServiceDeps{Users: stub})
	_, _, _, err := svc.ListUsers(context.Background(), repository.AdminUserFilters{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "list admin users")
}

func TestService_ListUsers_CountError(t *testing.T) {
	stub := &stubUserRepoForList{
		listFn: func(_ context.Context, _ repository.AdminUserFilters) ([]*user.User, string, error) {
			return nil, "", nil
		},
		countFn: func(_ context.Context, _ repository.AdminUserFilters) (int, error) {
			return 0, errors.New("count err")
		},
	}
	svc := NewService(ServiceDeps{Users: stub})
	_, _, _, err := svc.ListUsers(context.Background(), repository.AdminUserFilters{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "count admin users")
}

func TestService_GetUser_Success(t *testing.T) {
	f := newExtraFixture()
	uid := uuid.New()
	f.users.getByIDFn = func(_ context.Context, _ uuid.UUID) (*user.User, error) {
		return &user.User{ID: uid, Email: "x"}, nil
	}
	got, err := f.svc.GetUser(context.Background(), uid)
	require.NoError(t, err)
	assert.Equal(t, uid, got.ID)
}

func TestService_GetUser_NotFound(t *testing.T) {
	f := newExtraFixture()
	f.users.getByIDFn = func(_ context.Context, _ uuid.UUID) (*user.User, error) {
		return nil, user.ErrUserNotFound
	}
	_, err := f.svc.GetUser(context.Background(), uuid.New())
	require.Error(t, err)
	assert.True(t, errors.Is(err, user.ErrUserNotFound),
		"caller-detectable error must propagate via errors.Is")
}

// ─── conversations ────────────────────────────────────────────────────

func TestService_ListConversations_LimitDefaultedWhenInvalid(t *testing.T) {
	f := newExtraFixture()
	var capturedLimit int
	f.convs.listFn = func(_ context.Context, fl repository.AdminConversationFilters) ([]repository.AdminConversation, string, int, error) {
		capturedLimit = fl.Limit
		return nil, "", 0, nil
	}
	_, _, _, err := f.svc.ListConversations(context.Background(), "", -5, 0, "", "")
	require.NoError(t, err)
	assert.Equal(t, 20, capturedLimit, "negative limit must be normalised to default 20")
}

func TestService_ListConversations_RepoError(t *testing.T) {
	f := newExtraFixture()
	f.convs.listFn = func(_ context.Context, _ repository.AdminConversationFilters) ([]repository.AdminConversation, string, int, error) {
		return nil, "", 0, errors.New("db err")
	}
	_, _, _, err := f.svc.ListConversations(context.Background(), "", 10, 0, "", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "list conversations")
}

func TestService_GetConversation_Success(t *testing.T) {
	f := newExtraFixture()
	id := uuid.New()
	conv, err := f.svc.GetConversation(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, id, conv.ID)
}

func TestService_GetConversation_RepoError(t *testing.T) {
	f := newExtraFixture()
	f.convs.getByIDFn = func(_ context.Context, _ uuid.UUID) (*repository.AdminConversation, error) {
		return nil, errors.New("not found")
	}
	_, err := f.svc.GetConversation(context.Background(), uuid.New())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "get conversation")
}

func TestService_GetConversationMessages_LimitClampedWhenZero(t *testing.T) {
	f := newExtraFixture()
	var captured int
	f.convs.listMessagesFn = func(_ context.Context, _ uuid.UUID, _ string, lim int) ([]repository.AdminMessage, string, error) {
		captured = lim
		return nil, "", nil
	}
	_, _, err := f.svc.GetConversationMessages(context.Background(), uuid.New(), "", 0)
	require.NoError(t, err)
	assert.Equal(t, 50, captured)
}

func TestService_GetConversationMessages_LimitClampedWhenAbove100(t *testing.T) {
	f := newExtraFixture()
	var captured int
	f.convs.listMessagesFn = func(_ context.Context, _ uuid.UUID, _ string, lim int) ([]repository.AdminMessage, string, error) {
		captured = lim
		return nil, "", nil
	}
	_, _, err := f.svc.GetConversationMessages(context.Background(), uuid.New(), "", 999)
	require.NoError(t, err)
	assert.Equal(t, 50, captured)
}

func TestService_GetConversationMessages_RepoError(t *testing.T) {
	f := newExtraFixture()
	f.convs.listMessagesFn = func(_ context.Context, _ uuid.UUID, _ string, _ int) ([]repository.AdminMessage, string, error) {
		return nil, "", errors.New("db err")
	}
	_, _, err := f.svc.GetConversationMessages(context.Background(), uuid.New(), "", 10)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "get conversation messages")
}

// ─── reports ──────────────────────────────────────────────────────────

func TestService_ListConversationReports_RepoError(t *testing.T) {
	f := newExtraFixture()
	f.reports.listByConversationFn = func(_ context.Context, _ uuid.UUID) ([]*report.Report, error) {
		return nil, errors.New("db")
	}
	_, err := f.svc.ListConversationReports(context.Background(), uuid.New())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "list conversation reports")
}

func TestService_ListConversationReports_Success(t *testing.T) {
	f := newExtraFixture()
	rid := uuid.New()
	f.reports.listByConversationFn = func(_ context.Context, _ uuid.UUID) ([]*report.Report, error) {
		return []*report.Report{{ID: rid}}, nil
	}
	rs, err := f.svc.ListConversationReports(context.Background(), uuid.New())
	require.NoError(t, err)
	require.Len(t, rs, 1)
	assert.Equal(t, rid, rs[0].ID)
}

func TestService_ListUserReports_Success(t *testing.T) {
	f := newExtraFixture()
	a := []*report.Report{{ID: uuid.New()}}
	b := []*report.Report{{ID: uuid.New()}, {ID: uuid.New()}}
	f.reports.listByUserInvolvedFn = func(_ context.Context, _ uuid.UUID) ([]*report.Report, []*report.Report, error) {
		return a, b, nil
	}
	against, filed, err := f.svc.ListUserReports(context.Background(), uuid.New())
	require.NoError(t, err)
	assert.Len(t, against, 1)
	assert.Len(t, filed, 2)
}

func TestService_ListUserReports_Error(t *testing.T) {
	f := newExtraFixture()
	f.reports.listByUserInvolvedFn = func(_ context.Context, _ uuid.UUID) ([]*report.Report, []*report.Report, error) {
		return nil, nil, errors.New("db")
	}
	_, _, err := f.svc.ListUserReports(context.Background(), uuid.New())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "list user reports")
}

func TestService_ResolveReport_InvalidStatus(t *testing.T) {
	f := newExtraFixture()
	err := f.svc.ResolveReport(context.Background(), uuid.New(), "garbage", "n", uuid.New())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid status")
}

func TestService_ResolveReport_Resolved(t *testing.T) {
	f := newExtraFixture()
	rid := uuid.New()
	by := uuid.New()
	err := f.svc.ResolveReport(context.Background(), rid, string(report.StatusResolved), "looked into", by)
	require.NoError(t, err)
	calls := f.reports.snapshotUpdateStatusCalls()
	require.Len(t, calls, 1)
	assert.Equal(t, rid, calls[0].ID)
	assert.Equal(t, string(report.StatusResolved), calls[0].Status)
	assert.Equal(t, "looked into", calls[0].Note)
	assert.Equal(t, by, calls[0].ResolvedBy)
}

func TestService_ResolveReport_Dismissed(t *testing.T) {
	f := newExtraFixture()
	err := f.svc.ResolveReport(context.Background(), uuid.New(), string(report.StatusDismissed), "n", uuid.New())
	require.NoError(t, err)
}

func TestService_ResolveReport_RepoError(t *testing.T) {
	f := newExtraFixture()
	f.reports.updateStatusFn = func(_ context.Context, _ uuid.UUID, _ string, _ string, _ uuid.UUID) error {
		return errors.New("db")
	}
	err := f.svc.ResolveReport(context.Background(), uuid.New(), string(report.StatusResolved), "n", uuid.New())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "resolve report")
}

// ─── jobs ─────────────────────────────────────────────────────────────

func TestService_ListJobs_NormalisesLimit(t *testing.T) {
	f := newExtraFixture()
	var capturedLim int
	f.jobs.listAdminFn = func(_ context.Context, fl repository.AdminJobFilters) ([]repository.AdminJob, string, error) {
		capturedLim = fl.Limit
		return nil, "", nil
	}
	_, _, _, err := f.svc.ListJobs(context.Background(), "", "", "", "", "", 999, 0)
	require.NoError(t, err)
	assert.Equal(t, 20, capturedLim, "999 must be normalised down to 20")
}

func TestService_ListJobs_LoadsReportCounts(t *testing.T) {
	f := newExtraFixture()
	jobID := uuid.New()
	f.jobs.listAdminFn = func(_ context.Context, _ repository.AdminJobFilters) ([]repository.AdminJob, string, error) {
		return []repository.AdminJob{{ID: jobID, Title: "t"}}, "", nil
	}
	f.reports.pendingCountsByTargetsFn = func(_ context.Context, t string, ids []uuid.UUID) (map[uuid.UUID]int, error) {
		return map[uuid.UUID]int{jobID: 5}, nil
	}
	jobs, _, _, err := f.svc.ListJobs(context.Background(), "", "", "", "", "", 10, 0)
	require.NoError(t, err)
	require.Len(t, jobs, 1)
	assert.Equal(t, 5, jobs[0].PendingReportCount)
}

func TestService_ListJobs_CountError(t *testing.T) {
	f := newExtraFixture()
	f.jobs.countAdminFn = func(_ context.Context, _ repository.AdminJobFilters) (int, error) {
		return 0, errors.New("db")
	}
	_, _, _, err := f.svc.ListJobs(context.Background(), "", "", "", "", "", 10, 0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "list jobs")
}

func TestService_ListJobs_ListError(t *testing.T) {
	f := newExtraFixture()
	f.jobs.listAdminFn = func(_ context.Context, _ repository.AdminJobFilters) ([]repository.AdminJob, string, error) {
		return nil, "", errors.New("db")
	}
	_, _, _, err := f.svc.ListJobs(context.Background(), "", "", "", "", "", 10, 0)
	require.Error(t, err)
}

func TestService_ListJobs_PendingCountsError(t *testing.T) {
	f := newExtraFixture()
	f.jobs.listAdminFn = func(_ context.Context, _ repository.AdminJobFilters) ([]repository.AdminJob, string, error) {
		return []repository.AdminJob{{ID: uuid.New()}}, "", nil
	}
	f.reports.pendingCountsByTargetsFn = func(_ context.Context, _ string, _ []uuid.UUID) (map[uuid.UUID]int, error) {
		return nil, errors.New("rep err")
	}
	_, _, _, err := f.svc.ListJobs(context.Background(), "", "", "", "", "", 10, 0)
	require.Error(t, err)
}

func TestService_GetJob_Success(t *testing.T) {
	f := newExtraFixture()
	id := uuid.New()
	got, err := f.svc.GetJob(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, id, got.ID)
}

func TestService_GetJob_Error(t *testing.T) {
	f := newExtraFixture()
	f.jobs.getAdminFn = func(_ context.Context, _ uuid.UUID) (*repository.AdminJob, error) {
		return nil, errors.New("not found")
	}
	_, err := f.svc.GetJob(context.Background(), uuid.New())
	require.Error(t, err)
}

func TestService_DeleteJob_Success(t *testing.T) {
	f := newExtraFixture()
	err := f.svc.DeleteJob(context.Background(), uuid.New())
	require.NoError(t, err)
}

func TestService_DeleteJob_Error(t *testing.T) {
	f := newExtraFixture()
	f.jobs.deleteFn = func(_ context.Context, _ uuid.UUID) error { return errors.New("db") }
	err := f.svc.DeleteJob(context.Background(), uuid.New())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "delete job")
}

func TestService_ListJobApplications_Success(t *testing.T) {
	f := newExtraFixture()
	appID := uuid.New()
	f.apps.listAdminFn = func(_ context.Context, _ repository.AdminApplicationFilters) ([]repository.AdminJobApplication, string, error) {
		return []repository.AdminJobApplication{{ID: appID}}, "next", nil
	}
	apps, next, _, err := f.svc.ListJobApplications(context.Background(), "", "", "", "", "", 10, 0)
	require.NoError(t, err)
	assert.Equal(t, "next", next)
	require.Len(t, apps, 1)
	assert.Equal(t, appID, apps[0].ID)
}

func TestService_DeleteJobApplication_Error(t *testing.T) {
	f := newExtraFixture()
	f.apps.deleteFn = func(_ context.Context, _ uuid.UUID) error { return errors.New("db") }
	err := f.svc.DeleteJobApplication(context.Background(), uuid.New())
	require.Error(t, err)
}

func TestService_ListJobReports_Success(t *testing.T) {
	f := newExtraFixture()
	rid := uuid.New()
	f.reports.listByTargetFn = func(_ context.Context, target string, _ uuid.UUID) ([]*report.Report, error) {
		assert.Equal(t, string(report.TargetJob), target,
			"job reports MUST query the 'job' target type")
		return []*report.Report{{ID: rid}}, nil
	}
	got, err := f.svc.ListJobReports(context.Background(), uuid.New())
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, rid, got[0].ID)
}

func TestService_ListJobReports_Error(t *testing.T) {
	f := newExtraFixture()
	f.reports.listByTargetFn = func(_ context.Context, _ string, _ uuid.UUID) ([]*report.Report, error) {
		return nil, errors.New("db")
	}
	_, err := f.svc.ListJobReports(context.Background(), uuid.New())
	require.Error(t, err)
}

// ─── media ────────────────────────────────────────────────────────────

func TestService_ApproveMedia_PersistsAndAuditMissing(t *testing.T) {
	f := newExtraFixture()
	mid := uuid.New()
	adminID := uuid.New()
	f.medias.getByIDFn = func(_ context.Context, _ uuid.UUID) (*media.Media, error) {
		return &media.Media{ID: mid}, nil
	}
	err := f.svc.ApproveMedia(context.Background(), mid, adminID)
	require.NoError(t, err)
	updates := f.medias.snapshotUpdates()
	require.Len(t, updates, 1)
	assert.NotNil(t, updates[0].ReviewedBy, "ReviewedBy must be set after Approve")
	assert.Equal(t, adminID, *updates[0].ReviewedBy)
}

func TestService_ApproveMedia_GetByIDError(t *testing.T) {
	f := newExtraFixture()
	f.medias.getByIDFn = func(_ context.Context, _ uuid.UUID) (*media.Media, error) {
		return nil, errors.New("not found")
	}
	err := f.svc.ApproveMedia(context.Background(), uuid.New(), uuid.New())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "approve media")
}

func TestService_ApproveMedia_UpdateError(t *testing.T) {
	f := newExtraFixture()
	f.medias.getByIDFn = func(_ context.Context, _ uuid.UUID) (*media.Media, error) {
		return &media.Media{ID: uuid.New()}, nil
	}
	f.medias.updateFn = func(_ context.Context, _ *media.Media) error {
		return errors.New("save fail")
	}
	err := f.svc.ApproveMedia(context.Background(), uuid.New(), uuid.New())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "approve media: save")
}

func TestService_RejectMedia_Success(t *testing.T) {
	f := newExtraFixture()
	mid := uuid.New()
	f.medias.getByIDFn = func(_ context.Context, _ uuid.UUID) (*media.Media, error) {
		return &media.Media{ID: mid}, nil
	}
	err := f.svc.RejectMedia(context.Background(), mid, uuid.New())
	require.NoError(t, err)
	updates := f.medias.snapshotUpdates()
	require.Len(t, updates, 1)
}

func TestService_RejectMedia_GetError(t *testing.T) {
	f := newExtraFixture()
	f.medias.getByIDFn = func(_ context.Context, _ uuid.UUID) (*media.Media, error) {
		return nil, errors.New("not found")
	}
	err := f.svc.RejectMedia(context.Background(), uuid.New(), uuid.New())
	require.Error(t, err)
}

func TestService_DeleteMedia_DeletesFromStorageAndRepo(t *testing.T) {
	f := newExtraFixture()
	mid := uuid.New()
	f.medias.getByIDFn = func(_ context.Context, _ uuid.UUID) (*media.Media, error) {
		return &media.Media{ID: mid, FileURL: "key/object"}, nil
	}
	err := f.svc.DeleteMedia(context.Background(), mid)
	require.NoError(t, err)
	calls := f.storage.snapshotDeleteCalls()
	require.Len(t, calls, 1)
	assert.Equal(t, "key/object", calls[0])
}

func TestService_DeleteMedia_NoStorageStillDeletes(t *testing.T) {
	users := &mockUserRepo{}
	medias := &mockMediaRepo{}
	medias.getByIDFn = func(_ context.Context, _ uuid.UUID) (*media.Media, error) {
		return &media.Media{ID: uuid.New(), FileURL: "x"}, nil
	}
	svc := NewService(ServiceDeps{Users: users, MediaRepo: medias})
	err := svc.DeleteMedia(context.Background(), uuid.New())
	require.NoError(t, err)
}

func TestService_DeleteMedia_GetError(t *testing.T) {
	f := newExtraFixture()
	f.medias.getByIDFn = func(_ context.Context, _ uuid.UUID) (*media.Media, error) {
		return nil, errors.New("not found")
	}
	err := f.svc.DeleteMedia(context.Background(), uuid.New())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "delete media: get")
}

func TestService_DeleteMedia_RepoDeleteError(t *testing.T) {
	f := newExtraFixture()
	f.medias.getByIDFn = func(_ context.Context, _ uuid.UUID) (*media.Media, error) {
		return &media.Media{ID: uuid.New(), FileURL: "x"}, nil
	}
	f.medias.deleteFn = func(_ context.Context, _ uuid.UUID) error {
		return errors.New("db fail")
	}
	err := f.svc.DeleteMedia(context.Background(), uuid.New())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "delete media: remove")
}

func TestService_ListMedia_Success(t *testing.T) {
	f := newExtraFixture()
	id := uuid.New()
	f.medias.listAdminFn = func(_ context.Context, _ repository.AdminMediaFilters) ([]repository.AdminMediaItem, error) {
		return []repository.AdminMediaItem{{Media: media.Media{ID: id}}}, nil
	}
	f.medias.countAdminFn = func(_ context.Context, _ repository.AdminMediaFilters) (int, error) {
		return 1, nil
	}
	items, total, err := f.svc.ListMedia(context.Background(), repository.AdminMediaFilters{})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, items, 1)
	assert.Equal(t, id, items[0].ID)
}

func TestService_ListMedia_ListError(t *testing.T) {
	f := newExtraFixture()
	f.medias.listAdminFn = func(_ context.Context, _ repository.AdminMediaFilters) ([]repository.AdminMediaItem, error) {
		return nil, errors.New("db")
	}
	_, _, err := f.svc.ListMedia(context.Background(), repository.AdminMediaFilters{})
	require.Error(t, err)
}

func TestService_ListMedia_CountError(t *testing.T) {
	f := newExtraFixture()
	f.medias.countAdminFn = func(_ context.Context, _ repository.AdminMediaFilters) (int, error) {
		return 0, errors.New("db")
	}
	_, _, err := f.svc.ListMedia(context.Background(), repository.AdminMediaFilters{})
	require.Error(t, err)
}

func TestService_GetMedia_Success(t *testing.T) {
	f := newExtraFixture()
	id := uuid.New()
	got, err := f.svc.GetMedia(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, id, got.ID)
}

func TestService_GetMedia_Error(t *testing.T) {
	f := newExtraFixture()
	f.medias.getAdminByIDFn = func(_ context.Context, _ uuid.UUID) (*repository.AdminMediaItem, error) {
		return nil, errors.New("not found")
	}
	_, err := f.svc.GetMedia(context.Background(), uuid.New())
	require.Error(t, err)
}

// ─── reviews ──────────────────────────────────────────────────────────

func TestService_ListReviews_LoadsPendingCounts(t *testing.T) {
	f := newExtraFixture()
	rvID := uuid.New()
	f.reviews.listAdminFn = func(_ context.Context, _ repository.AdminReviewFilters) ([]repository.AdminReview, error) {
		return []repository.AdminReview{{Review: makeReviewWithID(rvID)}}, nil
	}
	f.reviews.countAdminFn = func(_ context.Context, _ repository.AdminReviewFilters) (int, error) {
		return 1, nil
	}
	f.reports.pendingCountsByTargetsFn = func(_ context.Context, target string, ids []uuid.UUID) (map[uuid.UUID]int, error) {
		assert.Equal(t, "review", target)
		return map[uuid.UUID]int{rvID: 4}, nil
	}
	got, total, err := f.svc.ListReviews(context.Background(), repository.AdminReviewFilters{})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, got, 1)
	assert.Equal(t, 4, got[0].PendingReportCount)
}

func TestService_ListReviews_NoRowsSkipsCounts(t *testing.T) {
	f := newExtraFixture()
	f.reviews.listAdminFn = func(_ context.Context, _ repository.AdminReviewFilters) ([]repository.AdminReview, error) {
		return nil, nil
	}
	got, total, err := f.svc.ListReviews(context.Background(), repository.AdminReviewFilters{})
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, got)
}

func TestService_ListReviews_ListErr(t *testing.T) {
	f := newExtraFixture()
	f.reviews.listAdminFn = func(_ context.Context, _ repository.AdminReviewFilters) ([]repository.AdminReview, error) {
		return nil, errors.New("db")
	}
	_, _, err := f.svc.ListReviews(context.Background(), repository.AdminReviewFilters{})
	require.Error(t, err)
}

func TestService_GetReview_Error(t *testing.T) {
	f := newExtraFixture()
	f.reviews.getAdminByIDFn = func(_ context.Context, _ uuid.UUID) (*repository.AdminReview, error) {
		return nil, errors.New("not found")
	}
	_, err := f.svc.GetReview(context.Background(), uuid.New())
	require.Error(t, err)
}

func TestService_GetReview_Success(t *testing.T) {
	f := newExtraFixture()
	id := uuid.New()
	got, err := f.svc.GetReview(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, id, got.ID)
}

func TestService_DeleteReview_Success(t *testing.T) {
	f := newExtraFixture()
	err := f.svc.DeleteReview(context.Background(), uuid.New())
	require.NoError(t, err)
}

func TestService_DeleteReview_Error(t *testing.T) {
	f := newExtraFixture()
	f.reviews.deleteAdminFn = func(_ context.Context, _ uuid.UUID) error { return errors.New("db") }
	err := f.svc.DeleteReview(context.Background(), uuid.New())
	require.Error(t, err)
}

func TestService_ListReviewReports_QueriesReviewTarget(t *testing.T) {
	f := newExtraFixture()
	called := false
	f.reports.listByTargetFn = func(_ context.Context, target string, _ uuid.UUID) ([]*report.Report, error) {
		called = true
		assert.Equal(t, string(report.TargetReview), target)
		return nil, nil
	}
	_, err := f.svc.ListReviewReports(context.Background(), uuid.New())
	require.NoError(t, err)
	assert.True(t, called)
}

func TestService_ListReviewReports_Error(t *testing.T) {
	f := newExtraFixture()
	f.reports.listByTargetFn = func(_ context.Context, _ string, _ uuid.UUID) ([]*report.Report, error) {
		return nil, errors.New("db")
	}
	_, err := f.svc.ListReviewReports(context.Background(), uuid.New())
	require.Error(t, err)
}

// ─── moderation listing ───────────────────────────────────────────────

func TestService_ListModerationItems_NormalisesLimitAndPage(t *testing.T) {
	f := newExtraFixture()
	var captured repository.ModerationFilters
	f.moderation.listFn = func(_ context.Context, fl repository.ModerationFilters) ([]repository.ModerationItem, error) {
		captured = fl
		return nil, nil
	}
	_, _, err := f.svc.ListModerationItems(context.Background(), repository.ModerationFilters{Limit: -1, Page: 0})
	require.NoError(t, err)
	assert.Equal(t, 20, captured.Limit, "negative limit defaulted to 20")
	assert.Equal(t, 1, captured.Page, "zero page defaulted to 1")
}

func TestService_ListModerationItems_ClampsLimit(t *testing.T) {
	f := newExtraFixture()
	var captured repository.ModerationFilters
	f.moderation.listFn = func(_ context.Context, fl repository.ModerationFilters) ([]repository.ModerationItem, error) {
		captured = fl
		return nil, nil
	}
	_, _, err := f.svc.ListModerationItems(context.Background(), repository.ModerationFilters{Limit: 9999, Page: 5})
	require.NoError(t, err)
	assert.Equal(t, 20, captured.Limit, "limit > 100 normalised back to 20 by service")
	assert.Equal(t, 5, captured.Page)
}

func TestService_ListModerationItems_ListError(t *testing.T) {
	f := newExtraFixture()
	f.moderation.listFn = func(_ context.Context, _ repository.ModerationFilters) ([]repository.ModerationItem, error) {
		return nil, errors.New("db")
	}
	_, _, err := f.svc.ListModerationItems(context.Background(), repository.ModerationFilters{Limit: 10, Page: 1})
	require.Error(t, err)
}

func TestService_ListModerationItems_CountError(t *testing.T) {
	f := newExtraFixture()
	f.moderation.listFn = func(_ context.Context, _ repository.ModerationFilters) ([]repository.ModerationItem, error) {
		return nil, nil
	}
	f.moderation.countFn = func(_ context.Context, _ repository.ModerationFilters) (int, error) {
		return 0, errors.New("db")
	}
	_, _, err := f.svc.ListModerationItems(context.Background(), repository.ModerationFilters{Limit: 10, Page: 1})
	require.Error(t, err)
}

func TestService_ModerationPendingCount_Success(t *testing.T) {
	f := newExtraFixture()
	f.moderation.pendingCountFn = func(_ context.Context) (int, error) { return 17, nil }
	got, err := f.svc.ModerationPendingCount(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 17, got)
}

func TestService_ModerationPendingCount_Error(t *testing.T) {
	f := newExtraFixture()
	f.moderation.pendingCountFn = func(_ context.Context) (int, error) { return 0, errors.New("db") }
	_, err := f.svc.ModerationPendingCount(context.Background())
	require.Error(t, err)
}

// ─── message / review moderation overrides ────────────────────────────

func TestService_ApproveMessageModeration_WritesResultsAndAudit(t *testing.T) {
	f := newExtraFixture()
	mid := uuid.New()
	adminID := uuid.New()
	err := f.svc.ApproveMessageModeration(context.Background(), mid, adminID)
	require.NoError(t, err)

	calls := f.modResults.snapshotMarkCalls()
	require.Len(t, calls, 1)
	assert.Equal(t, moderation.ContentTypeMessage, calls[0].ContentType)
	assert.Equal(t, mid, calls[0].ContentID)
	assert.Equal(t, adminID, calls[0].ReviewerID)
	assert.Equal(t, moderation.StatusClean, calls[0].NewStatus)

	auditCalls := f.audit.snapshot()
	require.Len(t, auditCalls, 1)
	assert.Equal(t, audit.Action("moderation.manual_approve_message"), auditCalls[0].Action)
}

func TestService_HideMessage_WritesHidden(t *testing.T) {
	f := newExtraFixture()
	err := f.svc.HideMessage(context.Background(), uuid.New(), uuid.New())
	require.NoError(t, err)
	calls := f.modResults.snapshotMarkCalls()
	require.Len(t, calls, 1)
	assert.Equal(t, moderation.StatusHidden, calls[0].NewStatus)
}

func TestService_RestoreMessageModeration_WritesClean(t *testing.T) {
	f := newExtraFixture()
	err := f.svc.RestoreMessageModeration(context.Background(), uuid.New(), uuid.New())
	require.NoError(t, err)
	calls := f.modResults.snapshotMarkCalls()
	require.Len(t, calls, 1)
	assert.Equal(t, moderation.StatusClean, calls[0].NewStatus)
}

func TestService_ApproveMessageModeration_ResultNotFoundIsIdempotent(t *testing.T) {
	f := newExtraFixture()
	f.modResults.markReviewedFn = func(_ context.Context, _ moderation.ContentType, _ uuid.UUID, _ uuid.UUID, _ moderation.Status) error {
		return moderation.ErrResultNotFound
	}
	err := f.svc.ApproveMessageModeration(context.Background(), uuid.New(), uuid.New())
	require.NoError(t, err, "missing result row must not abort the override (idempotency)")
	auditCalls := f.audit.snapshot()
	require.Len(t, auditCalls, 1, "audit row still fires even when results write was a no-op")
}

func TestService_ApproveMessageModeration_GenericErrorSurfaces(t *testing.T) {
	f := newExtraFixture()
	f.modResults.markReviewedFn = func(_ context.Context, _ moderation.ContentType, _ uuid.UUID, _ uuid.UUID, _ moderation.Status) error {
		return errors.New("db fail")
	}
	err := f.svc.ApproveMessageModeration(context.Background(), uuid.New(), uuid.New())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "results update")
}

func TestService_ApproveReviewModeration_Success(t *testing.T) {
	f := newExtraFixture()
	err := f.svc.ApproveReviewModeration(context.Background(), uuid.New(), uuid.New())
	require.NoError(t, err)
	calls := f.modResults.snapshotMarkCalls()
	require.Len(t, calls, 1)
	assert.Equal(t, moderation.ContentTypeReview, calls[0].ContentType)
}

func TestService_RestoreReviewModeration_Success(t *testing.T) {
	f := newExtraFixture()
	err := f.svc.RestoreReviewModeration(context.Background(), uuid.New(), uuid.New())
	require.NoError(t, err)
	calls := f.modResults.snapshotMarkCalls()
	require.Len(t, calls, 1)
}

func TestService_RestoreModeration_NoResultsWired_Error(t *testing.T) {
	users := &mockUserRepo{}
	svc := NewService(ServiceDeps{Users: users})
	err := svc.RestoreModeration(context.Background(), "job_title", uuid.New(), uuid.New())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "results repo not wired")
}

func TestService_RestoreModeration_PropagatesError(t *testing.T) {
	f := newExtraFixture()
	f.modResults.markReviewedFn = func(_ context.Context, _ moderation.ContentType, _ uuid.UUID, _ uuid.UUID, _ moderation.Status) error {
		return errors.New("boom")
	}
	err := f.svc.RestoreModeration(context.Background(), "profile_about", uuid.New(), uuid.New())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "admin restore moderation")
}

func TestService_RestoreModeration_Success_Audits(t *testing.T) {
	f := newExtraFixture()
	err := f.svc.RestoreModeration(context.Background(), "user_display_name", uuid.New(), uuid.New())
	require.NoError(t, err)
	auditCalls := f.audit.snapshot()
	require.Len(t, auditCalls, 1)
	assert.Equal(t, audit.Action("moderation.manual_restore_user_display_name"), auditCalls[0].Action)
}

// ─── audit nil-safe helpers ────────────────────────────────────────────

func TestService_LogAudit_NilRepoDegradesGracefully(t *testing.T) {
	svc := &Service{}
	// Should not panic — audit is best-effort.
	svc.logAudit(context.Background(), audit.NewEntryInput{Action: audit.ActionAdminUserBan})
}

func TestActorPtr_NilUUIDReturnsNil(t *testing.T) {
	got := actorPtr(uuid.Nil)
	assert.Nil(t, got)
}

func TestActorPtr_NonNilReturnsPointer(t *testing.T) {
	id := uuid.New()
	got := actorPtr(id)
	require.NotNil(t, got)
	assert.Equal(t, id, *got)
}

func TestStringifyTime_Nil(t *testing.T) {
	assert.Equal(t, "", stringifyTime(nil))
}

func TestStringifyTime_RFC3339(t *testing.T) {
	tm := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	got := stringifyTime(&tm)
	assert.Equal(t, "2026-05-01T12:00:00Z", got)
}

// makeReviewWithID is a tiny helper to avoid relying on the internal
// review.Review constructor (which has more required fields than we
// need to test the admin code path).
func makeReviewWithID(id uuid.UUID) review.Review {
	return review.Review{ID: id}
}

// ─── organization (wiring guards only) ────────────────────────────────

// These tests pin the "not wired" paths — every admin-org method
// returns a clear error when the optional Phase-6 dependencies are
// missing, instead of nil-deref panicking. The success paths are
// covered by the dedicated app/organization MembershipService tests.

func TestService_GetUserOrganizationDetail_NotWired_Error(t *testing.T) {
	svc := NewService(ServiceDeps{Users: &mockUserRepo{}})
	_, err := svc.GetUserOrganizationDetail(context.Background(), uuid.New())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "admin organization feature not wired")
}

func TestService_ForceTransferOwnership_NotWired_Error(t *testing.T) {
	svc := NewService(ServiceDeps{Users: &mockUserRepo{}})
	_, err := svc.ForceTransferOwnership(context.Background(), uuid.New(), uuid.New())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "admin organization feature not wired")
}

func TestService_ForceUpdateMemberRole_NotWired_Error(t *testing.T) {
	svc := NewService(ServiceDeps{Users: &mockUserRepo{}})
	_, err := svc.ForceUpdateMemberRole(context.Background(), uuid.New(), uuid.New(), "member")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "admin organization feature not wired")
}

func TestService_ForceRemoveMember_NotWired_Error(t *testing.T) {
	svc := NewService(ServiceDeps{Users: &mockUserRepo{}})
	err := svc.ForceRemoveMember(context.Background(), uuid.New(), uuid.New())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "admin organization feature not wired")
}

func TestService_ForceCancelInvitation_NotWired_Error(t *testing.T) {
	svc := NewService(ServiceDeps{Users: &mockUserRepo{}})
	err := svc.ForceCancelInvitation(context.Background(), uuid.New())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "admin organization feature not wired")
}
