package rules

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig_MatchesSpec(t *testing.T) {
	cfg := DefaultConfig()
	assert.InDelta(t, 0.006, cfg.NoiseCoefficient, 1e-9, "NoiseCoefficient default mismatch with ranking-v1 §6.1")
	assert.InDelta(t, 0.3, cfg.NoiseTop3Multiplier, 1e-9)
	assert.InDelta(t, 0.8, cfg.NoiseMidMultiplier, 1e-9)
	assert.InDelta(t, 1.5, cfg.NoiseTailMultiplier, 1e-9)
	assert.Equal(t, 60, cfg.RisingTalentMaxAge)
	assert.Equal(t, 5, cfg.RisingTalentSlotEvery)
	assert.InDelta(t, 5.0, cfg.RisingTalentDelta, 1e-9)
	assert.InDelta(t, 0.0, cfg.FeaturedBoost, 1e-9)
	assert.False(t, cfg.FeaturedEnabled)
	assert.Equal(t, 20, cfg.TopN)
	assert.Equal(t, int64(0), cfg.RandSeed)
}

func TestLoadConfigFromEnv_Overrides(t *testing.T) {
	t.Setenv("RANKING_NOISE_COEFFICIENT", "0.02")
	t.Setenv("RANKING_RISING_TALENT_MAX_AGE", "30")
	t.Setenv("RANKING_FEATURED_ENABLED", "true")
	t.Setenv("RANKING_FEATURED_BOOST", "0.15")
	t.Setenv("RANKING_RULES_TOP_N", "10")
	t.Setenv("RANKING_RULES_SEED", "42")

	cfg, err := LoadConfigFromEnv()
	require.NoError(t, err)
	assert.InDelta(t, 0.02, cfg.NoiseCoefficient, 1e-9)
	assert.Equal(t, 30, cfg.RisingTalentMaxAge)
	assert.True(t, cfg.FeaturedEnabled)
	assert.InDelta(t, 0.15, cfg.FeaturedBoost, 1e-9)
	assert.Equal(t, 10, cfg.TopN)
	assert.Equal(t, int64(42), cfg.RandSeed)
}

func TestLoadConfigFromEnv_MalformedRejected(t *testing.T) {
	cases := []struct {
		name    string
		envKey  string
		envVal  string
		wantErr string
	}{
		{"bad float", "RANKING_NOISE_COEFFICIENT", "not-a-float", "RANKING_NOISE_COEFFICIENT"},
		{"bad int", "RANKING_RISING_TALENT_MAX_AGE", "abc", "RANKING_RISING_TALENT_MAX_AGE"},
		{"bad bool", "RANKING_FEATURED_ENABLED", "maybe", "RANKING_FEATURED_ENABLED"},
		{"bad seed", "RANKING_RULES_SEED", "xx", "RANKING_RULES_SEED"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(tc.envKey, tc.envVal)
			_, err := LoadConfigFromEnv()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(c *Config)
		wantErr string
	}{
		{
			name:    "negative noise",
			mutate:  func(c *Config) { c.NoiseCoefficient = -1 },
			wantErr: "NoiseCoefficient",
		},
		{
			name:    "zero slot every",
			mutate:  func(c *Config) { c.RisingTalentSlotEvery = 0 },
			wantErr: "RisingTalentSlotEvery",
		},
		{
			name:    "negative max age",
			mutate:  func(c *Config) { c.RisingTalentMaxAge = -10 },
			wantErr: "RisingTalentMaxAge",
		},
		{
			name:    "zero TopN",
			mutate:  func(c *Config) { c.TopN = 0 },
			wantErr: "TopN",
		},
		{
			name:    "negative featured boost",
			mutate:  func(c *Config) { c.FeaturedBoost = -0.5 },
			wantErr: "FeaturedBoost",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := DefaultConfig()
			tc.mutate(&cfg)
			err := cfg.validate()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
		})
	}
}

func TestConfig_Validate_DefaultsPass(t *testing.T) {
	assert.NoError(t, DefaultConfig().validate())
}

func TestLoadConfigFromEnv_AllKnobs(t *testing.T) {
	t.Setenv("RANKING_NOISE_COEFFICIENT", "0.007")
	t.Setenv("RANKING_NOISE_TOP3_MULTIPLIER", "0.4")
	t.Setenv("RANKING_NOISE_MID_MULTIPLIER", "0.9")
	t.Setenv("RANKING_NOISE_TAIL_MULTIPLIER", "1.6")
	t.Setenv("RANKING_RISING_TALENT_MAX_AGE", "45")
	t.Setenv("RANKING_RISING_TALENT_SLOT_EVERY", "4")
	t.Setenv("RANKING_RISING_TALENT_DELTA", "3.0")
	t.Setenv("RANKING_FEATURED_BOOST", "0.10")
	t.Setenv("RANKING_FEATURED_ENABLED", "1")
	t.Setenv("RANKING_RULES_TOP_N", "15")
	t.Setenv("RANKING_RULES_SEED", "123")

	cfg, err := LoadConfigFromEnv()
	require.NoError(t, err)
	assert.InDelta(t, 0.007, cfg.NoiseCoefficient, 1e-9)
	assert.InDelta(t, 0.4, cfg.NoiseTop3Multiplier, 1e-9)
	assert.InDelta(t, 0.9, cfg.NoiseMidMultiplier, 1e-9)
	assert.InDelta(t, 1.6, cfg.NoiseTailMultiplier, 1e-9)
	assert.Equal(t, 45, cfg.RisingTalentMaxAge)
	assert.Equal(t, 4, cfg.RisingTalentSlotEvery)
	assert.InDelta(t, 3.0, cfg.RisingTalentDelta, 1e-9)
	assert.InDelta(t, 0.10, cfg.FeaturedBoost, 1e-9)
	assert.True(t, cfg.FeaturedEnabled)
	assert.Equal(t, 15, cfg.TopN)
	assert.Equal(t, int64(123), cfg.RandSeed)
}

func TestLoadConfigFromEnv_MalformedBranches(t *testing.T) {
	// Each malformed var must be rejected by LoadConfigFromEnv. This
	// exercises the error path on every read* helper branch.
	cases := []struct {
		envKey, envVal string
	}{
		{"RANKING_NOISE_TOP3_MULTIPLIER", "x"},
		{"RANKING_NOISE_MID_MULTIPLIER", "x"},
		{"RANKING_NOISE_TAIL_MULTIPLIER", "x"},
		{"RANKING_RISING_TALENT_SLOT_EVERY", "x"},
		{"RANKING_RISING_TALENT_DELTA", "x"},
		{"RANKING_FEATURED_BOOST", "x"},
		{"RANKING_RULES_TOP_N", "x"},
	}
	for _, tc := range cases {
		t.Run(tc.envKey, func(t *testing.T) {
			t.Setenv(tc.envKey, tc.envVal)
			_, err := LoadConfigFromEnv()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.envKey)
		})
	}
}

func TestLoadConfigFromEnv_EmptyStringsIgnored(t *testing.T) {
	t.Setenv("RANKING_NOISE_COEFFICIENT", "")
	t.Setenv("RANKING_RISING_TALENT_MAX_AGE", "")
	cfg, err := LoadConfigFromEnv()
	require.NoError(t, err)
	assert.InDelta(t, DefaultConfig().NoiseCoefficient, cfg.NoiseCoefficient, 1e-9,
		"empty env var must keep the default")
}
