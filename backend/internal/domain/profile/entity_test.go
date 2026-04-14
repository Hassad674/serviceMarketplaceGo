package profile

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---- NewProfile / defaults ----

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

	// Tier 1 defaults (migration 083)
	assert.Empty(t, p.City)
	assert.Empty(t, p.CountryCode)
	assert.Nil(t, p.Latitude)
	assert.Nil(t, p.Longitude)
	assert.NotNil(t, p.WorkMode, "work_mode should be a non-nil empty slice")
	assert.Len(t, p.WorkMode, 0)
	assert.Nil(t, p.TravelRadiusKm)
	assert.NotNil(t, p.LanguagesProfessional)
	assert.Len(t, p.LanguagesProfessional, 0)
	assert.NotNil(t, p.LanguagesConversational)
	assert.Len(t, p.LanguagesConversational, 0)
	assert.Equal(t, AvailabilityNow, p.AvailabilityStatus,
		"new orgs should default to available_now so they show up in listings")
	assert.Nil(t, p.ReferrerAvailabilityStatus,
		"referrer slot should start nil — only set after referrer mode enables")
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

// ---- Work mode ----

func TestIsValidWorkMode(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{"remote", WorkModeRemote, true},
		{"on_site", WorkModeOnSite, true},
		{"hybrid", WorkModeHybrid, true},
		{"empty", "", false},
		{"unknown", "nomad", false},
		{"wrong case", "Remote", false},
		{"whitespace", "remote ", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, IsValidWorkMode(tc.in))
		})
	}
}

func TestNormalizeWorkModes(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{"nil input yields empty non-nil slice", nil, []string{}},
		{"empty slice yields empty slice", []string{}, []string{}},
		{"all valid passes through", []string{"remote", "on_site"}, []string{"remote", "on_site"}},
		{"duplicates deduped, first-occurrence order", []string{"remote", "remote", "hybrid"}, []string{"remote", "hybrid"}},
		{"invalid entries dropped", []string{"remote", "nomad", "hybrid"}, []string{"remote", "hybrid"}},
		{"all invalid yields empty slice", []string{"nomad", "space"}, []string{}},
		{"mixed case dropped", []string{"Remote", "hybrid"}, []string{"hybrid"}},
		{"full catalog", []string{"remote", "on_site", "hybrid"}, []string{"remote", "on_site", "hybrid"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := NormalizeWorkModes(tc.in)
			require.NotNil(t, got, "result must never be nil")
			assert.Equal(t, tc.want, got)
		})
	}
}

// ---- Country code ----

func TestValidateCountryCode(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		wantErr error
	}{
		{"empty is valid (unset)", "", nil},
		{"FR is valid", "FR", nil},
		{"US is valid", "US", nil},
		{"one letter too short", "F", ErrInvalidCountryCode},
		{"three letters too long", "FRA", ErrInvalidCountryCode},
		{"lowercase rejected", "fr", ErrInvalidCountryCode},
		{"mixed case rejected", "Fr", ErrInvalidCountryCode},
		{"digits rejected", "12", ErrInvalidCountryCode},
		{"symbols rejected", "F!", ErrInvalidCountryCode},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateCountryCode(tc.in)
			if tc.wantErr == nil {
				assert.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, tc.wantErr)
			}
		})
	}
}

// ---- Language codes ----

func TestIsValidLanguageCode(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{"fr", "fr", true},
		{"en", "en", true},
		{"es", "es", true},
		{"uppercase rejected", "FR", false},
		{"mixed rejected", "Fr", false},
		{"one letter rejected", "f", false},
		{"three letters rejected", "fra", false},
		{"empty rejected", "", false},
		{"hyphenated tag rejected", "fr-fr", false},
		{"digits rejected", "f1", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, IsValidLanguageCode(tc.in))
		})
	}
}

func TestNormalizeLanguageCodes(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{"nil yields empty non-nil", nil, []string{}},
		{"all valid", []string{"fr", "en"}, []string{"fr", "en"}},
		{"dedupes", []string{"fr", "fr", "en"}, []string{"fr", "en"}},
		{"drops invalids", []string{"fr", "FR", "EN", "en"}, []string{"fr", "en"}},
		{"all invalid yields empty", []string{"french", "English"}, []string{}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := NormalizeLanguageCodes(tc.in)
			require.NotNil(t, got)
			assert.Equal(t, tc.want, got)
		})
	}
}

// ---- Availability status ----

func TestAvailabilityStatus_IsValid(t *testing.T) {
	tests := []struct {
		name string
		in   AvailabilityStatus
		want bool
	}{
		{"available_now", AvailabilityNow, true},
		{"available_soon", AvailabilitySoon, true},
		{"not_available", AvailabilityNot, true},
		{"empty rejected", "", false},
		{"unknown rejected", AvailabilityStatus("maybe"), false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.in.IsValid())
		})
	}
}

func TestParseAvailabilityStatus(t *testing.T) {
	t.Run("valid value", func(t *testing.T) {
		got, err := ParseAvailabilityStatus("available_soon")
		require.NoError(t, err)
		assert.Equal(t, AvailabilitySoon, got)
	})
	t.Run("unknown value", func(t *testing.T) {
		_, err := ParseAvailabilityStatus("unknown")
		assert.ErrorIs(t, err, ErrInvalidAvailabilityStatus)
	})
	t.Run("empty value", func(t *testing.T) {
		_, err := ParseAvailabilityStatus("")
		assert.ErrorIs(t, err, ErrInvalidAvailabilityStatus)
	})
}
