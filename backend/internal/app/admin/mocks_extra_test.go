package admin

// Additional mocks for the extended admin service tests. Kept in a
// separate file from mocks_test.go to keep diffs minimal and avoid
// merge conflicts with the SEC-05/SEC-13 test mocks.
//
// Each mock is a struct with function fields the test sets up per
// scenario (project's manual-mock convention) plus optional spies
// (counts, captured args) accessed via mu-protected accessors.

import (
	"context"
	"io"
	"sync"
	"time"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/job"
	"marketplace-backend/internal/domain/media"
	"marketplace-backend/internal/domain/milestone"
	"marketplace-backend/internal/domain/moderation"
	"marketplace-backend/internal/domain/proposal"
	"marketplace-backend/internal/domain/report"
	"marketplace-backend/internal/domain/review"
	"marketplace-backend/internal/port/repository"
	portservice "marketplace-backend/internal/port/service"
)

// --- mockReportRepo ---

var _ repository.ReportRepository = (*mockReportRepo)(nil)

type mockReportRepo struct {
	mu sync.Mutex

	listByConversationFn     func(ctx context.Context, id uuid.UUID) ([]*report.Report, error)
	listByUserInvolvedFn     func(ctx context.Context, id uuid.UUID) (against []*report.Report, filed []*report.Report, err error)
	listByTargetFn           func(ctx context.Context, t string, id uuid.UUID) ([]*report.Report, error)
	updateStatusFn           func(ctx context.Context, id uuid.UUID, status string, note string, by uuid.UUID) error
	pendingCountsByTargetsFn func(ctx context.Context, t string, ids []uuid.UUID) (map[uuid.UUID]int, error)

	updateStatusCalls []reportUpdateCall
}

type reportUpdateCall struct {
	ID         uuid.UUID
	Status     string
	Note       string
	ResolvedBy uuid.UUID
}

func (m *mockReportRepo) Create(_ context.Context, _ *report.Report) error { return nil }
func (m *mockReportRepo) GetByID(_ context.Context, _ uuid.UUID) (*report.Report, error) {
	return nil, nil
}
func (m *mockReportRepo) ListByStatus(_ context.Context, _ string, _ string, _ int) ([]*report.Report, string, error) {
	return nil, "", nil
}
func (m *mockReportRepo) ListByReporter(_ context.Context, _ uuid.UUID, _ string, _ int) ([]*report.Report, string, error) {
	return nil, "", nil
}
func (m *mockReportRepo) ListByTarget(ctx context.Context, t string, id uuid.UUID) ([]*report.Report, error) {
	if m.listByTargetFn != nil {
		return m.listByTargetFn(ctx, t, id)
	}
	return nil, nil
}
func (m *mockReportRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string, note string, by uuid.UUID) error {
	m.mu.Lock()
	m.updateStatusCalls = append(m.updateStatusCalls, reportUpdateCall{ID: id, Status: status, Note: note, ResolvedBy: by})
	m.mu.Unlock()
	if m.updateStatusFn != nil {
		return m.updateStatusFn(ctx, id, status, note, by)
	}
	return nil
}
func (m *mockReportRepo) HasPendingReport(_ context.Context, _ uuid.UUID, _ string, _ uuid.UUID) (bool, error) {
	return false, nil
}
func (m *mockReportRepo) ListByConversation(ctx context.Context, id uuid.UUID) ([]*report.Report, error) {
	if m.listByConversationFn != nil {
		return m.listByConversationFn(ctx, id)
	}
	return nil, nil
}
func (m *mockReportRepo) ListByUserInvolved(ctx context.Context, id uuid.UUID) ([]*report.Report, []*report.Report, error) {
	if m.listByUserInvolvedFn != nil {
		return m.listByUserInvolvedFn(ctx, id)
	}
	return nil, nil, nil
}
func (m *mockReportRepo) PendingCountsByTargets(ctx context.Context, t string, ids []uuid.UUID) (map[uuid.UUID]int, error) {
	if m.pendingCountsByTargetsFn != nil {
		return m.pendingCountsByTargetsFn(ctx, t, ids)
	}
	return map[uuid.UUID]int{}, nil
}

