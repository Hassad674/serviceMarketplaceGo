package skill

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainskill "marketplace-backend/internal/domain/skill"
)

// This file holds the catalog-read test suite (GetCuratedForExpertise,
// CountCuratedForExpertise, Autocomplete, GetProfileSkills) plus the
// shared test helpers used by the sibling test files
// (profile_skills_test.go and user_skill_test.go).

// validExpertiseKey is a canonical key from the frozen expertise
// catalog. Using "development" here (rather than a random string)
// avoids the test file having to import the expertise package just
// to pick a constant — the service already validates against the
// real catalog, so a known-good key keeps the tests readable.
const validExpertiseKey = "development"

// unknownExpertiseKey is deliberately not in the frozen catalog so
// every validation branch can be exercised with a stable value.
const unknownExpertiseKey = "not-a-real-expertise-domain"

// newTestService wires a Service with the given mocks, defaulting any
// missing mock to a no-op instance that panics on unexpected calls.
// Every test that only cares about one dependency can pass nil for
// the others.
func newTestService(
	catalog *mockSkillCatalog,
	profiles *mockProfileSkill,
	orgs *mockOrgTypeResolver,
) *Service {
	if catalog == nil {
		catalog = &mockSkillCatalog{}
	}
	if profiles == nil {
		profiles = &mockProfileSkill{}
	}
	if orgs == nil {
		orgs = &mockOrgTypeResolver{}
	}
	return NewService(catalog, profiles, orgs)
}

// ---- GetCuratedForExpertise ----

func TestService_GetCuratedForExpertise_ValidKey(t *testing.T) {
	want := []*domainskill.CatalogEntry{{SkillText: "react", DisplayText: "React"}}
	catalog := &mockSkillCatalog{
		ListCuratedByExpertiseFn: func(_ context.Context, key string, limit int) ([]*domainskill.CatalogEntry, error) {
			assert.Equal(t, validExpertiseKey, key)
			assert.Equal(t, 50, limit)
			return want, nil
		},
	}
	svc := newTestService(catalog, nil, nil)

	got, err := svc.GetCuratedForExpertise(context.Background(), validExpertiseKey, 0)

	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestService_GetCuratedForExpertise_InvalidKey(t *testing.T) {
	// Catalog mock has no Fn set: a call would panic, which proves
	// the service short-circuits before reaching the repository.
	svc := newTestService(nil, nil, nil)

	got, err := svc.GetCuratedForExpertise(context.Background(), unknownExpertiseKey, 50)

	assert.Nil(t, got)
	assert.ErrorIs(t, err, domainskill.ErrInvalidExpertiseKey)
}

func TestService_GetCuratedForExpertise_LimitClamping(t *testing.T) {
	tests := []struct {
		name       string
		inputLimit int
		wantLimit  int
	}{
		{"zero defaults to 50", 0, 50},
		{"negative defaults to 50", -5, 50},
		{"one allowed as-is", 1, 1},
		{"above 100 clamped to 100", 500, 100},
		{"exactly 100 allowed", 100, 100},
		{"mid-range passthrough", 42, 42},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotLimit := -1
			catalog := &mockSkillCatalog{
				ListCuratedByExpertiseFn: func(_ context.Context, _ string, limit int) ([]*domainskill.CatalogEntry, error) {
					gotLimit = limit
					return nil, nil
				},
			}
			svc := newTestService(catalog, nil, nil)

			_, err := svc.GetCuratedForExpertise(context.Background(), validExpertiseKey, tt.inputLimit)

			require.NoError(t, err)
			assert.Equal(t, tt.wantLimit, gotLimit)
		})
	}
}

// ---- CountCuratedForExpertise ----

func TestService_CountCuratedForExpertise_ValidKey(t *testing.T) {
	catalog := &mockSkillCatalog{
		CountCuratedByExpertiseFn: func(_ context.Context, key string) (int, error) {
			assert.Equal(t, validExpertiseKey, key)
			return 142, nil
		},
	}
	svc := newTestService(catalog, nil, nil)

	got, err := svc.CountCuratedForExpertise(context.Background(), validExpertiseKey)

	require.NoError(t, err)
	assert.Equal(t, 142, got)
}

