package expertise

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsValidKey(t *testing.T) {
	tests := []struct {
		name string
		key  string
		want bool
	}{
		{"empty string is rejected", "", false},
		{"unknown key is rejected", "blockchain", false},
		{"whitespace-padded key is rejected", " development ", false},
		{"case mismatch is rejected", "Development", false},

		{"development is accepted", "development", true},
		{"data_ai_ml is accepted", "data_ai_ml", true},
		{"design_ui_ux is accepted", "design_ui_ux", true},
		{"design_3d_animation is accepted", "design_3d_animation", true},
		{"video_motion is accepted", "video_motion", true},
		{"photo_audiovisual is accepted", "photo_audiovisual", true},
		{"marketing_growth is accepted", "marketing_growth", true},
		{"writing_translation is accepted", "writing_translation", true},
		{"business_dev_sales is accepted", "business_dev_sales", true},
		{"consulting_strategy is accepted", "consulting_strategy", true},
		{"product_ux_research is accepted", "product_ux_research", true},
		{"ops_admin_support is accepted", "ops_admin_support", true},
		{"legal is accepted", "legal", true},
		{"finance_accounting is accepted", "finance_accounting", true},
		{"hr_recruitment is accepted", "hr_recruitment", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, IsValidKey(tc.key))
		})
	}
}

func TestAll_catalogHasExpectedCount(t *testing.T) {
	// This test is a guard: if somebody adds or removes a catalog key
	// without updating the frontends' i18n files, the PR author sees
	// this test fail and is forced to update both sides together.
	// The expected count is wired to the LOCKED 15-key spec; bumping
	// it here is a deliberate product decision.
	assert.Len(t, All, 15, "catalog must contain exactly 15 keys — if you add one, update web and mobile i18n files too")
}

func TestAll_entriesAreUnique(t *testing.T) {
	seen := make(map[string]struct{}, len(All))
	for _, k := range All {
		_, dup := seen[k]
		assert.False(t, dup, "duplicate catalog entry: %s", k)
		seen[k] = struct{}{}
	}
}

func TestAll_entriesAreAllValid(t *testing.T) {
	// IsValidKey and All share state through the validKeys init
	// function; this test catches any future refactor that breaks
	// the invariant "every entry in All is valid per IsValidKey".
	for _, k := range All {
		assert.True(t, IsValidKey(k), "catalog entry %q should be a valid key", k)
	}
}

func TestMaxForOrgType(t *testing.T) {
	tests := []struct {
		name    string
		orgType OrgType
		want    int
	}{
		{"agency has max 8", OrgTypeAgency, 8},
		{"provider_personal has max 5", OrgTypeProviderPersonal, 5},
		{"enterprise is forbidden (0)", OrgTypeEnterprise, 0},
		{"empty type is safe-defaulted to 0", "", 0},
		{"unknown future type is safe-defaulted to 0", "platform_admin", 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, MaxForOrgType(tc.orgType))
		})
	}
}

func TestIsFeatureEnabled(t *testing.T) {
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
			assert.Equal(t, tc.want, IsFeatureEnabled(tc.orgType))
		})
	}
}