func (m *mockReportRepo) snapshotUpdateStatusCalls() []reportUpdateCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]reportUpdateCall, len(m.updateStatusCalls))
	copy(out, m.updateStatusCalls)
	return out
}

// --- mockJobsRepo ---

var _ repository.JobRepository = (*mockJobsRepo)(nil)

type mockJobsRepo struct {
	listAdminFn  func(ctx context.Context, f repository.AdminJobFilters) ([]repository.AdminJob, string, error)
	countAdminFn func(ctx context.Context, f repository.AdminJobFilters) (int, error)
	getAdminFn   func(ctx context.Context, id uuid.UUID) (*repository.AdminJob, error)
	deleteFn     func(ctx context.Context, id uuid.UUID) error
	countAllFn   func(ctx context.Context) (int, int, error)
}

func (m *mockJobsRepo) Create(_ context.Context, _ *job.Job) error               { return nil }
func (m *mockJobsRepo) GetByID(_ context.Context, _ uuid.UUID) (*job.Job, error) { return nil, nil }
func (m *mockJobsRepo) Update(_ context.Context, _ *job.Job) error               { return nil }
func (m *mockJobsRepo) ListByOrganization(_ context.Context, _ uuid.UUID, _ string, _ int) ([]*job.Job, string, error) {
	return nil, "", nil
}
func (m *mockJobsRepo) ListOpen(_ context.Context, _ repository.JobListFilters, _ string, _ int) ([]*job.Job, string, error) {
	return nil, "", nil
}
func (m *mockJobsRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}
func (m *mockJobsRepo) ListAdmin(ctx context.Context, f repository.AdminJobFilters) ([]repository.AdminJob, string, error) {
	if m.listAdminFn != nil {
		return m.listAdminFn(ctx, f)
	}
	return nil, "", nil
}
func (m *mockJobsRepo) CountAdmin(ctx context.Context, f repository.AdminJobFilters) (int, error) {
	if m.countAdminFn != nil {
		return m.countAdminFn(ctx, f)
	}
	return 0, nil
}
func (m *mockJobsRepo) GetAdmin(ctx context.Context, id uuid.UUID) (*repository.AdminJob, error) {
	if m.getAdminFn != nil {
		return m.getAdminFn(ctx, id)
	}
	return &repository.AdminJob{ID: id}, nil
}
func (m *mockJobsRepo) CountAll(ctx context.Context) (int, int, error) {
	if m.countAllFn != nil {
		return m.countAllFn(ctx)
	}
	return 0, 0, nil
}

// --- mockApplicationsRepo ---

var _ repository.JobApplicationRepository = (*mockApplicationsRepo)(nil)

type mockApplicationsRepo struct {
	listAdminFn  func(ctx context.Context, f repository.AdminApplicationFilters) ([]repository.AdminJobApplication, string, error)
	countAdminFn func(ctx context.Context, f repository.AdminApplicationFilters) (int, error)
	deleteFn     func(ctx context.Context, id uuid.UUID) error
}

func (m *mockApplicationsRepo) Create(_ context.Context, _ *job.JobApplication) error { return nil }
func (m *mockApplicationsRepo) GetByID(_ context.Context, _ uuid.UUID) (*job.JobApplication, error) {
	return nil, nil
}
func (m *mockApplicationsRepo) GetByJobAndApplicant(_ context.Context, _ uuid.UUID, _ uuid.UUID) (*job.JobApplication, error) {
	return nil, nil
}
func (m *mockApplicationsRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}
func (m *mockApplicationsRepo) ListByJob(_ context.Context, _ uuid.UUID, _ string, _ int) ([]*job.JobApplication, string, error) {
	return nil, "", nil
}
func (m *mockApplicationsRepo) ListByApplicantOrganization(_ context.Context, _ uuid.UUID, _ string, _ int) ([]*job.JobApplication, string, error) {
	return nil, "", nil
}
func (m *mockApplicationsRepo) CountByJob(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}
func (m *mockApplicationsRepo) ListAdmin(ctx context.Context, f repository.AdminApplicationFilters) ([]repository.AdminJobApplication, string, error) {
	if m.listAdminFn != nil {
		return m.listAdminFn(ctx, f)
	}
	return nil, "", nil
}
func (m *mockApplicationsRepo) CountAdmin(ctx context.Context, f repository.AdminApplicationFilters) (int, error) {
	if m.countAdminFn != nil {
		return m.countAdminFn(ctx, f)
	}
	return 0, nil
}

