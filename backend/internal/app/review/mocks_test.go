package review

import (
	"context"

	"github.com/google/uuid"

	proposaldomain "marketplace-backend/internal/domain/proposal"
	domain "marketplace-backend/internal/domain/review"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
)

// mockProposal is a test helper to build proposals with minimal fields.
type mockProposal = proposaldomain.Proposal

// mockReviewRepo implements repository.ReviewRepository for tests.
type mockReviewRepo struct {
	createFn        func(ctx context.Context, r *domain.Review) error
	getByIDFn       func(ctx context.Context, id uuid.UUID) (*domain.Review, error)
	listByUserFn    func(ctx context.Context, userID uuid.UUID, cursor string, limit int) ([]*domain.Review, string, error)
	getAverageFn    func(ctx context.Context, userID uuid.UUID) (*domain.AverageRating, error)
	hasReviewedFn   func(ctx context.Context, proposalID, reviewerID uuid.UUID) (bool, error)
}

func (m *mockReviewRepo) Create(ctx context.Context, r *domain.Review) error {
	if m.createFn != nil {
		return m.createFn(ctx, r)
	}
	return nil
}

func (m *mockReviewRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Review, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, domain.ErrNotFound
}

func (m *mockReviewRepo) ListByReviewedUser(ctx context.Context, userID uuid.UUID, cursor string, limit int) ([]*domain.Review, string, error) {
	if m.listByUserFn != nil {
		return m.listByUserFn(ctx, userID, cursor, limit)
	}
	return nil, "", nil
}

func (m *mockReviewRepo) GetAverageRating(ctx context.Context, userID uuid.UUID) (*domain.AverageRating, error) {
	if m.getAverageFn != nil {
		return m.getAverageFn(ctx, userID)
	}
	return &domain.AverageRating{}, nil
}

func (m *mockReviewRepo) HasReviewed(ctx context.Context, proposalID, reviewerID uuid.UUID) (bool, error) {
	if m.hasReviewedFn != nil {
		return m.hasReviewedFn(ctx, proposalID, reviewerID)
	}
	return false, nil
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

func (m *mockProposalRepo) GetByID(ctx context.Context, id uuid.UUID) (*proposaldomain.Proposal, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, proposaldomain.ErrProposalNotFound
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

func (m *mockProposalRepo) ListActiveProjects(ctx context.Context, userID uuid.UUID, cursor string, limit int) ([]*proposaldomain.Proposal, string, error) {
	return nil, "", nil
}

func (m *mockProposalRepo) GetDocuments(ctx context.Context, proposalID uuid.UUID) ([]*proposaldomain.ProposalDocument, error) {
	return nil, nil
}

func (m *mockProposalRepo) CreateDocument(ctx context.Context, doc *proposaldomain.ProposalDocument) error {
	return nil
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
