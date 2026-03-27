package call

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew_ValidAudioCall(t *testing.T) {
	c, err := New(uuid.New(), uuid.New(), uuid.New(), TypeAudio)
	require.NoError(t, err)
	assert.Equal(t, StatusRinging, c.Status)
	assert.Equal(t, TypeAudio, c.Type)
	assert.NotEmpty(t, c.RoomName)
	assert.Nil(t, c.StartedAt)
}

func TestNew_InvalidType(t *testing.T) {
	_, err := New(uuid.New(), uuid.New(), uuid.New(), Type("screenshare"))
	assert.ErrorIs(t, err, ErrInvalidCallType)
}

func TestNew_SelfCall(t *testing.T) {
	userID := uuid.New()
	_, err := New(uuid.New(), userID, userID, TypeAudio)
	assert.ErrorIs(t, err, ErrSelfCall)
}

func TestCall_Accept(t *testing.T) {
	c, _ := New(uuid.New(), uuid.New(), uuid.New(), TypeAudio)
	err := c.Accept()
	require.NoError(t, err)
	assert.Equal(t, StatusActive, c.Status)
	assert.NotNil(t, c.StartedAt)
}

func TestCall_AcceptFromActiveErr(t *testing.T) {
	c, _ := New(uuid.New(), uuid.New(), uuid.New(), TypeAudio)
	_ = c.Accept()
	err := c.Accept()
	assert.ErrorIs(t, err, ErrInvalidTransition)
}

func TestCall_Decline(t *testing.T) {
	c, _ := New(uuid.New(), uuid.New(), uuid.New(), TypeAudio)
	err := c.Decline()
	require.NoError(t, err)
	assert.Equal(t, StatusDeclined, c.Status)
}

func TestCall_DeclineFromActiveErr(t *testing.T) {
	c, _ := New(uuid.New(), uuid.New(), uuid.New(), TypeAudio)
	_ = c.Accept()
	err := c.Decline()
	assert.ErrorIs(t, err, ErrInvalidTransition)
}

func TestCall_End(t *testing.T) {
	c, _ := New(uuid.New(), uuid.New(), uuid.New(), TypeAudio)
	_ = c.Accept()
	err := c.End(120)
	require.NoError(t, err)
	assert.Equal(t, StatusEnded, c.Status)
	assert.Equal(t, 120, c.Duration)
}

func TestCall_EndFromRinging(t *testing.T) {
	c, _ := New(uuid.New(), uuid.New(), uuid.New(), TypeAudio)
	err := c.End(0)
	require.NoError(t, err)
	assert.Equal(t, StatusEnded, c.Status)
}

func TestCall_Miss(t *testing.T) {
	c, _ := New(uuid.New(), uuid.New(), uuid.New(), TypeAudio)
	err := c.Miss()
	require.NoError(t, err)
	assert.Equal(t, StatusMissed, c.Status)
}

func TestCall_MissFromActiveErr(t *testing.T) {
	c, _ := New(uuid.New(), uuid.New(), uuid.New(), TypeAudio)
	_ = c.Accept()
	err := c.Miss()
	assert.ErrorIs(t, err, ErrInvalidTransition)
}

func TestStatus_IsValid(t *testing.T) {
	tests := []struct {
		status Status
		valid  bool
	}{
		{StatusRinging, true},
		{StatusActive, true},
		{StatusDeclined, true},
		{StatusMissed, true},
		{StatusEnded, true},
		{Status("unknown"), false},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.valid, tt.status.IsValid(), "status=%s", tt.status)
	}
}

func TestType_IsValid(t *testing.T) {
	assert.True(t, TypeAudio.IsValid())
	assert.True(t, TypeVideo.IsValid())
	assert.False(t, Type("screenshare").IsValid())
}
