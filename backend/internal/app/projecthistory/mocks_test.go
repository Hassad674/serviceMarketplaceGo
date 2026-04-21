package projecthistory_test

import (
	"context"

	"github.com/google/uuid"

	milestonedomain "marketplace-backend/internal/domain/milestone"
	proposaldomain "marketplace-backend/internal/domain/proposal"
	reviewdomain "marketplace-backend/internal/domain/review"
	"marketplace-backend/internal/port/repository"
)

// --- mockProposalRepo ---

type mockProposalRepo struct {
	ListCompletedByOrganizationFunc    func(ctx context.Context, orgID uuid.UUID, cursor string, limit int) ([]*proposaldomain.Proposal, string, error)
	ListCompletedByClientOrganizationFn func(orgID uuid.UUID, limit int) ([]*proposaldomain.Proposal, error)
}

func (m *mockProposalRepo) Create(context.Context, *proposaldomain.Proposal) error { return nil }
func (m *mockProposalRepo) CreateWithDocuments(context.Context, *proposaldomain.Proposal, []*proposaldomain.ProposalDocument) error {
	return nil
}
func (m *mockProposalRepo) CreateWithDocumentsAndMilestones(context.Context, *proposaldomain.Proposal, []*proposaldomain.ProposalDocument, []*milestonedomain.Milestone) error {
	return nil
}
func (m *mockProposalRepo) GetByID(context.Context, uuid.UUID) (*proposaldomain.Proposal, error) {
	return nil, nil
}
func (m *mockProposalRepo) GetByIDs(context.Context, []uuid.UUID) ([]*proposaldomain.Proposal, error) {
	return nil, nil
}
func (m *mockProposalRepo) Update(context.Context, *proposaldomain.Proposal) error { return nil }
func (m *mockProposalRepo) GetLatestVersion(context.Context, uuid.UUID) (*proposaldomain.Proposal, error) {
	return nil, nil
}
func (m *mockProposalRepo) ListByConversation(context.Context, uuid.UUID) ([]*proposaldomain.Proposal, error) {
	return nil, nil
}
func (m *mockProposalRepo) ListActiveProjectsByOrganization(context.Context, uuid.UUID, string, int) ([]*proposaldomain.Proposal, string, error) {
	return nil, "", nil
}
func (m *mockProposalRepo) ListCompletedByOrganization(ctx context.Context, orgID uuid.UUID, cursor string, limit int) ([]*proposaldomain.Proposal, string, error) {
	if m.ListCompletedByOrganizationFunc != nil {
		return m.ListCompletedByOrganizationFunc(ctx, orgID, cursor, limit)
	}
	return nil, "", nil
}
func (m *mockProposalRepo) GetDocuments(context.Context, uuid.UUID) ([]*proposaldomain.ProposalDocument, error) {
	return nil, nil
}
func (m *mockProposalRepo) CreateDocument(context.Context, *proposaldomain.ProposalDocument) error {
	return nil
}
func (m *mockProposalRepo) IsOrgAuthorizedForProposal(context.Context, uuid.UUID, uuid.UUID) (bool, error) {
	return true, nil
}
func (m *mockProposalRepo) CountAll(context.Context) (int, int, error) { return 0, 0, nil }
func (m *mockProposalRepo) SumPaidByClientOrganization(context.Context, uuid.UUID) (int64, error) {
	return 0, nil
}
func (m *mockProposalRepo) ListCompletedByClientOrganization(_ context.Context, orgID uuid.UUID, limit int) ([]*proposaldomain.Proposal, error) {
	if m.ListCompletedByClientOrganizationFn != nil {
		return m.ListCompletedByClientOrganizationFn(orgID, limit)
	}
	return nil, nil
}

var _ repository.ProposalRepository = (*mockProposalRepo)(nil)

// --- mockReviewRepo ---

type mockReviewRepo struct {
	GetByProposalIDsFunc func(ctx context.Context, ids []uuid.UUID, side string) (map[uuid.UUID]*reviewdomain.Review, error)
}

func (m *mockReviewRepo) Create(context.Context, *reviewdomain.Review) error { return nil }
func (m *mockReviewRepo) CreateAndMaybeReveal(_ context.Context, r *reviewdomain.Review) (*reviewdomain.Review, error) {
	return r, nil
}
func (m *mockReviewRepo) GetByID(context.Context, uuid.UUID) (*reviewdomain.Review, error) {
	return nil, nil
}
func (m *mockReviewRepo) ListByReviewedOrganization(context.Context, uuid.UUID, string, int) ([]*reviewdomain.Review, string, error) {
	return nil, "", nil
}
func (m *mockReviewRepo) GetAverageRatingByOrganization(context.Context, uuid.UUID) (*reviewdomain.AverageRating, error) {
	return &reviewdomain.AverageRating{}, nil
}
func (m *mockReviewRepo) ListClientReviewsByOrganization(context.Context, uuid.UUID, int) ([]*reviewdomain.Review, error) {
	return nil, nil
}
func (m *mockReviewRepo) GetClientAverageRating(context.Context, uuid.UUID) (*reviewdomain.AverageRating, error) {
	return &reviewdomain.AverageRating{}, nil
}
func (m *mockReviewRepo) HasReviewed(context.Context, uuid.UUID, uuid.UUID) (bool, error) {
	return false, nil
}
func (m *mockReviewRepo) GetByProposalIDs(ctx context.Context, ids []uuid.UUID, side string) (map[uuid.UUID]*reviewdomain.Review, error) {
	if m.GetByProposalIDsFunc != nil {
		return m.GetByProposalIDsFunc(ctx, ids, side)
	}
	return map[uuid.UUID]*reviewdomain.Review{}, nil
}
func (m *mockReviewRepo) UpdateReviewModeration(context.Context, uuid.UUID, string, float64, []byte) error {
	return nil
}
func (m *mockReviewRepo) ListAdmin(context.Context, repository.AdminReviewFilters) ([]repository.AdminReview, error) {
	return nil, nil
}
func (m *mockReviewRepo) CountAdmin(context.Context, repository.AdminReviewFilters) (int, error) {
	return 0, nil
}
func (m *mockReviewRepo) GetAdminByID(context.Context, uuid.UUID) (*repository.AdminReview, error) {
	return nil, nil
}
func (m *mockReviewRepo) DeleteAdmin(context.Context, uuid.UUID) error { return nil }

var _ repository.ReviewRepository = (*mockReviewRepo)(nil)
