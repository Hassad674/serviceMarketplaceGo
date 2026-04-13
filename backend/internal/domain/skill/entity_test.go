package skill

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeSkillText(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"simple lowercase", "react", "react"},
		{"simple titlecase", "React", "react"},
		{"all uppercase", "REACT", "react"},
		{"leading/trailing spaces", "  React  ", "react"},
		{"mixed case with dots", "Next.js", "next.js"},
		{"uppercase with dots", "NEXT.JS", "next.js"},
		{"internal multi space", "React  JS", "react js"},
		{"internal tab and spaces", "React\t JS", "react js"},
		{"leading trailing and internal", " React   JS ", "react js"},
		{"double space between words", "A  B", "a b"},
		{"triple internal space with mixed case", "  REACT  NATIVE  ", "react native"},
		{"empty string", "", ""},
		{"whitespace only", "   ", ""},
		{"tabs and newlines only", "\t\n\r ", ""},
		{"unicode non-breaking space trimmed", "\u00a0React\u00a0", "react"},
		{"single char", "C", "c"},
		{"preserves punctuation", "c++", "c++"},
		{"preserves hyphen", "front-end", "front-end"},
		{"collapses multi unicode spaces", "go\u00a0\u00a0lang", "go lang"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, NormalizeSkillText(tc.in))
		})
	}
}

func TestNewCatalogEntry_Valid(t *testing.T) {
	t.Run("curated entry with multiple expertise keys", func(t *testing.T) {
		entry, err := NewCatalogEntry(
			"  React  ",
			"React",
			[]string{"development", "design_ui_ux"},
			true,
		)

		require.NoError(t, err)
		require.NotNil(t, entry)
		assert.Equal(t, "react", entry.SkillText)
		assert.Equal(t, "React", entry.DisplayText)
		assert.Equal(t, []string{"development", "design_ui_ux"}, entry.ExpertiseKeys)
		assert.True(t, entry.IsCurated)
		assert.Equal(t, 0, entry.UsageCount)
		assert.True(t, entry.CreatedAt.IsZero(), "repository layer fills timestamps")
		assert.True(t, entry.UpdatedAt.IsZero(), "repository layer fills timestamps")
	})

	t.Run("user-created entry with one expertise key", func(t *testing.T) {
		entry, err := NewCatalogEntry(
			"TypeScript",
			"TypeScript",
			[]string{"development"},
			false,
		)

		require.NoError(t, err)
		assert.Equal(t, "typescript", entry.SkillText)
		assert.Equal(t, "TypeScript", entry.DisplayText)
		assert.Equal(t, []string{"development"}, entry.ExpertiseKeys)
		assert.False(t, entry.IsCurated)
	})

	t.Run("display text trimmed of surrounding whitespace", func(t *testing.T) {
		entry, err := NewCatalogEntry("figma", "  Figma  ", nil, true)

		require.NoError(t, err)
		assert.Equal(t, "figma", entry.SkillText)
		assert.Equal(t, "Figma", entry.DisplayText)
	})

	t.Run("nil expertise keys normalize to empty slice", func(t *testing.T) {
		entry, err := NewCatalogEntry("docker", "Docker", nil, true)

		require.NoError(t, err)
		require.NotNil(t, entry.ExpertiseKeys)
		assert.Empty(t, entry.ExpertiseKeys)
	})

	t.Run("empty expertise keys normalize to empty non-nil slice", func(t *testing.T) {
		entry, err := NewCatalogEntry("kubernetes", "Kubernetes", []string{}, false)

		require.NoError(t, err)
		require.NotNil(t, entry.ExpertiseKeys)
		assert.Empty(t, entry.ExpertiseKeys)
	})
}

func TestNewCatalogEntry_InvalidSkillText(t *testing.T) {
	cases := []struct {
		name string
		raw  string
	}{
		{"empty string", ""},
		{"single space", " "},
		{"many spaces", "     "},
		{"tabs and newlines", "\t\n\r"},
		{"unicode non-breaking space only", "\u00a0\u00a0"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			entry, err := NewCatalogEntry(tc.raw, "Display", []string{"development"}, false)
			assert.Nil(t, entry)
			require.Error(t, err)
			assert.True(t, errors.Is(err, ErrInvalidSkillText))
		})
	}
}

