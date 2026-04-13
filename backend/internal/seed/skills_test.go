package seed

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/domain/expertise"
	"marketplace-backend/internal/domain/skill"
)

func TestCuratedSkills_NoDuplicates(t *testing.T) {
	seen := make(map[string]string)
	for _, s := range CuratedSkills {
		if existing, dup := seen[s.SkillText]; dup {
			t.Errorf("duplicate skill_text %q: %q collides with %q", s.SkillText, s.DisplayText, existing)
		}
		seen[s.SkillText] = s.DisplayText
	}
}

func TestCuratedSkills_SkillTextMatchesNormalized(t *testing.T) {
	for _, s := range CuratedSkills {
		expected := skill.NormalizeSkillText(s.DisplayText)
		assert.Equal(t, expected, s.SkillText,
			"SkillText for %q should be normalized to %q, got %q",
			s.DisplayText, expected, s.SkillText)
	}
}

func TestCuratedSkills_AllExpertiseKeysValid(t *testing.T) {
	for _, s := range CuratedSkills {
		for _, key := range s.ExpertiseKeys {
			assert.True(t, expertise.IsValidKey(key),
				"skill %q references unknown expertise key %q", s.DisplayText, key)
		}
	}
}

func TestCuratedSkills_AtLeastOneExpertiseKey(t *testing.T) {
	for _, s := range CuratedSkills {
		assert.NotEmpty(t, s.ExpertiseKeys, "skill %q has no expertise_keys", s.DisplayText)
		assert.LessOrEqual(t, len(s.ExpertiseKeys), 3, "skill %q has > 3 expertise_keys", s.DisplayText)
	}
}

func TestCuratedSkills_NoDuplicateExpertiseKeysPerSkill(t *testing.T) {
	for _, s := range CuratedSkills {
		seen := make(map[string]bool)
		for _, key := range s.ExpertiseKeys {
			if seen[key] {
				t.Errorf("skill %q has duplicate expertise key %q", s.DisplayText, key)
			}
			seen[key] = true
		}
	}
}

func TestCuratedSkills_MinimumCount(t *testing.T) {
	require.GreaterOrEqual(t, len(CuratedSkills), 400,
		"expected at least 400 curated skills, got %d", len(CuratedSkills))
}

func TestCuratedSkills_DistributionPerExpertise(t *testing.T) {
	counts := make(map[string]int)
	for _, s := range CuratedSkills {
		for _, key := range s.ExpertiseKeys {
			counts[key]++
		}
	}
	for _, key := range expertise.All {
		assert.GreaterOrEqual(t, counts[key], 15,
			"expertise domain %q has only %d skills (minimum 15)", key, counts[key])
	}
}
