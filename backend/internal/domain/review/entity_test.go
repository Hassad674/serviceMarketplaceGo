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
		ProposalID:             uuid.New(),
		ReviewerID:             uuid.New(),
		ReviewedID:             uuid.New(),
		ReviewerOrganizationID: uuid.New(),
		ReviewedOrganizationID: uuid.New(),
		Side:                   SideClientToProvider,
		GlobalRating:           5,
		Timeliness:             &timeliness,
		Communication:          &communication,
		Quality:                &quality,
		Comment:                "Great work!",
	})

	assert.NoError(t, err)
	assert.NotNil(t, r)
	assert.Equal(t, 5, r.GlobalRating)
	assert.Equal(t, &timeliness, r.Timeliness)
	assert.Equal(t, "Great work!", r.Comment)
	assert.Equal(t, SideClientToProvider, r.Side)
	assert.Nil(t, r.PublishedAt, "new reviews must start hidden (published_at nil)")
}

func TestNewReview_MinimalValid(t *testing.T) {
	r, err := NewReview(NewReviewInput{
		ProposalID:             uuid.New(),
		ReviewerID:             uuid.New(),
		ReviewedID:             uuid.New(),
		ReviewerOrganizationID: uuid.New(),
		ReviewedOrganizationID: uuid.New(),
		Side:                   SideClientToProvider,
		GlobalRating:           1,
	})

	assert.NoError(t, err)
	assert.NotNil(t, r)
	assert.Nil(t, r.Timeliness)
	assert.Nil(t, r.Communication)
	assert.Nil(t, r.Quality)
	assert.Empty(t, r.Comment)
}

func TestNewReview_ProviderToClient_NoSubCriteria(t *testing.T) {
	r, err := NewReview(NewReviewInput{
		ProposalID:             uuid.New(),
		ReviewerID:             uuid.New(),
		ReviewedID:             uuid.New(),
		ReviewerOrganizationID: uuid.New(),
		ReviewedOrganizationID: uuid.New(),
		Side:                   SideProviderToClient,
		GlobalRating:           4,
		Comment:                "Clear brief, paid on time.",
	})

	assert.NoError(t, err)
	assert.NotNil(t, r)
	assert.Equal(t, SideProviderToClient, r.Side)
	assert.Nil(t, r.Timeliness)
	assert.Nil(t, r.Communication)
	assert.Nil(t, r.Quality)
}

func TestNewReview_Validation(t *testing.T) {
	validProposalID := uuid.New()
	validReviewerID := uuid.New()
	validReviewedID := uuid.New()
	validReviewerOrg := uuid.New()
	validReviewedOrg := uuid.New()

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
				Side:         SideClientToProvider,
				GlobalRating: 5,
			},
			wantErr: ErrMissingProposal,
		},
		{
			name: "missing reviewer",
			input: NewReviewInput{
				ProposalID:   validProposalID,
				ReviewedID:   validReviewedID,
				Side:         SideClientToProvider,
				GlobalRating: 5,
			},
			wantErr: ErrMissingReviewer,
		},
		{
			name: "missing reviewed",
			input: NewReviewInput{
				ProposalID:   validProposalID,
				ReviewerID:   validReviewerID,
				Side:         SideClientToProvider,
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
				Side:         SideClientToProvider,
				GlobalRating: 5,
			},
			wantErr: ErrSelfReview,
		},
		{
			name: "invalid side — empty",
			input: NewReviewInput{
				ProposalID:             validProposalID,
				ReviewerID:             validReviewerID,
				ReviewedID:             validReviewedID,
				ReviewerOrganizationID: validReviewerOrg,
				ReviewedOrganizationID: validReviewedOrg,
				GlobalRating:           5,
			},
			wantErr: ErrInvalidSide,
		},
		{
			name: "invalid side — unknown value",
			input: NewReviewInput{
				ProposalID:             validProposalID,
				ReviewerID:             validReviewerID,
				ReviewedID:             validReviewedID,
				ReviewerOrganizationID: validReviewerOrg,
				ReviewedOrganizationID: validReviewedOrg,
				Side:                   "mutual",
				GlobalRating:           5,
			},
			wantErr: ErrInvalidSide,
		},
		{
			name: "rating too low",
			input: NewReviewInput{
				ProposalID:   validProposalID,
				ReviewerID:   validReviewerID,
				ReviewedID:   validReviewedID,
				Side:         SideClientToProvider,
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
				Side:         SideClientToProvider,
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
				Side:         SideClientToProvider,
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
				Side:          SideClientToProvider,
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
				Side:         SideClientToProvider,
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
				Side:         SideClientToProvider,
				GlobalRating: 4,
				Comment:      strings.Repeat("a", MaxCommentLength+1),
			},
			wantErr: ErrCommentTooLong,
		},
		{
			name: "provider-to-client with timeliness",
			input: NewReviewInput{
				ProposalID:             validProposalID,
				ReviewerID:             validReviewerID,
				ReviewedID:             validReviewedID,
				ReviewerOrganizationID: validReviewerOrg,
				ReviewedOrganizationID: validReviewedOrg,
				Side:                   SideProviderToClient,
				GlobalRating:           4,
				Timeliness:             intPtr(3),
			},
			wantErr: ErrInvalidSubCriteriaForSide,
		},
		{
			name: "provider-to-client with communication",
			input: NewReviewInput{
				ProposalID:             validProposalID,
				ReviewerID:             validReviewerID,
				ReviewedID:             validReviewedID,
				ReviewerOrganizationID: validReviewerOrg,
				ReviewedOrganizationID: validReviewedOrg,
				Side:                   SideProviderToClient,
				GlobalRating:           4,
				Communication:          intPtr(3),
			},
			wantErr: ErrInvalidSubCriteriaForSide,
		},
		{
			name: "provider-to-client with quality",
			input: NewReviewInput{
				ProposalID:             validProposalID,
				ReviewerID:             validReviewerID,
				ReviewedID:             validReviewedID,
				ReviewerOrganizationID: validReviewerOrg,
				ReviewedOrganizationID: validReviewedOrg,
				Side:                   SideProviderToClient,
				GlobalRating:           4,
				Quality:                intPtr(3),
			},
			wantErr: ErrInvalidSubCriteriaForSide,
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

func TestIsValidSide(t *testing.T) {
	tests := []struct {
		name string
		side string
		want bool
	}{
		{"client_to_provider is valid", SideClientToProvider, true},
		{"provider_to_client is valid", SideProviderToClient, true},
		{"empty is invalid", "", false},
		{"unknown is invalid", "foo", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsValidSide(tt.side))
		})
	}
}

func intPtr(v int) *int { return &v }
