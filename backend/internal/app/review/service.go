package review

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	domain "marketplace-backend/internal/domain/review"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
)

// ServiceDeps groups the dependencies for the review service.
type ServiceDeps struct {
	Reviews       repository.ReviewRepository
	Proposals     repository.ProposalRepository
	Users         repository.UserRepository
	Notifications service.NotificationSender
}

// Service orchestrates review use cases.
type Service struct {
	reviews        repository.ReviewRepository
	proposals      repository.ProposalRepository
	users          repository.UserRepository
	notifications  service.NotificationSender
	textModeration service.TextModerationService
	adminNotifier  service.AdminNotifierService
}

// SetAdminNotifier sets the admin notifier after construction.
func (s *Service) SetAdminNotifier(n service.AdminNotifierService) {
	s.adminNotifier = n
}

// NewService creates a new review service.
func NewService(deps ServiceDeps) *Service {
	return &Service{
		reviews:       deps.Reviews,
		proposals:     deps.Proposals,
		users:         deps.Users,
		notifications: deps.Notifications,
	}
}

// SetTextModeration sets the text moderation service after construction.
func (s *Service) SetTextModeration(svc service.TextModerationService) {
	s.textModeration = svc
}

// CreateReviewInput contains the data needed to create a review.
type CreateReviewInput struct {
	ProposalID    uuid.UUID
	ReviewerID    uuid.UUID
	GlobalRating  int
	Timeliness    *int
	Communication *int
	Quality       *int
	Comment       string
	VideoURL      string
	TitleVisible  bool
}

// CreateReview validates the context and persists a new review.
func (s *Service) CreateReview(ctx context.Context, in CreateReviewInput) (*domain.Review, error) {
	// Verify proposal exists and is completed
	p, err := s.proposals.GetByID(ctx, in.ProposalID)
	if err != nil {
		return nil, fmt.Errorf("get proposal: %w", err)
	}
	if p.Status != "completed" {
		return nil, domain.ErrNotCompleted
	}

	// Only the client (the party who pays) can leave a review.
	// Enterprise evaluates Freelance/Agency, Agency evaluates Freelance.
	// The provider never evaluates the client.
	if in.ReviewerID != p.ClientID {
		return nil, domain.ErrNotParticipant
	}

	// The reviewed party is always the provider.
	reviewedID := p.ProviderID

	// Check for duplicate review
	already, err := s.reviews.HasReviewed(ctx, in.ProposalID, in.ReviewerID)
	if err != nil {
		return nil, fmt.Errorf("check existing review: %w", err)
	}
	if already {
		return nil, domain.ErrAlreadyReviewed
	}

	// Resolve both parties' current organizations so the review is
	// visible to every operator of the reviewed org + tagged by the
	// reviewer's org (Stripe Dashboard shared workspace).
	reviewerUser, err := s.users.GetByID(ctx, in.ReviewerID)
	if err != nil {
		return nil, fmt.Errorf("lookup reviewer user: %w", err)
	}
	reviewedUser, err := s.users.GetByID(ctx, reviewedID)
	if err != nil {
		return nil, fmt.Errorf("lookup reviewed user: %w", err)
	}
	if reviewerUser.OrganizationID == nil || reviewedUser.OrganizationID == nil {
		return nil, fmt.Errorf("create review: participants must belong to an organization")
	}

	// Create domain entity
	r, err := domain.NewReview(domain.NewReviewInput{
		ProposalID:             in.ProposalID,
		ReviewerID:             in.ReviewerID,
		ReviewedID:             reviewedID,
		ReviewerOrganizationID: *reviewerUser.OrganizationID,
		ReviewedOrganizationID: *reviewedUser.OrganizationID,
		GlobalRating:           in.GlobalRating,
		Timeliness:             in.Timeliness,
		Communication:          in.Communication,
		Quality:                in.Quality,
		Comment:                in.Comment,
		VideoURL:               in.VideoURL,
		TitleVisible:           in.TitleVisible,
	})
	if err != nil {
		return nil, err
	}

	if err := s.reviews.Create(ctx, r); err != nil {
		return nil, fmt.Errorf("persist review: %w", err)
	}

	if s.notifications != nil {
		notifData, _ := json.Marshal(map[string]any{
			"review_id":   r.ID.String(),
			"proposal_id": r.ProposalID.String(),
			"rating":      r.GlobalRating,
		})
		_ = s.notifications.Send(ctx, service.NotificationInput{
			UserID: r.ReviewedID,
			Type:   "review_received",
			Title:  "New review received",
			Body:   fmt.Sprintf("You received a %d-star review", r.GlobalRating),
			Data:   notifData,
		})
	}

	s.moderateReviewIfNeeded(r)

	return r, nil
}

// moderateReviewIfNeeded fires a background text moderation check for the review comment.
func (s *Service) moderateReviewIfNeeded(r *domain.Review) {
	if s.textModeration == nil || r.Comment == "" {
		return
	}

	go s.runReviewModeration(r.ID, r.Comment)
}

// runReviewModeration calls the text moderation service and updates the review.
func (s *Service) runReviewModeration(reviewID uuid.UUID, comment string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := s.textModeration.AnalyzeText(ctx, comment)
	if err != nil {
		slog.Error("review text moderation failed", "error", err, "review_id", reviewID)
		return
	}

	if result.IsSafe {
		return
	}

	status := "flagged"
	if result.MaxScore >= 0.9 {
		status = "hidden"
	}

	labelsJSON, err := json.Marshal(result.Labels)
	if err != nil {
		slog.Error("marshal review moderation labels", "error", err, "review_id", reviewID)
		return
	}

	if err := s.reviews.UpdateReviewModeration(ctx, reviewID, status, result.MaxScore, labelsJSON); err != nil {
		slog.Error("update review moderation", "error", err, "review_id", reviewID)
	}

	// Notify admins of flagged review
	if s.adminNotifier != nil {
		if err := s.adminNotifier.IncrementAll(ctx, service.AdminNotifReviewsFlagged); err != nil {
			slog.Error("admin notifier: increment reviews_flagged", "error", err)
		}
	}
}

// ListByOrganization returns reviews received by an organization (public).
func (s *Service) ListByOrganization(ctx context.Context, orgID uuid.UUID, cursor string, limit int) ([]*domain.Review, string, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	return s.reviews.ListByReviewedOrganization(ctx, orgID, cursor, limit)
}

// GetAverageRatingByOrganization returns the average rating for an org.
func (s *Service) GetAverageRatingByOrganization(ctx context.Context, orgID uuid.UUID) (*domain.AverageRating, error) {
	return s.reviews.GetAverageRatingByOrganization(ctx, orgID)
}

// CanReview checks if the current user can review a given proposal.
// Only the client (the paying party) is allowed to leave a review.
func (s *Service) CanReview(ctx context.Context, proposalID, userID uuid.UUID) (bool, error) {
	p, err := s.proposals.GetByID(ctx, proposalID)
	if err != nil {
		return false, fmt.Errorf("get proposal: %w", err)
	}
	if p.Status != "completed" {
		return false, nil
	}
	// Only the client can review; the provider never evaluates.
	if userID != p.ClientID {
		return false, nil
	}
	already, err := s.reviews.HasReviewed(ctx, proposalID, userID)
	if err != nil {
		return false, fmt.Errorf("check existing review: %w", err)
	}
	return !already, nil
}
