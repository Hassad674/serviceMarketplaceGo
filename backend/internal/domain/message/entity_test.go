package message

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMessage_ValidTextMessage(t *testing.T) {
	convID := uuid.New()
	senderID := uuid.New()

	msg, err := NewMessage(convID, senderID, "Hello world", MessageTypeText, nil, 1)

	require.NoError(t, err)
	require.NotNil(t, msg)
	assert.Equal(t, convID, msg.ConversationID)
	assert.Equal(t, senderID, msg.SenderID)
	assert.Equal(t, "Hello world", msg.Content)
	assert.Equal(t, MessageTypeText, msg.Type)
	assert.Equal(t, 1, msg.Seq)
	assert.Equal(t, MessageStatusSent, msg.Status)
	assert.NotEqual(t, uuid.Nil, msg.ID)
	assert.False(t, msg.CreatedAt.IsZero())
	assert.False(t, msg.UpdatedAt.IsZero())
	assert.Nil(t, msg.EditedAt)
	assert.Nil(t, msg.DeletedAt)
}

func TestNewMessage_ValidFileMessage(t *testing.T) {
	convID := uuid.New()
	senderID := uuid.New()
	meta := json.RawMessage(`{"url":"https://example.com/file.pdf","filename":"file.pdf","size":1024,"mime_type":"application/pdf"}`)

	msg, err := NewMessage(convID, senderID, "file.pdf", MessageTypeFile, meta, 5)

	require.NoError(t, err)
	require.NotNil(t, msg)
	assert.Equal(t, MessageTypeFile, msg.Type)
	assert.Equal(t, "file.pdf", msg.Content)
	assert.NotNil(t, msg.Metadata)
}

func TestNewMessage_EmptyContent(t *testing.T) {
	msg, err := NewMessage(uuid.New(), uuid.New(), "", MessageTypeText, nil, 1)

	assert.ErrorIs(t, err, ErrEmptyContent)
	assert.Nil(t, msg)
}

func TestNewMessage_FileWithEmptyContent(t *testing.T) {
	// File messages with empty content are allowed (content check only applies to text)
	msg, err := NewMessage(uuid.New(), uuid.New(), "", MessageTypeFile, nil, 1)

	require.NoError(t, err)
	require.NotNil(t, msg)
	assert.Equal(t, MessageTypeFile, msg.Type)
}

func TestNewMessage_ValidVoiceMessage(t *testing.T) {
	convID := uuid.New()
	senderID := uuid.New()
	meta := json.RawMessage(`{"url":"https://example.com/voice.webm","duration":42.5,"size":8192,"mime_type":"audio/webm"}`)

	msg, err := NewMessage(convID, senderID, "", MessageTypeVoice, meta, 3)

	require.NoError(t, err)
	require.NotNil(t, msg)
	assert.Equal(t, MessageTypeVoice, msg.Type)
	assert.Empty(t, msg.Content)
	assert.NotNil(t, msg.Metadata)
}

func TestNewMessage_VoiceWithEmptyContent(t *testing.T) {
	msg, err := NewMessage(uuid.New(), uuid.New(), "", MessageTypeVoice, nil, 1)

	require.NoError(t, err)
	require.NotNil(t, msg)
	assert.Equal(t, MessageTypeVoice, msg.Type)
}

func TestNewMessage_ContentTooLong(t *testing.T) {
	longContent := strings.Repeat("a", MaxContentLength+1)
	msg, err := NewMessage(uuid.New(), uuid.New(), longContent, MessageTypeText, nil, 1)

	assert.ErrorIs(t, err, ErrContentTooLong)
	assert.Nil(t, msg)
}

func TestNewMessage_ContentExactlyAtMax(t *testing.T) {
	maxContent := strings.Repeat("a", MaxContentLength)
	msg, err := NewMessage(uuid.New(), uuid.New(), maxContent, MessageTypeText, nil, 1)

	require.NoError(t, err)
	require.NotNil(t, msg)
	assert.Len(t, msg.Content, MaxContentLength)
}

func TestNewMessage_InvalidType(t *testing.T) {
	msg, err := NewMessage(uuid.New(), uuid.New(), "hello", MessageType("image"), nil, 1)

	assert.ErrorIs(t, err, ErrInvalidMessageType)
	assert.Nil(t, msg)
}

func TestMessage_Edit(t *testing.T) {
	msg, _ := NewMessage(uuid.New(), uuid.New(), "original", MessageTypeText, nil, 1)
	originalUpdatedAt := msg.UpdatedAt

	msg.Edit("edited content")

	assert.Equal(t, "edited content", msg.Content)
	assert.NotNil(t, msg.EditedAt)
	assert.True(t, msg.UpdatedAt.After(originalUpdatedAt) || msg.UpdatedAt.Equal(originalUpdatedAt))
}

func TestMessage_SoftDelete(t *testing.T) {
	msg, _ := NewMessage(uuid.New(), uuid.New(), "to be deleted", MessageTypeText, nil, 1)

	msg.SoftDelete()

	assert.Empty(t, msg.Content)
	assert.NotNil(t, msg.DeletedAt)
}

func TestMessage_MarkDelivered(t *testing.T) {
	msg, _ := NewMessage(uuid.New(), uuid.New(), "hello", MessageTypeText, nil, 1)
	assert.Equal(t, MessageStatusSent, msg.Status)

	msg.MarkDelivered()

	assert.Equal(t, MessageStatusDelivered, msg.Status)
}

func TestMessage_MarkRead(t *testing.T) {
	msg, _ := NewMessage(uuid.New(), uuid.New(), "hello", MessageTypeText, nil, 1)

	msg.MarkRead()

	assert.Equal(t, MessageStatusRead, msg.Status)
}

func TestNewConversation(t *testing.T) {
	conv := NewConversation()

	assert.NotEqual(t, uuid.Nil, conv.ID)
	assert.False(t, conv.CreatedAt.IsZero())
	assert.False(t, conv.UpdatedAt.IsZero())
	assert.Equal(t, conv.CreatedAt, conv.UpdatedAt)
}

func TestNewConversation_UniqueIDs(t *testing.T) {
	conv1 := NewConversation()
	conv2 := NewConversation()

	assert.NotEqual(t, conv1.ID, conv2.ID)
}

func TestMessageType_IsValid(t *testing.T) {
	tests := []struct {
		name    string
		mt      MessageType
		isValid bool
	}{
		{"text is valid", MessageTypeText, true},
		{"file is valid", MessageTypeFile, true},
		{"voice is valid", MessageTypeVoice, true},
		{"image is invalid", MessageType("image"), false},
		{"empty is invalid", MessageType(""), false},
		{"audio is invalid", MessageType("audio"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.isValid, tt.mt.IsValid())
		})
	}
}

func TestMessageStatus_IsValid(t *testing.T) {
	tests := []struct {
		name    string
		ms      MessageStatus
		isValid bool
	}{
		{"sent is valid", MessageStatusSent, true},
		{"delivered is valid", MessageStatusDelivered, true},
		{"read is valid", MessageStatusRead, true},
		{"sending is invalid", MessageStatus("sending"), false},
		{"empty is invalid", MessageStatus(""), false},
		{"pending is invalid", MessageStatus("pending"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.isValid, tt.ms.IsValid())
		})
	}
}
