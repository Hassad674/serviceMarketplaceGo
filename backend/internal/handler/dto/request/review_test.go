package request

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"marketplace-backend/pkg/validator"
)

func TestCreateReviewRequest_Validation(t *testing.T) {
	valid := CreateReviewRequest{
		ProposalID:   uuid.NewString(),
		GlobalRating: 5,
	}
	require.NoError(t, validator.Validate(valid))

	t.Run("rating out of range high", func(t *testing.T) {
		r := valid
		r.GlobalRating = 6
		require.Error(t, validator.Validate(r))
	})
	t.Run("rating out of range low", func(t *testing.T) {
		r := valid
		r.GlobalRating = 0
		require.Error(t, validator.Validate(r))
	})
	t.Run("invalid proposal id", func(t *testing.T) {
		r := valid
		r.ProposalID = "not-a-uuid"
		require.Error(t, validator.Validate(r))
	})
	t.Run("comment too long", func(t *testing.T) {
		r := valid
		r.Comment = strings.Repeat("a", 2001)
		require.Error(t, validator.Validate(r))
	})
}
