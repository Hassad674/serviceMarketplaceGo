package review

import (
	"time"

	"github.com/google/uuid"
)

// Review represents a post-mission evaluation left by one party for the other.
// Since phase R3 extended, every review carries the reviewer's and the
// reviewed party's organization ids so the list + aggregate queries can
// filter by org without joining users.
type Review struct {
	ID                     uuid.UUID
	ProposalID             uuid.UUID
	ReviewerID             uuid.UUID
	ReviewedID             uuid.UUID
	ReviewerOrganizationID uuid.UUID
	ReviewedOrganizationID uuid.UUID
	GlobalRating           int
	Timeliness             *int
	Communication          *int
	Quality                *int
	Comment                string
	VideoURL               string
	TitleVisible           bool // When false, the mission title must be hidden on the provider's public history
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

// NewReviewInput groups parameters for creating a new Review.
type NewReviewInput struct {
	ProposalID             uuid.UUID
	ReviewerID             uuid.UUID
	ReviewedID             uuid.UUID
	ReviewerOrganizationID uuid.UUID
	ReviewedOrganizationID uuid.UUID
	GlobalRating           int
	Timeliness             *int
	Communication          *int
	Quality                *int
	Comment                string
	VideoURL               string
	TitleVisible           bool // Defaults to true when zero-valued; callers pass the explicit client choice.
}

// NewReview creates a validated Review from the given input.
func NewReview(in NewReviewInput) (*Review, error) {
	if in.ProposalID == uuid.Nil {
		return nil, ErrMissingProposal
	}
	if in.ReviewerID == uuid.Nil {
		return nil, ErrMissingReviewer
	}
	if in.ReviewedID == uuid.Nil {
		return nil, ErrMissingReviewed
	}
	if in.ReviewerID == in.ReviewedID {
		return nil, ErrSelfReview
	}
	if err := validateRating(in.GlobalRating); err != nil {
		return nil, err
	}
	if in.Timeliness != nil {
		if err := validateRating(*in.Timeliness); err != nil {
			return nil, err
		}
	}
	if in.Communication != nil {
		if err := validateRating(*in.Communication); err != nil {
			return nil, err
		}
	}
	if in.Quality != nil {
		if err := validateRating(*in.Quality); err != nil {
			return nil, err
		}
	}
	if len(in.Comment) > MaxCommentLength {
		return nil, ErrCommentTooLong
	}

	if in.ReviewerOrganizationID == uuid.Nil || in.ReviewedOrganizationID == uuid.Nil {
		return nil, ErrMissingReviewer
	}

	now := time.Now()
	return &Review{
		ID:                     uuid.New(),
		ProposalID:             in.ProposalID,
		ReviewerID:             in.ReviewerID,
		ReviewedID:             in.ReviewedID,
		ReviewerOrganizationID: in.ReviewerOrganizationID,
		ReviewedOrganizationID: in.ReviewedOrganizationID,
		GlobalRating:           in.GlobalRating,
		Timeliness:             in.Timeliness,
		Communication:          in.Communication,
		Quality:                in.Quality,
		Comment:                in.Comment,
		VideoURL:               in.VideoURL,
		TitleVisible:           in.TitleVisible,
		CreatedAt:              now,
		UpdatedAt:              now,
	}, nil
}

// AverageRating holds aggregated rating stats for a user.
type AverageRating struct {
	Average float64
	Count   int
}

const (
	MinRating        = 1
	MaxRating        = 5
	MaxCommentLength = 2000
)

func validateRating(r int) error {
	if r < MinRating || r > MaxRating {
		return ErrInvalidRating
	}
	return nil
}
