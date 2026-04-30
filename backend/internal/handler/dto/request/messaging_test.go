package request

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"marketplace-backend/pkg/validator"
)

func TestStartConversationRequest_Validation(t *testing.T) {
	valid := StartConversationRequest{
		RecipientOrgID: uuid.NewString(),
		Content:        "hello",
	}
	require.NoError(t, validator.Validate(valid))

	t.Run("invalid org id", func(t *testing.T) {
		r := valid
		r.RecipientOrgID = "not-uuid"
		require.Error(t, validator.Validate(r))
	})
	t.Run("empty content", func(t *testing.T) {
		r := valid
		r.Content = ""
		require.Error(t, validator.Validate(r))
	})
	t.Run("oversized content", func(t *testing.T) {
		r := valid
		r.Content = strings.Repeat("x", 10001)
		require.Error(t, validator.Validate(r))
	})
}

func TestSendMessageRequest_Validation(t *testing.T) {
	require.Error(t, validator.Validate(SendMessageRequest{Content: ""}))
	require.Error(t, validator.Validate(SendMessageRequest{Content: "hi", ReplyToID: "not-uuid"}))
	require.NoError(t, validator.Validate(SendMessageRequest{Content: "hi"}))
}

func TestEditMessageRequest_Validation(t *testing.T) {
	require.Error(t, validator.Validate(EditMessageRequest{Content: ""}))
}

func TestPresignedURLRequest_Validation(t *testing.T) {
	require.Error(t, validator.Validate(PresignedURLRequest{Filename: ""}))
}

func TestMarkAsReadRequest_Validation(t *testing.T) {
	require.Error(t, validator.Validate(MarkAsReadRequest{Seq: -1}))
}