// --- mockProposalRepo ---

var _ repository.ProposalRepository = (*mockProposalRepo)(nil)

type mockProposalRepo struct {
	countAllFn func(ctx context.Context) (int, int, error)
}

func (m *mockProposalRepo) Create(_ context.Context, _ *proposal.Proposal) error { return nil }
func (m *mockProposalRepo) CreateWithDocuments(_ context.Context, _ *proposal.Proposal, _ []*proposal.ProposalDocument) error {
	return nil
}
func (m *mockProposalRepo) CreateWithDocumentsAndMilestones(_ context.Context, _ *proposal.Proposal, _ []*proposal.ProposalDocument, _ []*milestone.Milestone) error {
	return nil
}
func (m *mockProposalRepo) GetByID(_ context.Context, _ uuid.UUID) (*proposal.Proposal, error) {
	return nil, nil
}
func (m *mockProposalRepo) GetByIDForOrg(_ context.Context, _, _ uuid.UUID) (*proposal.Proposal, error) {
	return nil, nil
}
func (m *mockProposalRepo) GetByIDs(_ context.Context, _ []uuid.UUID) ([]*proposal.Proposal, error) {
	return nil, nil
}
func (m *mockProposalRepo) Update(_ context.Context, _ *proposal.Proposal) error { return nil }
func (m *mockProposalRepo) GetLatestVersion(_ context.Context, _ uuid.UUID) (*proposal.Proposal, error) {
	return nil, nil
}
func (m *mockProposalRepo) ListByConversation(_ context.Context, _ uuid.UUID) ([]*proposal.Proposal, error) {
	return nil, nil
}
func (m *mockProposalRepo) ListActiveProjectsByOrganization(_ context.Context, _ uuid.UUID, _ string, _ int) ([]*proposal.Proposal, string, error) {
	return nil, "", nil
}
func (m *mockProposalRepo) ListCompletedByOrganization(_ context.Context, _ uuid.UUID, _ string, _ int) ([]*proposal.Proposal, string, error) {
	return nil, "", nil
}
func (m *mockProposalRepo) GetDocuments(_ context.Context, _ uuid.UUID) ([]*proposal.ProposalDocument, error) {
	return nil, nil
}
func (m *mockProposalRepo) CreateDocument(_ context.Context, _ *proposal.ProposalDocument) error {
	return nil
}
func (m *mockProposalRepo) IsOrgAuthorizedForProposal(_ context.Context, _ uuid.UUID, _ uuid.UUID) (bool, error) {
	return true, nil
}
func (m *mockProposalRepo) CountAll(ctx context.Context) (int, int, error) {
	if m.countAllFn != nil {
		return m.countAllFn(ctx)
	}
	return 0, 0, nil
}
func (m *mockProposalRepo) SumPaidByClientOrganization(_ context.Context, _ uuid.UUID) (int64, error) {
	return 0, nil
}
func (m *mockProposalRepo) ListCompletedByClientOrganization(_ context.Context, _ uuid.UUID, _ int) ([]*proposal.Proposal, error) {
	return nil, nil
}

// --- mockAdminConversationsRepo ---

var _ repository.AdminConversationRepository = (*mockAdminConversationsRepo)(nil)

type mockAdminConversationsRepo struct {
	listFn         func(ctx context.Context, f repository.AdminConversationFilters) ([]repository.AdminConversation, string, int, error)
	getByIDFn      func(ctx context.Context, id uuid.UUID) (*repository.AdminConversation, error)
	listMessagesFn func(ctx context.Context, id uuid.UUID, c string, l int) ([]repository.AdminMessage, string, error)
}

