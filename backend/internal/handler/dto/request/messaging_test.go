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

// ResolvedContentType returns content_type when present, otherwise
// falls back to mime_type. Both fields exist for backward
// compatibility with older mobile builds; the helper centralises the
// fallback so handlers don't repeat the conditional everywhere.
func TestPresignedURLRequest_ResolvedContentType(t *testing.T) {
	cases := []struct {
		name     string
		req      PresignedURLRequest
		expected string
	}{
		{
			name: "content_type wins when set",
			req: PresignedURLRequest{
				Filename:    "doc.pdf",
				ContentType: "application/pdf",
				MimeType:    "image/jpeg",
			},
			expected: "application/pdf",
		},
		{
			name: "falls back to mime_type when content_type is empty",
			req: PresignedURLRequest{
				Filename: "img.jpg",
				MimeType: "image/jpeg",
			},
			expected: "image/jpeg",
		},
		{
			name: "empty when both are missing",
			req: PresignedURLRequest{
				Filename: "f.bin",
			},
			expected: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expected, tc.req.ResolvedContentType())
		})
	}
}
