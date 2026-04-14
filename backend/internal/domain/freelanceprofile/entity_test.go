package freelanceprofile_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/domain/freelanceprofile"
	"marketplace-backend/internal/domain/profile"
)

func TestNew_SeedsDefaults(t *testing.T) {
	orgID := uuid.New()
	p := freelanceprofile.New(orgID)

	require.NotNil(t, p)
	assert.NotEqual(t, uuid.Nil, p.ID, "surrogate ID must be allocated")
	assert.Equal(t, orgID, p.OrganizationID)
	assert.Equal(t, profile.AvailabilityNow, p.AvailabilityStatus,
		"default availability is available_now")
	assert.NotNil(t, p.ExpertiseDomains, "expertise slice must not be nil")
	assert.Empty(t, p.ExpertiseDomains)
	assert.Empty(t, p.Title)
	assert.Empty(t, p.About)
	assert.Empty(t, p.VideoURL)
	assert.False(t, p.CreatedAt.IsZero())
	assert.False(t, p.UpdatedAt.IsZero())
}

func TestUpdateCore_OverwritesAllThreeFields(t *testing.T) {
	p := freelanceprofile.New(uuid.New())
	before := p.UpdatedAt

	p.UpdateCore("Senior Go Engineer", "Builds marketplaces.", "https://example.com/v.mp4")

	assert.Equal(t, "Senior Go Engineer", p.Title)
	assert.Equal(t, "Builds marketplaces.", p.About)
	assert.Equal(t, "https://example.com/v.mp4", p.VideoURL)
	assert.True(t, p.UpdatedAt.After(before) || p.UpdatedAt.Equal(before),
		"updated_at must not regress")
}

func TestUpdateCore_AcceptsEmptyStringsVerbatim(t *testing.T) {
	p := freelanceprofile.New(uuid.New())
	p.UpdateCore("Existing", "Existing about", "https://old.example/v.mp4")

	p.UpdateCore("", "", "")

	assert.Empty(t, p.Title)
	assert.Empty(t, p.About)
	assert.Empty(t, p.VideoURL)
}

func TestUpdateAvailability_ValidValues(t *testing.T) {
	tests := []struct {
		name   string
		status profile.AvailabilityStatus
	}{
		{"available_now", profile.AvailabilityNow},
		{"available_soon", profile.AvailabilitySoon},
		{"not_available", profile.AvailabilityNot},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := freelanceprofile.New(uuid.New())
			err := p.UpdateAvailability(tt.status)
			assert.NoError(t, err)
			assert.Equal(t, tt.status, p.AvailabilityStatus)
		})
	}
}

func TestUpdateAvailability_InvalidValue(t *testing.T) {
	p := freelanceprofile.New(uuid.New())
	err := p.UpdateAvailability(profile.AvailabilityStatus("definitely_not_valid"))
	assert.ErrorIs(t, err, profile.ErrInvalidAvailabilityStatus)
}

func TestUpdateExpertiseDomains_ReplacesList(t *testing.T) {
	p := freelanceprofile.New(uuid.New())

	p.UpdateExpertiseDomains([]string{"development", "design_ui_ux"})
	assert.Equal(t, []string{"development", "design_ui_ux"}, p.ExpertiseDomains)

	p.UpdateExpertiseDomains([]string{"marketing"})
	assert.Equal(t, []string{"marketing"}, p.ExpertiseDomains)
}

func TestUpdateExpertiseDomains_NilBecomesEmptySlice(t *testing.T) {
	p := freelanceprofile.New(uuid.New())
	p.UpdateExpertiseDomains(nil)

	assert.NotNil(t, p.ExpertiseDomains)
	assert.Empty(t, p.ExpertiseDomains)
}