func TestNewCatalogEntry_InvalidDisplayText(t *testing.T) {
	cases := []struct {
		name    string
		display string
	}{
		{"empty string", ""},
		{"single space", " "},
		{"tabs only", "\t\t"},
		{"mixed whitespace", " \t\n "},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			entry, err := NewCatalogEntry("react", tc.display, []string{"development"}, false)
			assert.Nil(t, entry)
			require.Error(t, err)
			assert.True(t, errors.Is(err, ErrInvalidDisplayText))
		})
	}
}

func TestNewCatalogEntry_DeduplicatesExpertiseKeys(t *testing.T) {
	t.Run("drops repeat keys keeping first occurrence", func(t *testing.T) {
		entry, err := NewCatalogEntry(
			"figma",
			"Figma",
			[]string{"design_ui_ux", "design_ui_ux", "design_3d_animation"},
			true,
		)

		require.NoError(t, err)
		assert.Equal(t, []string{"design_ui_ux", "design_3d_animation"}, entry.ExpertiseKeys)
	})

	t.Run("preserves order of first occurrence across many dupes", func(t *testing.T) {
		entry, err := NewCatalogEntry(
			"photoshop",
			"Photoshop",
			[]string{"design_ui_ux", "photo_audiovisual", "design_ui_ux", "photo_audiovisual", "design_3d_animation"},
			true,
		)

		require.NoError(t, err)
		assert.Equal(t,
			[]string{"design_ui_ux", "photo_audiovisual", "design_3d_animation"},
			entry.ExpertiseKeys,
		)
	})

	t.Run("no duplicates leaves list unchanged", func(t *testing.T) {
		entry, err := NewCatalogEntry(
			"typescript",
			"TypeScript",
			[]string{"development", "data_ai_ml"},
			true,
		)

		require.NoError(t, err)
		assert.Equal(t, []string{"development", "data_ai_ml"}, entry.ExpertiseKeys)
	})
}

func TestCatalogEntry_IsInExpertise(t *testing.T) {
	entry := &CatalogEntry{
		SkillText:     "react",
		DisplayText:   "React",
		ExpertiseKeys: []string{"development", "design_ui_ux"},
	}

	tests := []struct {
		name string
		key  string
		want bool
	}{
		{"first key matches", "development", true},
		{"last key matches", "design_ui_ux", true},
		{"unknown key does not match", "marketing_growth", false},
		{"empty key does not match", "", false},
		{"case mismatch does not match", "Development", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, entry.IsInExpertise(tc.key))
		})
	}

	t.Run("empty expertise list returns false for every key", func(t *testing.T) {
		empty := &CatalogEntry{SkillText: "x", DisplayText: "X"}
		assert.False(t, empty.IsInExpertise("development"))
		assert.False(t, empty.IsInExpertise(""))
	})
}

func TestMaxSkillsForOrgType(t *testing.T) {
	tests := []struct {
		name    string
		orgType OrgType
		want    int
	}{
		{"agency has max 40", OrgTypeAgency, 40},
		{"provider_personal has max 25", OrgTypeProviderPersonal, 25},
		{"enterprise is forbidden (0)", OrgTypeEnterprise, 0},
		{"empty type is safe-defaulted to 0", "", 0},
		{"unknown future type is safe-defaulted to 0", "platform_admin", 0},
		{"random garbage string is safe-defaulted to 0", "not-a-role", 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, MaxSkillsForOrgType(tc.orgType))
		})
	}
}

func TestIsSkillsFeatureEnabled(t *testing.T) {
	tests := []struct {
		name    string
		orgType OrgType
		want    bool
	}{
		{"agency enabled", OrgTypeAgency, true},
		{"provider_personal enabled", OrgTypeProviderPersonal, true},
		{"enterprise disabled", OrgTypeEnterprise, false},
		{"empty disabled", "", false},
		{"unknown disabled", "platform_admin", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, IsSkillsFeatureEnabled(tc.orgType))
		})
	}
}