func (m *mockAdminConversationsRepo) List(ctx context.Context, f repository.AdminConversationFilters) ([]repository.AdminConversation, string, int, error) {
	if m.listFn != nil {
		return m.listFn(ctx, f)
	}
	return nil, "", 0, nil
}
func (m *mockAdminConversationsRepo) GetByID(ctx context.Context, id uuid.UUID) (*repository.AdminConversation, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return &repository.AdminConversation{ID: id}, nil
}
func (m *mockAdminConversationsRepo) ListMessages(ctx context.Context, id uuid.UUID, c string, l int) ([]repository.AdminMessage, string, error) {
	if m.listMessagesFn != nil {
		return m.listMessagesFn(ctx, id, c, l)
	}
	return nil, "", nil
}

// --- mockMediaRepo ---

var _ repository.MediaRepository = (*mockMediaRepo)(nil)

type mockMediaRepo struct {
	mu             sync.Mutex
	getByIDFn      func(ctx context.Context, id uuid.UUID) (*media.Media, error)
	updateFn       func(ctx context.Context, m *media.Media) error
	deleteFn       func(ctx context.Context, id uuid.UUID) error
	listAdminFn    func(ctx context.Context, f repository.AdminMediaFilters) ([]repository.AdminMediaItem, error)
	countAdminFn   func(ctx context.Context, f repository.AdminMediaFilters) (int, error)
	getAdminByIDFn func(ctx context.Context, id uuid.UUID) (*repository.AdminMediaItem, error)

	updates []*media.Media
}

func (m *mockMediaRepo) Create(_ context.Context, _ *media.Media) error { return nil }
func (m *mockMediaRepo) GetByID(ctx context.Context, id uuid.UUID) (*media.Media, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return &media.Media{ID: id, FileURL: "x"}, nil
}
func (m *mockMediaRepo) GetAdminByID(ctx context.Context, id uuid.UUID) (*repository.AdminMediaItem, error) {
	if m.getAdminByIDFn != nil {
		return m.getAdminByIDFn(ctx, id)
	}
	return &repository.AdminMediaItem{Media: media.Media{ID: id}}, nil
}
func (m *mockMediaRepo) GetByJobID(_ context.Context, _ string) (*media.Media, error) {
	return nil, nil
}
func (m *mockMediaRepo) Update(ctx context.Context, mm *media.Media) error {
	m.mu.Lock()
	cp := *mm
	m.updates = append(m.updates, &cp)
	m.mu.Unlock()
	if m.updateFn != nil {
		return m.updateFn(ctx, mm)
	}
	return nil
}
func (m *mockMediaRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}
func (m *mockMediaRepo) ListAdmin(ctx context.Context, f repository.AdminMediaFilters) ([]repository.AdminMediaItem, error) {
	if m.listAdminFn != nil {
		return m.listAdminFn(ctx, f)
	}
	return nil, nil
}
func (m *mockMediaRepo) CountAdmin(ctx context.Context, f repository.AdminMediaFilters) (int, error) {
	if m.countAdminFn != nil {
		return m.countAdminFn(ctx, f)
	}
	return 0, nil
}
func (m *mockMediaRepo) ClearSource(_ context.Context, _ string, _ uuid.UUID) error {
	return nil
}
func (m *mockMediaRepo) CountRejectedByUploader(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}

func (m *mockMediaRepo) snapshotUpdates() []*media.Media {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*media.Media, len(m.updates))
	copy(out, m.updates)
	return out
}

// --- mockReviewRepo ---

var _ repository.ReviewRepository = (*mockReviewRepo)(nil)

type mockReviewRepo struct {
	listAdminFn    func(ctx context.Context, f repository.AdminReviewFilters) ([]repository.AdminReview, error)
	countAdminFn   func(ctx context.Context, f repository.AdminReviewFilters) (int, error)
	getAdminByIDFn func(ctx context.Context, id uuid.UUID) (*repository.AdminReview, error)
	deleteAdminFn  func(ctx context.Context, id uuid.UUID) error
}

