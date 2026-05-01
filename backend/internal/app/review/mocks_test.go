package review

import (
	"context"
	"time"

	"github.com/google/uuid"

	milestonedomain "marketplace-backend/internal/domain/milestone"
	proposaldomain "marketplace-backend/internal/domain/proposal"
	domain "marketplace-backend/internal/domain/review"
	userdomain "marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
)

// mockProposal is a test helper to build proposals with minimal fields.
type mockProposal = proposaldomain.Proposal

// mockReviewRepo implements repository.ReviewRepository for tests.
type mockReviewRepo struct {
	createFn          func(ctx context.Context, r *domain.Review) error
	createAndRevealFn func(ctx context.Context, r *domain.Review) (*domain.Review, error)
	getByIDFn         func(ctx context.Context, id uuid.UUID) (*domain.Review, error)
	listByUserFn      func(ctx context.Context, userID uuid.UUID, cursor string, limit int) ([]*domain.Review, string, error)
	getAverageFn      func(ctx context.Context, userID uuid.UUID) (*domain.AverageRating, error)
	hasReviewedFn     func(ctx context.Context, proposalID, reviewerID uuid.UUID) (bool, error)
}

func (m *mockReviewRepo) Create(ctx context.Context, r *domain.Review) error {
	if m.createFn != nil {
		return m.createFn(ctx, r)
	}
	return nil
}

func (m *mockReviewRepo) CreateAndMaybeReveal(ctx context.Context, r *domain.Review) (*domain.Review, error) {
	if m.createAndRevealFn != nil {
		return m.createAndRevealFn(ctx, r)
	}
	// Default: echo back the provided entity unchanged (still hidden).
	return r, nil
}

func (m *mockReviewRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Review, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, domain.ErrNotFound
}

// GetByIDForOrg defaults to GetByID — review tests do not yet
// exercise the org-scoped path but the port requires the method.
func (m *mockReviewRepo) GetByIDForOrg(ctx context.Context, id, _ uuid.UUID) (*domain.Review, error) {
	return m.GetByID(ctx, id)
}

func (m *mockReviewRepo) ListByReviewedOrganization(ctx context.Context, userID uuid.UUID, cursor string, limit int) ([]*domain.Review, string, error) {
	if m.listByUserFn != nil {
		return m.listByUserFn(ctx, userID, cursor, limit)
	}
	return nil, "", nil
}

func (m *mockReviewRepo) GetAverageRatingByOrganization(ctx context.Context, userID uuid.UUID) (*domain.AverageRating, error) {
	if m.getAverageFn != nil {
		return m.getAverageFn(ctx, userID)
	}
	return &domain.AverageRating{}, nil
}

func (m *mockReviewRepo) ListClientReviewsByOrganization(_ context.Context, _ uuid.UUID, _ int) ([]*domain.Review, error) {
	return nil, nil
}

func (m *mockReviewRepo) GetClientAverageRating(_ context.Context, _ uuid.UUID) (*domain.AverageRating, error) {
	return &domain.AverageRating{}, nil
}

func (m *mockReviewRepo) HasReviewed(ctx context.Context, proposalID, reviewerID uuid.UUID) (bool, error) {
	if m.hasReviewedFn != nil {
		return m.hasReviewedFn(ctx, proposalID, reviewerID)
	}
	return false, nil
}

func (m *mockReviewRepo) GetByProposalIDs(_ context.Context, _ []uuid.UUID, _ string) (map[uuid.UUID]*domain.Review, error) {
	return map[uuid.UUID]*domain.Review{}, nil
}

func (m *mockReviewRepo) ListAdmin(_ context.Context, _ repository.AdminReviewFilters) ([]repository.AdminReview, error) {
	return nil, nil
}

func (m *mockReviewRepo) CountAdmin(_ context.Context, _ repository.AdminReviewFilters) (int, error) {
	return 0, nil
}

func (m *mockReviewRepo) GetAdminByID(_ context.Context, _ uuid.UUID) (*repository.AdminReview, error) {
	return nil, domain.ErrNotFound
}

