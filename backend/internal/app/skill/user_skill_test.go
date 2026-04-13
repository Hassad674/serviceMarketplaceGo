package skill

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainskill "marketplace-backend/internal/domain/skill"
)

// This file holds the CreateUserSkill test suite and the helper-
// function coverage tests. Split out of service_test.go to keep
// every test file under the 600-line guideline.

func TestService_CreateUserSkill_InvalidDisplayText(t *testing.T) {
	svc := newTestService(nil, nil, nil)

	got, err := svc.CreateUserSkill(context.Background(), CreateUserSkillInput{
		DisplayText:   "   ",
		ExpertiseKeys: []string{validExpertiseKey},
	})

	assert.Nil(t, got)
	require.Error(t, err)
	// NewCatalogEntry normalizes whitespace-only to empty and returns
	// ErrInvalidSkillText first (normalization runs before display
	// trim). Both sentinels are acceptable wrapped errors — the test
	// asserts on whichever the domain constructor surfaces today.
	assert.True(t,
		errors.Is(err, domainskill.ErrInvalidSkillText) ||
			errors.Is(err, domainskill.ErrInvalidDisplayText),
		"expected an invalid-text sentinel, got %v", err)
}

func TestService_CreateUserSkill_InvalidExpertiseKeysFilteredOut(t *testing.T) {
	var upserted *domainskill.CatalogEntry
	catalog := &mockSkillCatalog{
		FindByTextFn: func(_ context.Context, _ string) (*domainskill.CatalogEntry, error) {
			if upserted == nil {
				return nil, nil
			}
			return upserted, nil
		},
		UpsertFn: func(_ context.Context, entry *domainskill.CatalogEntry) error {
			upserted = entry
			return nil
		},
	}
	svc := newTestService(catalog, nil, nil)

	got, err := svc.CreateUserSkill(context.Background(), CreateUserSkillInput{
		DisplayText: "Svelte",
		ExpertiseKeys: []string{
			validExpertiseKey,
			unknownExpertiseKey, // dropped silently
			"",                  // dropped silently
		},
	})

	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "svelte", got.SkillText)
	assert.Equal(t, "Svelte", got.DisplayText)
	assert.Equal(t, []string{validExpertiseKey}, got.ExpertiseKeys,
		"only the one valid key should survive the filter")
	assert.False(t, got.IsCurated)
}

func TestService_CreateUserSkill_ExistingReturnedAsIs(t *testing.T) {
	existing := &domainskill.CatalogEntry{
		SkillText:   "react",
		DisplayText: "React", // curated casing preserved
		IsCurated:   true,
	}
	var upsertCalled bool
	catalog := &mockSkillCatalog{
		FindByTextFn: func(_ context.Context, text string) (*domainskill.CatalogEntry, error) {
			assert.Equal(t, "react", text)
			return existing, nil
		},
		UpsertFn: func(_ context.Context, _ *domainskill.CatalogEntry) error {
			upsertCalled = true
			return nil
		},
	}
	svc := newTestService(catalog, nil, nil)

	got, err := svc.CreateUserSkill(context.Background(), CreateUserSkillInput{
		DisplayText:   "REACT", // user typed it in caps
		ExpertiseKeys: []string{validExpertiseKey},
	})

	require.NoError(t, err)
	assert.Same(t, existing, got, "existing entry must be returned untouched")
	assert.False(t, upsertCalled, "Upsert must not be called when the skill already exists")
}

func TestService_CreateUserSkill_NewSkillUpsertedAndRefetched(t *testing.T) {
	var (
		upserted    *domainskill.CatalogEntry
		findCallNum int
	)
	refetched := &domainskill.CatalogEntry{SkillText: "rust"}
	catalog := &mockSkillCatalog{
		FindByTextFn: func(_ context.Context, _ string) (*domainskill.CatalogEntry, error) {
			findCallNum++
			if findCallNum == 1 {
				return nil, nil // first lookup: does not exist
			}
			return refetched, nil // second lookup: post-upsert refetch
		},
		UpsertFn: func(_ context.Context, entry *domainskill.CatalogEntry) error {
			upserted = entry
			return nil
		},
	}
	svc := newTestService(catalog, nil, nil)

	got, err := svc.CreateUserSkill(context.Background(), CreateUserSkillInput{
		DisplayText:   "Rust",
		ExpertiseKeys: []string{validExpertiseKey},
	})

	require.NoError(t, err)
	assert.Same(t, refetched, got)
	require.NotNil(t, upserted)
	assert.Equal(t, "rust", upserted.SkillText)
	assert.Equal(t, "Rust", upserted.DisplayText)
	assert.False(t, upserted.IsCurated)
	assert.Equal(t, 2, findCallNum, "service must find-then-upsert-then-refetch")
}

