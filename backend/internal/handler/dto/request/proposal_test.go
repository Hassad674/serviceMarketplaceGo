package request

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/pkg/validator"
)

func TestCreateProposalRequest_Validation(t *testing.T) {
	validReq := func() CreateProposalRequest {
		return CreateProposalRequest{
			RecipientID:    uuid.NewString(),
			ConversationID: uuid.NewString(),
			Title:          "A title",
			Description:    "A meaningful description",
			Amount:         10000,
		}
	}

	t.Run("happy path", func(t *testing.T) {
		assert.NoError(t, validator.Validate(validReq()))
	})

	t.Run("invalid recipient_id", func(t *testing.T) {
		r := validReq()
		r.RecipientID = "not-a-uuid"
		err := validator.Validate(r)
		require.Error(t, err)
		ve, _ := validator.IsValidationError(err)
		assert.Contains(t, fieldNames(ve), "recipientid")
	})

	t.Run("title too long", func(t *testing.T) {
		r := validReq()
		r.Title = strings.Repeat("a", 201)
		err := validator.Validate(r)
		require.Error(t, err)
	})

	t.Run("description too long", func(t *testing.T) {
		r := validReq()
		r.Description = strings.Repeat("a", 5001)
		err := validator.Validate(r)
		require.Error(t, err)
	})

	t.Run("amount negative", func(t *testing.T) {
		r := validReq()
		r.Amount = -1
		err := validator.Validate(r)
		require.Error(t, err)
	})

	t.Run("amount above cap (Stripe overflow guard)", func(t *testing.T) {
		r := validReq()
		r.Amount = 1_000_000_000
		err := validator.Validate(r)
		require.Error(t, err)
	})

	t.Run("too many milestones", func(t *testing.T) {
		r := validReq()
		ms := make([]MilestoneInputRequest, 21)
		for i := range ms {
			ms[i] = MilestoneInputRequest{Sequence: i + 1, Title: "m", Amount: 100}
		}
		r.Milestones = ms
		err := validator.Validate(r)
		require.Error(t, err)
	})

	t.Run("invalid payment_mode", func(t *testing.T) {
		r := validReq()
		r.PaymentMode = "weekly"
		err := validator.Validate(r)
		require.Error(t, err)
	})

	t.Run("document with bad URL", func(t *testing.T) {
		r := validReq()
		r.Documents = []DocumentInput{{Filename: "f", URL: "not a url", Size: 1, MimeType: "x"}}
		err := validator.Validate(r)
		require.Error(t, err)
	})
}

func TestModifyProposalRequest_Validation(t *testing.T) {
	err := validator.Validate(ModifyProposalRequest{Title: "", Description: "", Amount: -1})
	require.Error(t, err)
}