func (m *mockReviewRepo) Create(_ context.Context, _ *review.Review) error { return nil }
func (m *mockReviewRepo) CreateAndMaybeReveal(_ context.Context, r *review.Review) (*review.Review, error) {
	return r, nil
}
func (m *mockReviewRepo) GetByID(_ context.Context, _ uuid.UUID) (*review.Review, error) {
	return nil, nil
}
func (m *mockReviewRepo) GetByIDForOrg(_ context.Context, _, _ uuid.UUID) (*review.Review, error) {
	return nil, nil
}
func (m *mockReviewRepo) ListByReviewedOrganization(_ context.Context, _ uuid.UUID, _ string, _ int) ([]*review.Review, string, error) {
	return nil, "", nil
}
func (m *mockReviewRepo) GetAverageRatingByOrganization(_ context.Context, _ uuid.UUID) (*review.AverageRating, error) {
	return nil, nil
}
func (m *mockReviewRepo) ListClientReviewsByOrganization(_ context.Context, _ uuid.UUID, _ int) ([]*review.Review, error) {
	return nil, nil
}
func (m *mockReviewRepo) GetClientAverageRating(_ context.Context, _ uuid.UUID) (*review.AverageRating, error) {
	return nil, nil
}
func (m *mockReviewRepo) HasReviewed(_ context.Context, _ uuid.UUID, _ uuid.UUID) (bool, error) {
	return false, nil
}
func (m *mockReviewRepo) GetByProposalIDs(_ context.Context, _ []uuid.UUID, _ string) (map[uuid.UUID]*review.Review, error) {
	return nil, nil
}
func (m *mockReviewRepo) ListAdmin(ctx context.Context, f repository.AdminReviewFilters) ([]repository.AdminReview, error) {
	if m.listAdminFn != nil {
		return m.listAdminFn(ctx, f)
	}
	return nil, nil
}
func (m *mockReviewRepo) CountAdmin(ctx context.Context, f repository.AdminReviewFilters) (int, error) {
	if m.countAdminFn != nil {
		return m.countAdminFn(ctx, f)
	}
	return 0, nil
}
func (m *mockReviewRepo) GetAdminByID(ctx context.Context, id uuid.UUID) (*repository.AdminReview, error) {
	if m.getAdminByIDFn != nil {
		return m.getAdminByIDFn(ctx, id)
	}
	return &repository.AdminReview{Review: review.Review{ID: id}}, nil
}
func (m *mockReviewRepo) DeleteAdmin(ctx context.Context, id uuid.UUID) error {
	if m.deleteAdminFn != nil {
		return m.deleteAdminFn(ctx, id)
	}
	return nil
}

// --- mockModerationRepo ---

var _ repository.AdminModerationRepository = (*mockModerationRepo)(nil)

type mockModerationRepo struct {
	listFn         func(ctx context.Context, f repository.ModerationFilters) ([]repository.ModerationItem, error)
	countFn        func(ctx context.Context, f repository.ModerationFilters) (int, error)
	pendingCountFn func(ctx context.Context) (int, error)
}

func (m *mockModerationRepo) List(ctx context.Context, f repository.ModerationFilters) ([]repository.ModerationItem, error) {
	if m.listFn != nil {
		return m.listFn(ctx, f)
	}
	return nil, nil
}
func (m *mockModerationRepo) Count(ctx context.Context, f repository.ModerationFilters) (int, error) {
	if m.countFn != nil {
		return m.countFn(ctx, f)
	}
	return 0, nil
}
func (m *mockModerationRepo) PendingCount(ctx context.Context) (int, error) {
	if m.pendingCountFn != nil {
		return m.pendingCountFn(ctx)
	}
	return 0, nil
}

// --- mockModerationResultsRepo ---

var _ repository.ModerationResultsRepository = (*mockModerationResultsRepo)(nil)