func TestService_CreateUserSkill_AllInvalidExpertiseKeysStillCreates(t *testing.T) {
	var upserted *domainskill.CatalogEntry
	catalog := &mockSkillCatalog{
		FindByTextFn: func(_ context.Context, _ string) (*domainskill.CatalogEntry, error) {
			if upserted == nil {
				return nil, nil
			}
			return upserted, nil
		},
		UpsertFn: func(_ context.Context, entry *domainskill.CatalogEntry) error {
			upserted = entry
			return nil
		},
	}
	svc := newTestService(catalog, nil, nil)

	got, err := svc.CreateUserSkill(context.Background(), CreateUserSkillInput{
		DisplayText:   "Haskell",
		ExpertiseKeys: []string{unknownExpertiseKey, "also-bogus", ""},
	})

	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "haskell", got.SkillText)
	assert.Empty(t, got.ExpertiseKeys, "all keys were filtered out, result must have none")
}

func TestService_CreateUserSkill_FindByTextError(t *testing.T) {
	boom := errors.New("catalog unreachable")
	catalog := &mockSkillCatalog{
		FindByTextFn: func(_ context.Context, _ string) (*domainskill.CatalogEntry, error) {
			return nil, boom
		},
	}
	svc := newTestService(catalog, nil, nil)

	got, err := svc.CreateUserSkill(context.Background(), CreateUserSkillInput{
		DisplayText: "Elixir",
	})

	assert.Nil(t, got)
	require.Error(t, err)
	assert.ErrorIs(t, err, boom)
}

func TestService_CreateUserSkill_UpsertError(t *testing.T) {
	boom := errors.New("write failed")
	catalog := &mockSkillCatalog{
		FindByTextFn: func(_ context.Context, _ string) (*domainskill.CatalogEntry, error) {
			return nil, nil
		},
		UpsertFn: func(_ context.Context, _ *domainskill.CatalogEntry) error {
			return boom
		},
	}
	svc := newTestService(catalog, nil, nil)

	got, err := svc.CreateUserSkill(context.Background(), CreateUserSkillInput{
		DisplayText: "Elixir",
	})

	assert.Nil(t, got)
	require.Error(t, err)
	assert.ErrorIs(t, err, boom)
}

func TestService_CreateUserSkill_RefetchError(t *testing.T) {
	boom := errors.New("refetch exploded")
	var findCalls int
	catalog := &mockSkillCatalog{
		FindByTextFn: func(_ context.Context, _ string) (*domainskill.CatalogEntry, error) {
			findCalls++
			if findCalls == 1 {
				return nil, nil
			}
			return nil, boom
		},
		UpsertFn: func(_ context.Context, _ *domainskill.CatalogEntry) error {
			return nil
		},
	}
	svc := newTestService(catalog, nil, nil)

	got, err := svc.CreateUserSkill(context.Background(), CreateUserSkillInput{
		DisplayText: "Elixir",
	})

	assert.Nil(t, got)
	require.Error(t, err)
	assert.ErrorIs(t, err, boom)
}

// ---- Helpers ----

func TestClampLimit(t *testing.T) {
	tests := []struct {
		name               string
		limit, def, lo, hi int
		want               int
	}{
		{"zero returns default", 0, 20, 1, 50, 20},
		{"negative returns default", -10, 20, 1, 50, 20},
		{"below min clamps up", 1, 20, 5, 50, 5},
		{"above max clamps down", 999, 20, 1, 50, 50},
		{"within range passthrough", 15, 20, 1, 50, 15},
		{"exact min allowed", 5, 20, 5, 50, 5},
		{"exact max allowed", 50, 20, 1, 50, 50},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := clampLimit(tt.limit, tt.def, tt.lo, tt.hi)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNormalizeAndDedupe(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{"nil input returns empty", nil, []string{}},
		{"empty input returns empty", []string{}, []string{}},
		{
			"drops empty and whitespace-only",
			[]string{"react", "", "   ", "\t"},
			[]string{"react"},
		},
		{
			"normalizes casing and whitespace",
			[]string{"  REACT  ", "GoLang", "Next.js"},
			[]string{"react", "golang", "next.js"},
		},
		{
			"drops duplicates preserving first occurrence",
			[]string{"React", "Go", "react", "GO", "Rust"},
			[]string{"react", "go", "rust"},
		},
		{
			"duplicates with different whitespace collapse to one",
			[]string{"React", "  react ", "REACT"},
			[]string{"react"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeAndDedupe(tt.in)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFilterValidExpertiseKeys(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{"nil returns empty", nil, []string{}},
		{"all invalid returns empty", []string{"nope", "also-nope", ""}, []string{}},
		{"keeps only valid keys", []string{validExpertiseKey, "nope", validExpertiseKey}, []string{validExpertiseKey}},
		{"dedup preserves order", []string{validExpertiseKey, "design_ui_ux", validExpertiseKey}, []string{validExpertiseKey, "design_ui_ux"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterValidExpertiseKeys(tt.in)
			assert.Equal(t, tt.want, got)
		})
	}
}
