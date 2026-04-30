package request

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"marketplace-backend/pkg/validator"
)

func TestOpenDisputeRequest_Validation(t *testing.T) {
	valid := OpenDisputeRequest{
		ProposalID:  uuid.NewString(),
		Reason:      "Late delivery",
		Description: "Provider missed the deadline by 2 weeks.",
	}
	require.NoError(t, validator.Validate(valid))

	t.Run("invalid proposal id", func(t *testing.T) {
		r := valid
		r.ProposalID = "not-uuid"
		require.Error(t, validator.Validate(r))
	})

	t.Run("missing reason", func(t *testing.T) {
		r := valid
		r.Reason = ""
		require.Error(t, validator.Validate(r))
	})

	t.Run("description too long", func(t *testing.T) {
		r := valid
		r.Description = strings.Repeat("a", 5001)
		require.Error(t, validator.Validate(r))
	})

	t.Run("requested amount overflow", func(t *testing.T) {
		r := valid
		r.RequestedAmount = 1_000_000_000
		require.Error(t, validator.Validate(r))
	})
}

func TestAdminResolveDisputeRequest_Validation(t *testing.T) {
	require.Error(t, validator.Validate(AdminResolveDisputeRequest{
		AmountClient: -1, AmountProvider: 0,
	}))
}

func TestAskAIDisputeRequest_Validation(t *testing.T) {
	require.Error(t, validator.Validate(AskAIDisputeRequest{Question: ""}))
	require.NoError(t, validator.Validate(AskAIDisputeRequest{Question: "Why is this dispute frozen?"}))
}