func TestService_CountCuratedForExpertise_InvalidKey(t *testing.T) {
	svc := newTestService(nil, nil, nil)

	got, err := svc.CountCuratedForExpertise(context.Background(), unknownExpertiseKey)

	assert.Equal(t, 0, got)
	assert.ErrorIs(t, err, domainskill.ErrInvalidExpertiseKey)
}

// ---- Autocomplete ----

func TestService_Autocomplete_EmptyQueryReturnsNil(t *testing.T) {
	svc := newTestService(nil, nil, nil)

	got, err := svc.Autocomplete(context.Background(), "", 20)

	assert.NoError(t, err)
	assert.Nil(t, got)
}

func TestService_Autocomplete_WhitespaceOnlyQueryReturnsNil(t *testing.T) {
	svc := newTestService(nil, nil, nil)

	got, err := svc.Autocomplete(context.Background(), "   \t\n ", 20)

	assert.NoError(t, err)
	assert.Nil(t, got)
}

func TestService_Autocomplete_NormalizesAndDelegates(t *testing.T) {
	want := []*domainskill.CatalogEntry{{SkillText: "react native"}}
	var (
		gotQuery string
		gotLimit int
	)
	catalog := &mockSkillCatalog{
		SearchAutocompleteFn: func(_ context.Context, q string, limit int) ([]*domainskill.CatalogEntry, error) {
			gotQuery = q
			gotLimit = limit
			return want, nil
		},
	}
	svc := newTestService(catalog, nil, nil)

	got, err := svc.Autocomplete(context.Background(), "  REACT   Native  ", 0)

	require.NoError(t, err)
	assert.Equal(t, want, got)
	assert.Equal(t, "react native", gotQuery, "query should be lowercased + collapsed")
	assert.Equal(t, 20, gotLimit, "limit 0 should default to 20")
}

func TestService_Autocomplete_UppercaseLowercased(t *testing.T) {
	var gotQuery string
	catalog := &mockSkillCatalog{
		SearchAutocompleteFn: func(_ context.Context, q string, _ int) ([]*domainskill.CatalogEntry, error) {
			gotQuery = q
			return nil, nil
		},
	}
	svc := newTestService(catalog, nil, nil)

	_, err := svc.Autocomplete(context.Background(), "GoLang", 10)

	require.NoError(t, err)
	assert.Equal(t, "golang", gotQuery)
}

func TestService_Autocomplete_LimitClamping(t *testing.T) {
	tests := []struct {
		name       string
		inputLimit int
		wantLimit  int
	}{
		{"zero defaults to 20", 0, 20},
		{"negative defaults to 20", -1, 20},
		{"above 50 clamped to 50", 500, 50},
		{"exactly 50 allowed", 50, 50},
		{"mid-range passthrough", 15, 15},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotLimit int
			catalog := &mockSkillCatalog{
				SearchAutocompleteFn: func(_ context.Context, _ string, limit int) ([]*domainskill.CatalogEntry, error) {
					gotLimit = limit
					return nil, nil
				},
			}
			svc := newTestService(catalog, nil, nil)

			_, err := svc.Autocomplete(context.Background(), "react", tt.inputLimit)

			require.NoError(t, err)
			assert.Equal(t, tt.wantLimit, gotLimit)
		})
	}
}

// ---- GetProfileSkills ----

func TestService_GetProfileSkills_DelegatesToRepo(t *testing.T) {
	orgID := uuid.New()
	want := []*domainskill.ProfileSkill{
		{OrganizationID: orgID, SkillText: "react", Position: 0},
		{OrganizationID: orgID, SkillText: "go", Position: 1},
	}
	profiles := &mockProfileSkill{
		ListByOrgIDFn: func(_ context.Context, id uuid.UUID) ([]*domainskill.ProfileSkill, error) {
			assert.Equal(t, orgID, id)
			return want, nil
		},
	}
	svc := newTestService(nil, profiles, nil)

	got, err := svc.GetProfileSkills(context.Background(), orgID)

	require.NoError(t, err)
	assert.Equal(t, want, got)
}
