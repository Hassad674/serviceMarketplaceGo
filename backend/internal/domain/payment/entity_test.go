package payment

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPaymentInfo(t *testing.T) {
	userID := uuid.New()

	info := NewPaymentInfo(userID)

	require.NotNil(t, info)
	assert.Equal(t, userID, info.UserID)
	assert.NotEqual(t, uuid.Nil, info.ID)
	assert.False(t, info.CreatedAt.IsZero())
	assert.False(t, info.UpdatedAt.IsZero())
	assert.Empty(t, info.StripeAccountID)
	assert.False(t, info.StripeVerified)
}

func TestSetStripeAccount(t *testing.T) {
	info := NewPaymentInfo(uuid.New())

	info.SetStripeAccount("acct_test_123")

	assert.Equal(t, "acct_test_123", info.StripeAccountID)
}

func TestMarkStripeVerified(t *testing.T) {
	info := NewPaymentInfo(uuid.New())

	info.MarkStripeVerified()

	assert.True(t, info.StripeVerified)
}
