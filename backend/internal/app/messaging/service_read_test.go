package messaging

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/domain/message"
)

func TestMarkAsRead_RepoError(t *testing.T) {
	msgRepo := &mockMessageRepo{
		isOrgAuthorizedFn: func(_ context.Context, _, _ uuid.UUID) (bool, error) {
			return true, nil
		},
		markAsReadFn: func(_ context.Context, _, _ uuid.UUID, _ int) error {
			return fmt.Errorf("db connection lost")
		},
	}

	svc := newTestService(msgRepo, nil, nil, nil, nil, nil)

	err := svc.MarkAsRead(context.Background(), MarkAsReadInput{
		UserID:         uuid.New(),
		OrgID:          uuid.New(),
		ConversationID: uuid.New(),
		Seq:            5,
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mark as read")
}

func TestMarkAsRead_AuthorizationCheckError(t *testing.T) {
	msgRepo := &mockMessageRepo{
		isOrgAuthorizedFn: func(_ context.Context, _, _ uuid.UUID) (bool, error) {
			return false, fmt.Errorf("db error")
		},
	}

	svc := newTestService(msgRepo, nil, nil, nil, nil, nil)

	err := svc.MarkAsRead(context.Background(), MarkAsReadInput{
		UserID:         uuid.New(),
		OrgID:          uuid.New(),
		ConversationID: uuid.New(),
		Seq:            1,
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "check org authorized")
}

func TestMarkAsRead_ContinuesOnMarkMessagesError(t *testing.T) {
	// MarkMessagesAsRead failure is logged but does not fail the request.
	var markAsReadCalled bool
	msgRepo := &mockMessageRepo{
		isOrgAuthorizedFn: func(_ context.Context, _, _ uuid.UUID) (bool, error) {
			return true, nil
		},
		markAsReadFn: func(_ context.Context, _, _ uuid.UUID, _ int) error {
			markAsReadCalled = true
			return nil
		},
		markMessagesAsReadFn: func(_ context.Context, _, _ uuid.UUID, _ int) error {
			return fmt.Errorf("partial failure")
		},
	}

	svc := newTestService(msgRepo, nil, nil, nil, nil, nil)

	err := svc.MarkAsRead(context.Background(), MarkAsReadInput{
		UserID:         uuid.New(),
		OrgID:          uuid.New(),
		ConversationID: uuid.New(),
		Seq:            20,
	})

	require.NoError(t, err, "MarkMessagesAsRead failure should not propagate")
	assert.True(t, markAsReadCalled)
}

func TestDeliverMessage_MessageNotFound(t *testing.T) {
	msgRepo := &mockMessageRepo{
		getMessageFn: func(_ context.Context, _ uuid.UUID) (*message.Message, error) {
			return nil, message.ErrMessageNotFound
		},
	}

	svc := newTestService(msgRepo, nil, nil, nil, nil, nil)

	err := svc.DeliverMessage(context.Background(), uuid.New(), uuid.New())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get message")
}

func TestDeliverMessage_UpdateStatusError(t *testing.T) {
	msgRepo := &mockMessageRepo{
		getMessageFn: func(_ context.Context, id uuid.UUID) (*message.Message, error) {
			return &message.Message{
				ID:             id,
				ConversationID: uuid.New(),
				Status:         message.MessageStatusSent,
			}, nil
		},
		isOrgAuthorizedFn: func(_ context.Context, _, _ uuid.UUID) (bool, error) {
			return true, nil
		},
		updateMessageStatusFn: func(_ context.Context, _ uuid.UUID, _ message.MessageStatus) error {
			return fmt.Errorf("db write error")
		},
	}

	svc := newTestService(msgRepo, nil, nil, nil, nil, nil)

	err := svc.DeliverMessage(context.Background(), uuid.New(), uuid.New())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "db write error")
}

func TestGetTotalUnread_ZeroCount(t *testing.T) {
	msgRepo := &mockMessageRepo{
		getTotalUnreadFn: func(_ context.Context, _ uuid.UUID) (int, error) {
			return 0, nil
		},
	}

	svc := newTestService(msgRepo, nil, nil, nil, nil, nil)

	count, err := svc.GetTotalUnread(context.Background(), uuid.New())

	require.NoError(t, err)
	assert.Equal(t, 0, count)
}
