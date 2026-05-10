package twofactor

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNew_ValidatesInputs covers every rejection branch of New().
// Table-driven so adding a new validation rule is one line + one row.
func TestNew_ValidatesInputs(t *testing.T) {
	validUser := uuid.New()
	tests := []struct {
		name    string
		in      NewChallengeInput
		wantErr error
	}{
		{
			name: "happy path",
			in: NewChallengeInput{
				UserID:   validUser,
				CodeHash: "$2a$10$abcdef",
			},
		},
		{
			name: "zero user id rejected",
			in: NewChallengeInput{
				UserID:   uuid.Nil,
				CodeHash: "$2a$10$abcdef",
			},
			wantErr: ErrUserIDRequired,
		},
		{
			name: "empty code hash rejected",
			in: NewChallengeInput{
				UserID:   validUser,
				CodeHash: "",
			},
			wantErr: ErrCodeHashRequired,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := New(tt.in)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, c)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, c)
			assert.Equal(t, validUser, c.UserID)
			assert.Equal(t, DefaultAttempts, c.AttemptsLeft)
			assert.True(t, c.ExpiresAt.After(time.Now().Add(DefaultTTL-time.Second)))
		})
	}
}

func TestNew_DefaultsAndOverrides(t *testing.T) {
	uid := uuid.New()
	c, err := New(NewChallengeInput{
		UserID:       uid,
		CodeHash:     "h",
		AttemptsLeft: 3,
		TTL:          5 * time.Minute,
	})
	require.NoError(t, err)
	assert.Equal(t, 3, c.AttemptsLeft)
	assert.True(t, c.ExpiresAt.Before(time.Now().Add(6*time.Minute)))
	assert.True(t, c.ExpiresAt.After(time.Now().Add(4*time.Minute)))
}

func TestChallenge_IsExpired(t *testing.T) {
	uid := uuid.New()

	t.Run("fresh challenge is not expired", func(t *testing.T) {
		c, err := New(NewChallengeInput{UserID: uid, CodeHash: "h"})
		require.NoError(t, err)
		assert.False(t, c.IsExpired())
	})

	t.Run("expired when clock moves past expiry", func(t *testing.T) {
		c, err := New(NewChallengeInput{UserID: uid, CodeHash: "h"})
		require.NoError(t, err)
		t.Cleanup(RestoreNow)
		SetNowForTests(func() time.Time { return c.ExpiresAt.Add(time.Second) })
		assert.True(t, c.IsExpired())
	})
}

func TestChallenge_IsPending(t *testing.T) {
	uid := uuid.New()
	c, err := New(NewChallengeInput{UserID: uid, CodeHash: "h"})
	require.NoError(t, err)

	assert.True(t, c.IsPending(), "fresh challenge is pending")

	c.MarkUsed()
	assert.False(t, c.IsPending(), "used challenge is not pending")
}

func TestChallenge_DecrementAttemptsFloor(t *testing.T) {
	c, err := New(NewChallengeInput{
		UserID:       uuid.New(),
		CodeHash:     "h",
		AttemptsLeft: 1,
	})
	require.NoError(t, err)
	c.DecrementAttempts()
	assert.Equal(t, 0, c.AttemptsLeft)
	c.DecrementAttempts() // floor — must not go negative
	assert.Equal(t, 0, c.AttemptsLeft)
	assert.False(t, c.IsPending(), "zero attempts means not pending")
}

func TestChallenge_MarkUsed(t *testing.T) {
	c, err := New(NewChallengeInput{UserID: uuid.New(), CodeHash: "h"})
	require.NoError(t, err)
	assert.Nil(t, c.UsedAt)
	assert.False(t, c.IsUsed())

	c.MarkUsed()
	assert.NotNil(t, c.UsedAt)
	assert.True(t, c.IsUsed())
}
