package review

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNewReview_Valid(t *testing.T) {
	timeliness := 4
	communication := 5
	quality := 3

	r, err := NewReview(NewReviewInput{
		ProposalID:    uuid.New(),
		ReviewerID:    uuid.New(),
		ReviewedID:    uuid.New(),
		GlobalRating:  5,
		Timeliness:    &timeliness,
		Communication: &communication,
		Quality:       &quality,
		Comment:       "Great work!",
	})

	assert.NoError(t, err)
	assert.NotNil(t, r)
	assert.Equal(t, 5, r.GlobalRating)
	assert.Equal(t, &timeliness, r.Timeliness)
	assert.Equal(t, "Great work!", r.Comment)
}

func TestNewReview_MinimalValid(t *testing.T) {
	r, err := NewReview(NewReviewInput{
		ProposalID:   uuid.New(),
		ReviewerID:   uuid.New(),
		ReviewedID:   uuid.New(),
		GlobalRating: 1,
	})

	assert.NoError(t, err)
	assert.NotNil(t, r)
	assert.Nil(t, r.Timeliness)
	assert.Nil(t, r.Communication)
	assert.Nil(t, r.Quality)
	assert.Empty(t, r.Comment)
}

func TestNewReview_Validation(t *testing.T) {
	validProposalID := uuid.New()
	validReviewerID := uuid.New()
	validReviewedID := uuid.New()

	tests := []struct {
		name    string
		input   NewReviewInput
		wantErr error
	}{
		{
			name: "missing proposal",
			input: NewReviewInput{
				ReviewerID:   validReviewerID,
				ReviewedID:   validReviewedID,
				GlobalRating: 5,
			},
			wantErr: ErrMissingProposal,
		},
		{
			name: "missing reviewer",
			input: NewReviewInput{
				ProposalID:   validProposalID,
				ReviewedID:   validReviewedID,
				GlobalRating: 5,
			},
			wantErr: ErrMissingReviewer,
		},
		{
			name: "missing reviewed",
			input: NewReviewInput{
				ProposalID:   validProposalID,
				ReviewerID:   validReviewerID,
				GlobalRating: 5,
			},
			wantErr: ErrMissingReviewed,
		},
		{
			name: "self review",
			input: NewReviewInput{
				ProposalID:   validProposalID,
				ReviewerID:   validReviewerID,
				ReviewedID:   validReviewerID,
				GlobalRating: 5,
			},
			wantErr: ErrSelfReview,
		},
		{
			name: "rating too low",
			input: NewReviewInput{
				ProposalID:   validProposalID,
				ReviewerID:   validReviewerID,
				ReviewedID:   validReviewedID,
				GlobalRating: 0,
			},
			wantErr: ErrInvalidRating,
		},
		{
			name: "rating too high",
			input: NewReviewInput{
				ProposalID:   validProposalID,
				ReviewerID:   validReviewerID,
				ReviewedID:   validReviewedID,
				GlobalRating: 6,
			},
			wantErr: ErrInvalidRating,
		},
		{
			name: "invalid timeliness rating",
			input: NewReviewInput{
				ProposalID:   validProposalID,
				ReviewerID:   validReviewerID,
				ReviewedID:   validReviewedID,
				GlobalRating: 4,
				Timeliness:   intPtr(0),
			},
			wantErr: ErrInvalidRating,
		},
		{
			name: "invalid communication rating",
			input: NewReviewInput{
				ProposalID:    validProposalID,
				ReviewerID:    validReviewerID,
				ReviewedID:    validReviewedID,
				GlobalRating:  4,
				Communication: intPtr(7),
			},
			wantErr: ErrInvalidRating,
		},
		{
			name: "invalid quality rating",
			input: NewReviewInput{
				ProposalID:   validProposalID,
				ReviewerID:   validReviewerID,
				ReviewedID:   validReviewedID,
				GlobalRating: 4,
				Quality:      intPtr(-1),
			},
			wantErr: ErrInvalidRating,
		},
		{
			name: "comment too long",
			input: NewReviewInput{
				ProposalID:   validProposalID,
				ReviewerID:   validReviewerID,
				ReviewedID:   validReviewedID,
				GlobalRating: 4,
				Comment:      strings.Repeat("a", MaxCommentLength+1),
			},
			wantErr: ErrCommentTooLong,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := NewReview(tt.input)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.Nil(t, r)
		})
	}
}

func intPtr(v int) *int { return &v }
