package user

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccountStatusError_Suspended(t *testing.T) {
	err := NewSuspendedError("policy violation")

	assert.True(t, errors.Is(err, ErrAccountSuspended))
	assert.False(t, errors.Is(err, ErrAccountBanned))
	assert.Equal(t, ErrAccountSuspended.Error(), err.Error())

	var statusErr *AccountStatusError
	require.True(t, errors.As(err, &statusErr))
	assert.Equal(t, "policy violation", statusErr.Reason)
	assert.Equal(t, ErrAccountSuspended, statusErr.Sentinel)
}

func TestAccountStatusError_Banned(t *testing.T) {
	err := NewBannedError("ban reason")

	assert.True(t, errors.Is(err, ErrAccountBanned))
	assert.False(t, errors.Is(err, ErrAccountSuspended))
	assert.Equal(t, ErrAccountBanned.Error(), err.Error())

	var statusErr *AccountStatusError
	require.True(t, errors.As(err, &statusErr))
	assert.Equal(t, "ban reason", statusErr.Reason)
	assert.Equal(t, ErrAccountBanned, statusErr.Sentinel)
}

func TestAccountStatusError_Unwrap(t *testing.T) {
	suspended := NewSuspendedError("test")
	assert.Equal(t, ErrAccountSuspended, suspended.Unwrap())

	banned := NewBannedError("test")
	assert.Equal(t, ErrAccountBanned, banned.Unwrap())
}
