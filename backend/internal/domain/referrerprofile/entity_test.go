package referrerprofile_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/domain/profile"
	"marketplace-backend/internal/domain/referrerprofile"
)

func TestNew_SeedsDefaults(t *testing.T) {
	orgID := uuid.New()
	p := referrerprofile.New(orgID)

	require.NotNil(t, p)
	assert.NotEqual(t, uuid.Nil, p.ID)
	assert.Equal(t, orgID, p.OrganizationID)
	assert.Equal(t, profile.AvailabilityNow, p.AvailabilityStatus)
	assert.NotNil(t, p.ExpertiseDomains)
	assert.Empty(t, p.ExpertiseDomains)
	assert.False(t, p.CreatedAt.IsZero())
	assert.False(t, p.UpdatedAt.IsZero())
}

func TestUpdateCore_OverwritesAllThreeFields(t *testing.T) {
	p := referrerprofile.New(uuid.New())
	p.UpdateCore("Top Apporteur", "Finds deals", "https://example.com/v.mp4")

	assert.Equal(t, "Top Apporteur", p.Title)
	assert.Equal(t, "Finds deals", p.About)
	assert.Equal(t, "https://example.com/v.mp4", p.VideoURL)
}

func TestUpdateAvailability_ValidValues(t *testing.T) {
	cases := []profile.AvailabilityStatus{
		profile.AvailabilityNow,
		profile.AvailabilitySoon,
		profile.AvailabilityNot,
	}
	for _, status := range cases {
		t.Run(string(status), func(t *testing.T) {
			p := referrerprofile.New(uuid.New())
			err := p.UpdateAvailability(status)
			assert.NoError(t, err)
			assert.Equal(t, status, p.AvailabilityStatus)
		})
	}
}

func TestUpdateAvailability_InvalidValue(t *testing.T) {
	p := referrerprofile.New(uuid.New())
	err := p.UpdateAvailability(profile.AvailabilityStatus(""))
	assert.ErrorIs(t, err, profile.ErrInvalidAvailabilityStatus)
}

func TestUpdateExpertiseDomains_ReplacesListAndHandlesNil(t *testing.T) {
	p := referrerprofile.New(uuid.New())

	p.UpdateExpertiseDomains([]string{"development"})
	assert.Equal(t, []string{"development"}, p.ExpertiseDomains)

	p.UpdateExpertiseDomains(nil)
	assert.NotNil(t, p.ExpertiseDomains)
	assert.Empty(t, p.ExpertiseDomains)
}
