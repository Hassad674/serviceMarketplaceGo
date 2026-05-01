package review

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	appmoderation "marketplace-backend/internal/app/moderation"
	"marketplace-backend/internal/domain/moderation"
	domain "marketplace-backend/internal/domain/review"
	"marketplace-backend/internal/handler/middleware"
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
	reviews                repository.ReviewRepository
	proposals              repository.ProposalRepository
	users                  repository.UserRepository
	notifications          service.NotificationSender
	moderationOrchestrator *appmoderation.Service
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

// SetModerationOrchestrator wires the central moderation pipeline.
// Optional: when nil, automated review-comment moderation is disabled.
// Replaces the legacy SetTextModeration + SetAdminNotifier +
// SetAuditRepo trio — the orchestrator handles those collaborators
// internally.
func (s *Service) SetModerationOrchestrator(svc *appmoderation.Service) {
	s.moderationOrchestrator = svc
}

// CreateReviewInput contains the data needed to create a review. Note
// that the review side is NOT in this input: it is always derived
// server-side from the reviewer id vs. the proposal's participants.
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

// CreateReview validates the context and persists a new review. It
// implements the double-blind reveal protocol:
//
//  1. Verify the proposal exists and is completed.
//  2. Derive the review side from the reviewer's position (client or
//     provider) on the proposal. A third party → ErrNotParticipant.
//  3. Enforce the 14-day review window. Past the deadline → the review
//     is rejected outright (ErrReviewWindowClosed).
//  4. Reject duplicate submissions from the same reviewer.
//  5. Persist the review via CreateAndMaybeReveal, which atomically
//     inserts the row and flips pending reviews on the proposal to
//     published_at = NOW() whenever the pair is complete (or when a
//     backfilled client review is already visible).
//  6. Fire notifications to the counterpart (and, when a reveal
//     happened, to the reviewer themselves).
func (s *Service) CreateReview(ctx context.Context, in CreateReviewInput) (*domain.Review, error) {
	// CreateReview is always called from a user-authenticated
	// route; the org context is populated by the auth middleware.
	// MustGetOrgID surfaces a missing context as a panic so a
	// programming bug in a future caller (handler forgot to
	// enforce auth, or a test forgot to set the context) is loud
	// instead of silently degrading to a cross-tenant read.
	orgID := middleware.MustGetOrgID(ctx)
	p, err := s.proposals.GetByIDForOrg(ctx, in.ProposalID, orgID)
	if err != nil {
		return nil, fmt.Errorf("get proposal: %w", err)
	}
	if p.Status != "completed" {
		return nil, domain.ErrNotCompleted
	}

	side, reviewedID, err := deriveReviewSide(in.ReviewerID, p.ClientID, p.ProviderID)
	if err != nil {
		return nil, err
	}

	if err := enforceReviewWindow(p.CompletedAt); err != nil {
		return nil, err
	}

	already, err := s.reviews.HasReviewed(ctx, in.ProposalID, in.ReviewerID)
	if err != nil {
		return nil, fmt.Errorf("check existing review: %w", err)
	}
	if already {
		return nil, domain.ErrAlreadyReviewed
	}

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

	r, err := domain.NewReview(domain.NewReviewInput{
		ProposalID:             in.ProposalID,
		ReviewerID:             in.ReviewerID,
		ReviewedID:             reviewedID,
		ReviewerOrganizationID: *reviewerUser.OrganizationID,
		ReviewedOrganizationID: *reviewedUser.OrganizationID,
		Side:                   side,
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

	persisted, err := s.reviews.CreateAndMaybeReveal(ctx, r)
	if err != nil {
		return nil, fmt.Errorf("persist review: %w", err)
	}

	s.sendReviewNotifications(ctx, persisted, p.Title, reviewerUser.DisplayName)

	s.moderateReviewIfNeeded(persisted)

	return persisted, nil
}

// deriveReviewSide returns the review side implied by the reviewer's
// position on the proposal. A third party → ErrNotParticipant.
func deriveReviewSide(reviewerID, clientID, providerID uuid.UUID) (side string, reviewedID uuid.UUID, err error) {
	switch reviewerID {
	case clientID:
		return domain.SideClientToProvider, providerID, nil
	case providerID:
		return domain.SideProviderToClient, clientID, nil
	default:
		return "", uuid.Nil, domain.ErrNotParticipant
	}
}

// enforceReviewWindow returns ErrReviewWindowClosed when the proposal
// was completed more than ReviewWindowDays ago. Missing completion
// timestamps are treated as closed — the status check above should
// already have caught non-completed proposals.
func enforceReviewWindow(completedAt *time.Time) error {
	if completedAt == nil {
		return domain.ErrReviewWindowClosed
	}
	if time.Since(*completedAt) > domain.ReviewWindow {
		return domain.ErrReviewWindowClosed
	}
	return nil
}

// sendReviewNotifications fires the user-facing notifications that
// accompany a review submission. It uses the post-transaction value of
// PublishedAt to decide whether the reveal message should be sent.
func (s *Service) sendReviewNotifications(ctx context.Context, r *domain.Review, proposalTitle, reviewerDisplayName string) {
	if s.notifications == nil {
		return
	}

	revealed := r.PublishedAt != nil
	reviewerLabel := reviewerDisplayName
	if reviewerLabel == "" {
		reviewerLabel = "Someone"
	}

	notifData, _ := json.Marshal(map[string]any{
		"review_id":      r.ID.String(),
		"proposal_id":    r.ProposalID.String(),
		"proposal_title": proposalTitle,
		"side":           r.Side,
		"rating":         r.GlobalRating,
		"revealed":       revealed,
	})

	var counterpartTitle, counterpartBody string
	if revealed {
		counterpartTitle = "New review received"
		counterpartBody = fmt.Sprintf("%s left you a review.", reviewerLabel)
	} else {
		counterpartTitle = "You were reviewed"
		counterpartBody = fmt.Sprintf("%s reviewed you. Submit your review within %d days to see it.", reviewerLabel, domain.ReviewWindowDays)
	}

	_ = s.notifications.Send(ctx, service.NotificationInput{
		UserID: r.ReviewedID,
		Type:   "review_received",
		Title:  counterpartTitle,
		Body:   counterpartBody,
		Data:   notifData,
	})

	// When the reveal happened on this submission, also ping the
	// reviewer so they know the counterpart's review is now visible.
	if revealed {
		_ = s.notifications.Send(ctx, service.NotificationInput{
			UserID: r.ReviewerID,
			Type:   "review_revealed",
			Title:  "Reviews unlocked",
			Body:   fmt.Sprintf("Reviews are now visible for mission %q.", proposalTitle),
			Data:   notifData,
		})
	}
}

// moderateReviewIfNeeded fires a background moderation check for the
// review comment via the central orchestrator. Returns immediately —
// the analysis runs in a goroutine so review submission stays snappy.
func (s *Service) moderateReviewIfNeeded(r *domain.Review) {
	if s.moderationOrchestrator == nil || r.Comment == "" {
		return
	}

	authorID := r.ReviewerID
	go s.runReviewModeration(r.ID, &authorID, r.Comment)
}

// runReviewModeration is the goroutine entry point. The 30s timeout
// matches the messaging pipeline so both surfaces share an SLA.
func (s *Service) runReviewModeration(reviewID uuid.UUID, authorID *uuid.UUID, comment string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := s.moderationOrchestrator.Moderate(ctx, appmoderation.ModerateInput{
		ContentType:  moderation.ContentTypeReview,
		ContentID:    reviewID,
		AuthorUserID: authorID,
		Text:         comment,
	})
	if err != nil {
		slog.Error("review text moderation failed", "error", err, "review_id", reviewID)
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
// Both parties (client and provider) are now eligible, subject to the
// 14-day window and duplicate-review rules.
func (s *Service) CanReview(ctx context.Context, proposalID, userID uuid.UUID) (bool, error) {
	// Same boundary contract as CreateReview — every CanReview
	// call originates from /api/v1/reviews/can-review/{proposalId}
	// which is gated behind Auth + RequireRole. Reach this code
	// path without the org context = bug.
	orgID := middleware.MustGetOrgID(ctx)
	p, err := s.proposals.GetByIDForOrg(ctx, proposalID, orgID)
	if err != nil {
		return false, fmt.Errorf("get proposal: %w", err)
	}
	if p.Status != "completed" {
		return false, nil
	}
	if userID != p.ClientID && userID != p.ProviderID {
		return false, nil
	}
	if err := enforceReviewWindow(p.CompletedAt); err != nil {
		return false, nil
	}
	already, err := s.reviews.HasReviewed(ctx, proposalID, userID)
	if err != nil {
		return false, fmt.Errorf("check existing review: %w", err)
	}
	return !already, nil
}