func (m *mockReviewRepo) DeleteAdmin(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (m *mockReviewRepo) UpdateReviewModeration(_ context.Context, _ uuid.UUID, _ string, _ float64, _ []byte) error {
	return nil
}

// mockProposalRepo implements the subset of ProposalRepository used by review service.
type mockProposalRepo struct {
	getByIDFn func(ctx context.Context, id uuid.UUID) (*mockProposal, error)
}

func (m *mockProposalRepo) Create(ctx context.Context, p *proposaldomain.Proposal) error {
	return nil
}

func (m *mockProposalRepo) CreateWithDocuments(ctx context.Context, p *proposaldomain.Proposal, docs []*proposaldomain.ProposalDocument) error {
	return nil
}

func (m *mockProposalRepo) CreateWithDocumentsAndMilestones(ctx context.Context, p *proposaldomain.Proposal, docs []*proposaldomain.ProposalDocument, _ []*milestonedomain.Milestone) error {
	return nil
}

func (m *mockProposalRepo) GetByID(ctx context.Context, id uuid.UUID) (*proposaldomain.Proposal, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, proposaldomain.ErrProposalNotFound
}

// GetByIDForOrg delegates to GetByID so the review service's
// migration to the org-aware variant transparently uses the
// existing test fixtures.
func (m *mockProposalRepo) GetByIDForOrg(ctx context.Context, id, _ uuid.UUID) (*proposaldomain.Proposal, error) {
	return m.GetByID(ctx, id)
}

func (m *mockProposalRepo) GetByIDs(context.Context, []uuid.UUID) ([]*proposaldomain.Proposal, error) {
	return nil, nil
}

func (m *mockProposalRepo) Update(ctx context.Context, p *proposaldomain.Proposal) error {
	return nil
}

func (m *mockProposalRepo) GetLatestVersion(ctx context.Context, rootID uuid.UUID) (*proposaldomain.Proposal, error) {
	return nil, nil
}

func (m *mockProposalRepo) ListByConversation(ctx context.Context, convID uuid.UUID) ([]*proposaldomain.Proposal, error) {
	return nil, nil
}

func (m *mockProposalRepo) ListActiveProjectsByOrganization(ctx context.Context, orgID uuid.UUID, cursor string, limit int) ([]*proposaldomain.Proposal, string, error) {
	return nil, "", nil
}

func (m *mockProposalRepo) ListCompletedByOrganization(_ context.Context, _ uuid.UUID, _ string, _ int) ([]*proposaldomain.Proposal, string, error) {
	return nil, "", nil
}

func (m *mockProposalRepo) GetDocuments(ctx context.Context, proposalID uuid.UUID) ([]*proposaldomain.ProposalDocument, error) {
	return nil, nil
}

func (m *mockProposalRepo) CreateDocument(ctx context.Context, doc *proposaldomain.ProposalDocument) error {
	return nil
}

// IsOrgAuthorizedForProposal is stubbed for review tests — review service
// fetches proposals directly via GetByID and does not rely on this gate
// (the review side validates via proposal status + reviewer id). Always
// returning true keeps the review unit tests focused on review logic.
func (m *mockProposalRepo) IsOrgAuthorizedForProposal(ctx context.Context, proposalID, orgID uuid.UUID) (bool, error) {
	return true, nil
}

// --- mockNotificationSender ---

type mockNotificationSender struct {
	sendFn func(ctx context.Context, input service.NotificationInput) error
	calls  []service.NotificationInput
}

func (m *mockNotificationSender) Send(ctx context.Context, input service.NotificationInput) error {
	m.calls = append(m.calls, input)
	if m.sendFn != nil {
		return m.sendFn(ctx, input)
	}
	return nil
}
func (m *mockProposalRepo) CountAll(_ context.Context) (int, int, error) { return 0, 0, nil }
func (m *mockProposalRepo) SumPaidByClientOrganization(context.Context, uuid.UUID) (int64, error) {
	return 0, nil
}
func (m *mockProposalRepo) ListCompletedByClientOrganization(context.Context, uuid.UUID, int) ([]*proposaldomain.Proposal, error) {
	return nil, nil
}

// --- mockUserRepo (minimal, org-aware) ---

type mockUserRepo struct{}

func (m *mockUserRepo) Create(context.Context, *userdomain.User) error { return nil }
func (m *mockUserRepo) GetByID(_ context.Context, id uuid.UUID) (*userdomain.User, error) {
	// Every user in review tests has a stub personal org so CreateReview
	// can resolve both parties' orgs without requiring explicit wiring
	// in every test case.
	stubOrg := uuid.New()
	return &userdomain.User{ID: id, OrganizationID: &stubOrg}, nil
}
func (m *mockUserRepo) GetByEmail(context.Context, string) (*userdomain.User, error) {
	return nil, userdomain.ErrUserNotFound
}
func (m *mockUserRepo) Update(context.Context, *userdomain.User) error          { return nil }
func (m *mockUserRepo) Delete(context.Context, uuid.UUID) error                 { return nil }
func (m *mockUserRepo) ExistsByEmail(context.Context, string) (bool, error)     { return false, nil }
func (m *mockUserRepo) ListAdmin(context.Context, repository.AdminUserFilters) ([]*userdomain.User, string, error) {
	return nil, "", nil
}
func (m *mockUserRepo) CountAdmin(context.Context, repository.AdminUserFilters) (int, error) {
	return 0, nil
}
func (m *mockUserRepo) CountByRole(context.Context) (map[string]int, error) { return nil, nil }
func (m *mockUserRepo) CountByStatus(context.Context) (map[string]int, error) {
	return nil, nil
}
func (m *mockUserRepo) RecentSignups(context.Context, int) ([]*userdomain.User, error) {
	return nil, nil
}
func (m *mockUserRepo) GetStripeAccount(context.Context, uuid.UUID) (string, string, error) {
	return "", "", nil
}
func (m *mockUserRepo) FindUserIDByStripeAccount(context.Context, string) (uuid.UUID, error) {
	return uuid.Nil, nil
}
func (m *mockUserRepo) SetStripeAccount(context.Context, uuid.UUID, string, string) error {
	return nil
}
func (m *mockUserRepo) ClearStripeAccount(context.Context, uuid.UUID) error           { return nil }
func (m *mockUserRepo) GetStripeLastState(context.Context, uuid.UUID) ([]byte, error) { return nil, nil }
func (m *mockUserRepo) SaveStripeLastState(context.Context, uuid.UUID, []byte) error  { return nil }
func (m *mockUserRepo) SetKYCFirstEarning(context.Context, uuid.UUID, time.Time) error {
	return nil
}
func (m *mockUserRepo) GetKYCPendingUsers(context.Context) ([]*userdomain.User, error) { return nil, nil }
func (m *mockUserRepo) SaveKYCNotificationState(context.Context, uuid.UUID, map[string]time.Time) error {
	return nil
}
func (m *mockUserRepo) BumpSessionVersion(context.Context, uuid.UUID) (int, error) {
	return 0, nil
}
func (m *mockUserRepo) GetSessionVersion(context.Context, uuid.UUID) (int, error) {
	return 0, nil
}
func (m *mockUserRepo) UpdateEmailNotificationsEnabled(context.Context, uuid.UUID, bool) error {
	return nil
}
func (m *mockUserRepo) TouchLastActive(context.Context, uuid.UUID) error {
	return nil
}
