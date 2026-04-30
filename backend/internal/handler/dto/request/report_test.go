package request

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"marketplace-backend/pkg/validator"
)

func TestCreateReportRequest_Validation(t *testing.T) {
	valid := CreateReportRequest{
		TargetType: "user",
		TargetID:   uuid.NewString(),
		Reason:     "Spam",
	}
	require.NoError(t, validator.Validate(valid))

	t.Run("invalid target id", func(t *testing.T) {
		r := valid
		r.TargetID = "not-uuid"
		require.Error(t, validator.Validate(r))
	})
	t.Run("missing reason", func(t *testing.T) {
		r := valid
		r.Reason = ""
		require.Error(t, validator.Validate(r))
	})
	t.Run("invalid conversation id", func(t *testing.T) {
		r := valid
		r.ConversationID = "not-uuid"
		require.Error(t, validator.Validate(r))
	})
}
