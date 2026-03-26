package profile

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProfile_CreatesWithCorrectDefaults(t *testing.T) {
	userID := uuid.New()

	p := NewProfile(userID)

	require.NotNil(t, p)
	assert.Equal(t, userID, p.UserID)
	assert.Empty(t, p.Title, "title should be empty by default")
	assert.Empty(t, p.About, "about should be empty by default")
	assert.Empty(t, p.PhotoURL, "photo url should be empty by default")
	assert.Empty(t, p.PresentationVideoURL, "presentation video should be empty by default")
	assert.Empty(t, p.ReferrerAbout, "referrer about should be empty by default")
	assert.Empty(t, p.ReferrerVideoURL, "referrer video should be empty by default")
	assert.False(t, p.CreatedAt.IsZero(), "created_at should be set")
	assert.False(t, p.UpdatedAt.IsZero(), "updated_at should be set")
}

func TestNewProfile_TimestampsAreClose(t *testing.T) {
	p := NewProfile(uuid.New())

	// created_at and updated_at should be set to the same time on creation
	assert.Equal(t, p.CreatedAt, p.UpdatedAt, "timestamps should match on creation")
}

func TestNewProfile_DifferentUsersGetDifferentProfiles(t *testing.T) {
	userA := uuid.New()
	userB := uuid.New()

	pA := NewProfile(userA)
	pB := NewProfile(userB)

	assert.NotEqual(t, pA.UserID, pB.UserID)
}
