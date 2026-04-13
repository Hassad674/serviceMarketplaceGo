package review

import (
	"time"

	"github.com/google/uuid"
)

// Review represents a post-mission evaluation left by one party for the other.
//
// Since phase R3 extended, every review carries the reviewer's and the
// reviewed party's organization ids so the list + aggregate queries can
// filter by org without joining users.
//
// Since phase R18 (double-blind reviews), every review also carries:
//   - Side         — which direction the evaluation goes (client→provider or
//                    provider→client).
//   - PublishedAt  — NULL while the review is hidden (awaiting the counterpart
//                    or the 14-day deadline), set to the reveal moment once
//                    the review becomes visible on the public surface.
//
// Reviews are immutable once published: the reveal is the point of no return.
type Review struct {
	ID                     uuid.UUID
	ProposalID             uuid.UUID
	ReviewerID             uuid.UUID
	ReviewedID             uuid.UUID
	ReviewerOrganizationID uuid.UUID
	ReviewedOrganizationID uuid.UUID
	Side                   string
	GlobalRating           int
	Timeliness             *int
	Communication          *int
	Quality                *int
	Comment                string
	VideoURL               string
	TitleVisible           bool       // When false, the mission title must be hidden on the provider's public history.
	CreatedAt              time.Time
	UpdatedAt              time.Time
	PublishedAt            *time.Time // nil until the review is revealed.
}

// NewReviewInput groups parameters for creating a new Review.
type NewReviewInput struct {
	ProposalID             uuid.UUID
	ReviewerID             uuid.UUID
	ReviewedID             uuid.UUID
	ReviewerOrganizationID uuid.UUID
	ReviewedOrganizationID uuid.UUID
	Side                   string
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
	if !IsValidSide(in.Side) {
		return nil, ErrInvalidSide
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

	// Provider→client reviews cannot carry provider-specific sub-criteria:
	// those ratings only make sense when evaluating the delivery work.
	if in.Side == SideProviderToClient {
		if in.Timeliness != nil || in.Communication != nil || in.Quality != nil {
			return nil, ErrInvalidSubCriteriaForSide
		}
	}

	now := time.Now()
	return &Review{
		ID:                     uuid.New(),
		ProposalID:             in.ProposalID,
		ReviewerID:             in.ReviewerID,
		ReviewedID:             in.ReviewedID,
		ReviewerOrganizationID: in.ReviewerOrganizationID,
		ReviewedOrganizationID: in.ReviewedOrganizationID,
		Side:                   in.Side,
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

	// SideClientToProvider marks a review authored by the client against the
	// provider. These carry the delivery sub-criteria and feed the provider's
	// public rating.
	SideClientToProvider = "client_to_provider"

	// SideProviderToClient marks a review authored by the provider against
	// the client. These cannot carry delivery sub-criteria.
	SideProviderToClient = "provider_to_client"

	// ReviewWindowDays is the hard deadline (in days) after which a review
	// can no longer be submitted — and also the reveal deadline, past which
	// any pending review auto-publishes.
	ReviewWindowDays = 14
)

// ReviewWindow is the time.Duration form of ReviewWindowDays, computed once.
var ReviewWindow = time.Duration(ReviewWindowDays) * 24 * time.Hour

// IsValidSide reports whether s is one of the two known review sides.
func IsValidSide(s string) bool {
	return s == SideClientToProvider || s == SideProviderToClient
}

func validateRating(r int) error {
	if r < MinRating || r > MaxRating {
		return ErrInvalidRating
	}
	return nil
}