type mockModerationResultsRepo struct {
	mu             sync.Mutex
	markReviewedFn func(ctx context.Context, ct moderation.ContentType, id uuid.UUID, reviewerID uuid.UUID, st moderation.Status) error

	markCalls []moderationMarkCall
}

type moderationMarkCall struct {
	ContentType moderation.ContentType
	ContentID   uuid.UUID
	ReviewerID  uuid.UUID
	NewStatus   moderation.Status
}

func (m *mockModerationResultsRepo) Upsert(_ context.Context, _ *moderation.Result) error {
	return nil
}
func (m *mockModerationResultsRepo) GetByContent(_ context.Context, _ moderation.ContentType, _ uuid.UUID) (*moderation.Result, error) {
	return nil, nil
}
func (m *mockModerationResultsRepo) List(_ context.Context, _ repository.ModerationResultsFilters) ([]*moderation.Result, int, error) {
	return nil, 0, nil
}
func (m *mockModerationResultsRepo) MarkReviewed(ctx context.Context, ct moderation.ContentType, id uuid.UUID, reviewerID uuid.UUID, st moderation.Status) error {
	m.mu.Lock()
	m.markCalls = append(m.markCalls, moderationMarkCall{ContentType: ct, ContentID: id, ReviewerID: reviewerID, NewStatus: st})
	m.mu.Unlock()
	if m.markReviewedFn != nil {
		return m.markReviewedFn(ctx, ct, id, reviewerID, st)
	}
	return nil
}

func (m *mockModerationResultsRepo) snapshotMarkCalls() []moderationMarkCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]moderationMarkCall, len(m.markCalls))
	copy(out, m.markCalls)
	return out
}

// --- mockStorageService ---

var _ portservice.StorageService = (*mockStorageService)(nil)

type mockStorageService struct {
	mu          sync.Mutex
	deleteCalls []string
	deleteErr   error
}

func (m *mockStorageService) Upload(_ context.Context, _ string, _ io.Reader, _ string, _ int64) (string, error) {
	return "", nil
}
func (m *mockStorageService) Delete(_ context.Context, key string) error {
	m.mu.Lock()
	m.deleteCalls = append(m.deleteCalls, key)
	m.mu.Unlock()
	return m.deleteErr
}
func (m *mockStorageService) GetPublicURL(_ string) string                              { return "" }
func (m *mockStorageService) GetPresignedUploadURL(_ context.Context, _ string, _ string, _ time.Duration) (string, error) {
	return "", nil
}
func (m *mockStorageService) GetPresignedDownloadURL(_ context.Context, _ string, _ time.Duration) (string, error) {
	return "", nil
}
func (m *mockStorageService) GetPresignedDownloadURLAsAttachment(_ context.Context, _ string, _ string, _ time.Duration) (string, error) {
	return "", nil
}
func (m *mockStorageService) Download(_ context.Context, _ string) ([]byte, error) {
	return nil, nil
}

func (m *mockStorageService) snapshotDeleteCalls() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]string, len(m.deleteCalls))
	copy(out, m.deleteCalls)
	return out
}

// --- mockAdminNotifier ---

var _ portservice.AdminNotifierService = (*mockAdminNotifier)(nil)

type mockAdminNotifier struct {
	getAllFn func(ctx context.Context, adminID uuid.UUID) (map[string]int64, error)
	resetFn  func(ctx context.Context, adminID uuid.UUID, category string) error
}

func (m *mockAdminNotifier) IncrementAll(_ context.Context, _ string) error { return nil }
func (m *mockAdminNotifier) GetAll(ctx context.Context, adminID uuid.UUID) (map[string]int64, error) {
	if m.getAllFn != nil {
		return m.getAllFn(ctx, adminID)
	}
	return map[string]int64{}, nil
}
func (m *mockAdminNotifier) Reset(ctx context.Context, adminID uuid.UUID, category string) error {
	if m.resetFn != nil {
		return m.resetFn(ctx, adminID, category)
	}
	return nil
}
