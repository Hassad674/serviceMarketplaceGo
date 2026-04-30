package request

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"marketplace-backend/pkg/validator"
)

func TestInitiateCallRequest_Validation(t *testing.T) {
	valid := InitiateCallRequest{
		ConversationID: uuid.NewString(),
		RecipientID:    uuid.NewString(),
		Type:           "audio",
	}
	require.NoError(t, validator.Validate(valid))

	t.Run("invalid type", func(t *testing.T) {
		r := valid
		r.Type = "screen"
		require.Error(t, validator.Validate(r))
	})
	t.Run("invalid conversation id", func(t *testing.T) {
		r := valid
		r.ConversationID = "not-uuid"
		require.Error(t, validator.Validate(r))
	})
}

func TestEndCallRequest_Validation(t *testing.T) {
	require.Error(t, validator.Validate(EndCallRequest{Duration: -1}))
	require.Error(t, validator.Validate(EndCallRequest{Duration: 86401}))
	require.NoError(t, validator.Validate(EndCallRequest{Duration: 60}))
}
