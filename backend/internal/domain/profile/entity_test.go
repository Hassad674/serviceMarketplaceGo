package profile

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProfile_CreatesWithCorrectDefaults(t *testing.T) {
	orgID := uuid.New()

	p := NewProfile(orgID)

	require.NotNil(t, p)
	assert.Equal(t, orgID, p.OrganizationID)
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
	assert.Equal(t, p.CreatedAt, p.UpdatedAt, "timestamps should match on creation")
}

func TestNewProfile_DifferentOrgsGetDifferentProfiles(t *testing.T) {
	orgA := uuid.New()
	orgB := uuid.New()

	pA := NewProfile(orgA)
	pB := NewProfile(orgB)

	assert.NotEqual(t, pA.OrganizationID, pB.OrganizationID)
}
