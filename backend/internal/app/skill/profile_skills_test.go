package skill

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainskill "marketplace-backend/internal/domain/skill"
)

// This file holds the ReplaceProfileSkills test suite. It is split
// out of service_test.go to keep every test file under the 600-line
// backend guideline. The ReplaceProfileSkills path has the highest
// branching factor of the service (org-type resolution, feature
// toggle, limit enforcement, catalog lookup, persistence) so its
// tests dominate the suite.

// makeUniqueTexts generates n unique two-letter-suffixed strings,
// handy for exercising the per-org-type limits without having to
// invent 40 different skill names by hand.
func makeUniqueTexts(n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = "skill-" + string(rune('a'+i%26)) + string(rune('a'+(i/26)%26))
	}
	return out
}

func TestService_ReplaceProfileSkills_OrgTypeResolverError(t *testing.T) {
	orgID := uuid.New()
	boom := errors.New("org lookup failed")
	orgs := &mockOrgTypeResolver{
		GetOrgTypeFn: func(_ context.Context, _ uuid.UUID) (string, error) {
			return "", boom
		},
	}
	svc := newTestService(nil, nil, orgs)

	err := svc.ReplaceProfileSkills(context.Background(), ReplaceProfileSkillsInput{
		OrganizationID: orgID,
		SkillTexts:     []string{"react"},
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, boom)
}

func TestService_ReplaceProfileSkills_EnterpriseDisabled(t *testing.T) {
	orgs := &mockOrgTypeResolver{
		GetOrgTypeFn: func(_ context.Context, _ uuid.UUID) (string, error) {
			return domainskill.OrgTypeEnterprise, nil
		},
	}
	svc := newTestService(nil, nil, orgs)

	err := svc.ReplaceProfileSkills(context.Background(), ReplaceProfileSkillsInput{
		OrganizationID: uuid.New(),
		SkillTexts:     []string{"react"},
	})

	assert.ErrorIs(t, err, domainskill.ErrSkillsDisabledForOrgType)
}

func TestService_ReplaceProfileSkills_UnknownOrgTypeDisabled(t *testing.T) {
	// An unknown org type falls through the limits switch and yields 0,
	// which IsSkillsFeatureEnabled treats as "disabled".
	orgs := &mockOrgTypeResolver{
		GetOrgTypeFn: func(_ context.Context, _ uuid.UUID) (string, error) {
			return "some_future_type", nil
		},
	}
	svc := newTestService(nil, nil, orgs)

	err := svc.ReplaceProfileSkills(context.Background(), ReplaceProfileSkillsInput{
		OrganizationID: uuid.New(),
		SkillTexts:     []string{"react"},
	})

	assert.ErrorIs(t, err, domainskill.ErrSkillsDisabledForOrgType)
}

func TestService_ReplaceProfileSkills_AgencyUpTo40Allowed(t *testing.T) {
	unique := makeUniqueTexts(40)
	// Sanity check: the fixture really produces 40 distinct entries.
	seen := map[string]struct{}{}
	for _, u := range unique {
		seen[u] = struct{}{}
	}
	require.Equal(t, 40, len(seen), "test fixture must produce 40 unique skills")

	orgs := &mockOrgTypeResolver{
		GetOrgTypeFn: func(_ context.Context, _ uuid.UUID) (string, error) {
			return domainskill.OrgTypeAgency, nil
		},
	}
	catalog := &mockSkillCatalog{
		FindByTextFn: func(_ context.Context, text string) (*domainskill.CatalogEntry, error) {
			return &domainskill.CatalogEntry{SkillText: text, DisplayText: text}, nil
		},
	}
	var persisted []*domainskill.ProfileSkill
	profiles := &mockProfileSkill{
		ReplaceForOrgFn: func(_ context.Context, _ uuid.UUID, skills []*domainskill.ProfileSkill) error {
			persisted = skills
			return nil
		},
	}
	withReplaceDefaults(profiles, catalog)
	svc := newTestService(catalog, profiles, orgs)

	err := svc.ReplaceProfileSkills(context.Background(), ReplaceProfileSkillsInput{
		OrganizationID: uuid.New(),
		SkillTexts:     unique,
	})

	require.NoError(t, err)
	assert.Len(t, persisted, 40)
}

func TestService_ReplaceProfileSkills_AgencyOver40Rejected(t *testing.T) {
	orgs := &mockOrgTypeResolver{
		GetOrgTypeFn: func(_ context.Context, _ uuid.UUID) (string, error) {
			return domainskill.OrgTypeAgency, nil
		},
	}
	// Catalog and profiles mocks have no Fn set: any call would panic,
	// proving the service rejects BEFORE touching either repository.
	svc := newTestService(nil, nil, orgs)

	err := svc.ReplaceProfileSkills(context.Background(), ReplaceProfileSkillsInput{
		OrganizationID: uuid.New(),
		SkillTexts:     makeUniqueTexts(41),
	})

	assert.ErrorIs(t, err, domainskill.ErrTooManySkills)
}

func TestService_ReplaceProfileSkills_ProviderUpTo25Allowed(t *testing.T) {
	orgs := &mockOrgTypeResolver{
		GetOrgTypeFn: func(_ context.Context, _ uuid.UUID) (string, error) {
			return domainskill.OrgTypeProviderPersonal, nil
		},
	}
	catalog := &mockSkillCatalog{
		FindByTextFn: func(_ context.Context, text string) (*domainskill.CatalogEntry, error) {
			return &domainskill.CatalogEntry{SkillText: text, DisplayText: text}, nil
		},
	}
	var persisted []*domainskill.ProfileSkill
	profiles := &mockProfileSkill{
		ReplaceForOrgFn: func(_ context.Context, _ uuid.UUID, skills []*domainskill.ProfileSkill) error {
			persisted = skills
			return nil
		},
	}
	withReplaceDefaults(profiles, catalog)
	svc := newTestService(catalog, profiles, orgs)

	err := svc.ReplaceProfileSkills(context.Background(), ReplaceProfileSkillsInput{
		OrganizationID: uuid.New(),
		SkillTexts:     makeUniqueTexts(25),
	})

	require.NoError(t, err)
	assert.Len(t, persisted, 25)
}

func TestService_ReplaceProfileSkills_ProviderOver25Rejected(t *testing.T) {
	orgs := &mockOrgTypeResolver{
		GetOrgTypeFn: func(_ context.Context, _ uuid.UUID) (string, error) {
			return domainskill.OrgTypeProviderPersonal, nil
		},
	}
	svc := newTestService(nil, nil, orgs)

	err := svc.ReplaceProfileSkills(context.Background(), ReplaceProfileSkillsInput{
		OrganizationID: uuid.New(),
		SkillTexts:     makeUniqueTexts(26),
	})

	assert.ErrorIs(t, err, domainskill.ErrTooManySkills)
}

func TestService_ReplaceProfileSkills_DuplicatesDedupedThenCounted(t *testing.T) {
	// 41 raw entries but only 2 unique after normalization — the limit
	// check runs on the post-dedupe list, so this must succeed on an
	// agency org (agency max = 40).
	raw := make([]string, 0, 42)
	for i := 0; i < 20; i++ {
		raw = append(raw, "React")
	}
	for i := 0; i < 21; i++ {
		raw = append(raw, "  REACT  ") // normalizes to "react"
	}
	raw = append(raw, "Go")

	orgs := &mockOrgTypeResolver{
		GetOrgTypeFn: func(_ context.Context, _ uuid.UUID) (string, error) {
			return domainskill.OrgTypeAgency, nil
		},
	}
	catalog := &mockSkillCatalog{
		FindByTextFn: func(_ context.Context, text string) (*domainskill.CatalogEntry, error) {
			return &domainskill.CatalogEntry{SkillText: text, DisplayText: text}, nil
		},
	}
	var persisted []*domainskill.ProfileSkill
	profiles := &mockProfileSkill{
		ReplaceForOrgFn: func(_ context.Context, _ uuid.UUID, skills []*domainskill.ProfileSkill) error {
			persisted = skills
			return nil
		},
	}
	withReplaceDefaults(profiles, catalog)
	svc := newTestService(catalog, profiles, orgs)

	err := svc.ReplaceProfileSkills(context.Background(), ReplaceProfileSkillsInput{
		OrganizationID: uuid.New(),
		SkillTexts:     raw,
	})

	require.NoError(t, err)
	require.Len(t, persisted, 2, "duplicates must be collapsed to a single entry")
	assert.Equal(t, "react", persisted[0].SkillText)
	assert.Equal(t, 0, persisted[0].Position)
	assert.Equal(t, "go", persisted[1].SkillText)
	assert.Equal(t, 1, persisted[1].Position)
}

func TestService_ReplaceProfileSkills_UnknownSkillReturnsWrappedError(t *testing.T) {
	orgs := &mockOrgTypeResolver{
		GetOrgTypeFn: func(_ context.Context, _ uuid.UUID) (string, error) {
			return domainskill.OrgTypeAgency, nil
		},
	}
	catalog := &mockSkillCatalog{
		FindByTextFn: func(_ context.Context, text string) (*domainskill.CatalogEntry, error) {
			if text == "react" {
				return &domainskill.CatalogEntry{SkillText: "react"}, nil
			}
			return nil, nil // not found
		},
	}
	// No Fn on profiles: a ReplaceForOrg call would panic. Proves the
	// service rejects before persistence.
	svc := newTestService(catalog, nil, orgs)

	err := svc.ReplaceProfileSkills(context.Background(), ReplaceProfileSkillsInput{
		OrganizationID: uuid.New(),
		SkillTexts:     []string{"React", "MadeUpSkill"},
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, domainskill.ErrSkillNotFound)
	assert.Contains(t, err.Error(), `"madeupskill"`, "wrapped error should surface the offending text")
}

func TestService_ReplaceProfileSkills_FindByTextRepoError(t *testing.T) {
	orgs := &mockOrgTypeResolver{
		GetOrgTypeFn: func(_ context.Context, _ uuid.UUID) (string, error) {
			return domainskill.OrgTypeAgency, nil
		},
	}
	boom := errors.New("database exploded")
	catalog := &mockSkillCatalog{
		FindByTextFn: func(_ context.Context, _ string) (*domainskill.CatalogEntry, error) {
			return nil, boom
		},
	}
	svc := newTestService(catalog, nil, orgs)

	err := svc.ReplaceProfileSkills(context.Background(), ReplaceProfileSkillsInput{
		OrganizationID: uuid.New(),
		SkillTexts:     []string{"react"},
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, boom)
}

func TestService_ReplaceProfileSkills_ValidPayloadAssignsContiguousPositions(t *testing.T) {
	orgID := uuid.New()
	orgs := &mockOrgTypeResolver{
		GetOrgTypeFn: func(_ context.Context, _ uuid.UUID) (string, error) {
			return domainskill.OrgTypeAgency, nil
		},
	}
	catalog := &mockSkillCatalog{
		FindByTextFn: func(_ context.Context, text string) (*domainskill.CatalogEntry, error) {
			return &domainskill.CatalogEntry{SkillText: text, DisplayText: text}, nil
		},
	}
	var (
		persisted      []*domainskill.ProfileSkill
		persistedOrgID uuid.UUID
	)
	profiles := &mockProfileSkill{
		ReplaceForOrgFn: func(_ context.Context, id uuid.UUID, skills []*domainskill.ProfileSkill) error {
			persistedOrgID = id
			persisted = skills
			return nil
		},
	}
	withReplaceDefaults(profiles, catalog)
	svc := newTestService(catalog, profiles, orgs)

	err := svc.ReplaceProfileSkills(context.Background(), ReplaceProfileSkillsInput{
		OrganizationID: orgID,
		SkillTexts:     []string{"React", "Go", "TypeScript"},
	})

	require.NoError(t, err)
	assert.Equal(t, orgID, persistedOrgID)
	require.Len(t, persisted, 3)
	assert.Equal(t, "react", persisted[0].SkillText)
	assert.Equal(t, 0, persisted[0].Position)
	assert.Equal(t, orgID, persisted[0].OrganizationID)
	assert.Equal(t, "go", persisted[1].SkillText)
	assert.Equal(t, 1, persisted[1].Position)
	assert.Equal(t, "typescript", persisted[2].SkillText)
	assert.Equal(t, 2, persisted[2].Position)
}

func TestService_ReplaceProfileSkills_EmptyPayloadClears(t *testing.T) {
	orgs := &mockOrgTypeResolver{
		GetOrgTypeFn: func(_ context.Context, _ uuid.UUID) (string, error) {
			return domainskill.OrgTypeAgency, nil
		},
	}
	// Catalog mock has no Fn set — proves the service does not call
	// FindByText when the payload is empty.
	var (
		called    bool
		persisted []*domainskill.ProfileSkill
	)
	profiles := &mockProfileSkill{
		ReplaceForOrgFn: func(_ context.Context, _ uuid.UUID, skills []*domainskill.ProfileSkill) error {
			called = true
			persisted = skills
			return nil
		},
	}
	withReplaceDefaults(profiles, nil)
	svc := newTestService(nil, profiles, orgs)

	err := svc.ReplaceProfileSkills(context.Background(), ReplaceProfileSkillsInput{
		OrganizationID: uuid.New(),
		SkillTexts:     []string{},
	})

	require.NoError(t, err)
	assert.True(t, called, "ReplaceForOrg must be called even with empty payload")
	assert.Empty(t, persisted)
	assert.NotNil(t, persisted, "empty slice should be non-nil")
}

func TestService_ReplaceProfileSkills_PersistenceErrorWrapped(t *testing.T) {
	orgs := &mockOrgTypeResolver{
		GetOrgTypeFn: func(_ context.Context, _ uuid.UUID) (string, error) {
			return domainskill.OrgTypeAgency, nil
		},
	}
	catalog := &mockSkillCatalog{
		FindByTextFn: func(_ context.Context, text string) (*domainskill.CatalogEntry, error) {
			return &domainskill.CatalogEntry{SkillText: text}, nil
		},
	}
	boom := errors.New("write failed")
	profiles := &mockProfileSkill{
		ReplaceForOrgFn: func(_ context.Context, _ uuid.UUID, _ []*domainskill.ProfileSkill) error {
			return boom
		},
	}
	withReplaceDefaults(profiles, catalog)
	svc := newTestService(catalog, profiles, orgs)

	err := svc.ReplaceProfileSkills(context.Background(), ReplaceProfileSkillsInput{
		OrganizationID: uuid.New(),
		SkillTexts:     []string{"react"},
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, boom)
	assert.Contains(t, err.Error(), "persist")
}

// TestService_ReplaceProfileSkills_UsageCountDiff verifies the
// diff-based usage_count update that runs after a successful
// ReplaceForOrg: newly-added skills are incremented, removed skills
// are decremented, and skills present in both lists are left alone.
func TestService_ReplaceProfileSkills_UsageCountDiff(t *testing.T) {
	orgID := uuid.New()
	orgs := &mockOrgTypeResolver{
		GetOrgTypeFn: func(_ context.Context, _ uuid.UUID) (string, error) {
			return domainskill.OrgTypeAgency, nil
		},
	}

	// Previous profile state: [react, go, typescript]
	// New profile state:      [react, python, docker]
	//
	//   Untouched: react (in both)
	//   Added:     python, docker → +1 each
	//   Removed:   go, typescript → -1 each
	previousSkills := []*domainskill.ProfileSkill{
		{OrganizationID: orgID, SkillText: "react", Position: 0},
		{OrganizationID: orgID, SkillText: "go", Position: 1},
		{OrganizationID: orgID, SkillText: "typescript", Position: 2},
	}

	var (
		incremented []string
		decremented []string
	)
	catalog := &mockSkillCatalog{
		FindByTextFn: func(_ context.Context, text string) (*domainskill.CatalogEntry, error) {
			return &domainskill.CatalogEntry{SkillText: text, DisplayText: text}, nil
		},
		IncrementUsageCountFn: func(_ context.Context, text string) error {
			incremented = append(incremented, text)
			return nil
		},
		DecrementUsageCountFn: func(_ context.Context, text string) error {
			decremented = append(decremented, text)
			return nil
		},
	}
	profiles := &mockProfileSkill{
		ListByOrgIDFn: func(_ context.Context, _ uuid.UUID) ([]*domainskill.ProfileSkill, error) {
			return previousSkills, nil
		},
		ReplaceForOrgFn: func(_ context.Context, _ uuid.UUID, _ []*domainskill.ProfileSkill) error {
			return nil
		},
	}
	svc := newTestService(catalog, profiles, orgs)

	err := svc.ReplaceProfileSkills(context.Background(), ReplaceProfileSkillsInput{
		OrganizationID: orgID,
		SkillTexts:     []string{"React", "Python", "Docker"},
	})
	require.NoError(t, err)

	assert.ElementsMatch(t, []string{"python", "docker"}, incremented,
		"expected python and docker to be incremented (added to profile)")
	assert.ElementsMatch(t, []string{"go", "typescript"}, decremented,
		"expected go and typescript to be decremented (removed from profile)")
	assert.NotContains(t, incremented, "react",
		"react was in both lists — must not be incremented")
	assert.NotContains(t, decremented, "react",
		"react was in both lists — must not be decremented")
}

// TestService_ReplaceProfileSkills_UsageCountErrorsSwallowed verifies
// that individual counter update failures do NOT abort the service
// call — the replacement has already committed, so bailing out would
// leave the caller in an ambiguous state. Stale cache is accepted
// as the intentional V1 trade-off.
func TestService_ReplaceProfileSkills_UsageCountErrorsSwallowed(t *testing.T) {
	orgs := &mockOrgTypeResolver{
		GetOrgTypeFn: func(_ context.Context, _ uuid.UUID) (string, error) {
			return domainskill.OrgTypeAgency, nil
		},
	}
	catalog := &mockSkillCatalog{
		FindByTextFn: func(_ context.Context, text string) (*domainskill.CatalogEntry, error) {
			return &domainskill.CatalogEntry{SkillText: text, DisplayText: text}, nil
		},
		IncrementUsageCountFn: func(_ context.Context, _ string) error {
			return errors.New("redis down")
		},
		DecrementUsageCountFn: func(_ context.Context, _ string) error {
			return errors.New("redis still down")
		},
	}
	profiles := &mockProfileSkill{
		ListByOrgIDFn: func(_ context.Context, _ uuid.UUID) ([]*domainskill.ProfileSkill, error) {
			return []*domainskill.ProfileSkill{{SkillText: "oldskill"}}, nil
		},
		ReplaceForOrgFn: func(_ context.Context, _ uuid.UUID, _ []*domainskill.ProfileSkill) error {
			return nil
		},
	}
	svc := newTestService(catalog, profiles, orgs)

	err := svc.ReplaceProfileSkills(context.Background(), ReplaceProfileSkillsInput{
		OrganizationID: uuid.New(),
		SkillTexts:     []string{"newskill"},
	})
	// Counter failures are swallowed by design — the service returns nil
	// because the transactional replacement has already persisted.
	require.NoError(t, err)
}

// TestService_ReplaceProfileSkills_ListPreviousErrorAbortsBeforeReplace
// verifies that a failure to snapshot the previous skills list aborts
// the service call BEFORE ReplaceForOrg runs, so the profile is never
// mutated when we cannot compute the diff.
func TestService_ReplaceProfileSkills_ListPreviousErrorAbortsBeforeReplace(t *testing.T) {
	orgs := &mockOrgTypeResolver{
		GetOrgTypeFn: func(_ context.Context, _ uuid.UUID) (string, error) {
			return domainskill.OrgTypeAgency, nil
		},
	}
	catalog := &mockSkillCatalog{
		FindByTextFn: func(_ context.Context, text string) (*domainskill.CatalogEntry, error) {
			return &domainskill.CatalogEntry{SkillText: text, DisplayText: text}, nil
		},
	}
	boom := errors.New("list previous failed")
	profiles := &mockProfileSkill{
		ListByOrgIDFn: func(_ context.Context, _ uuid.UUID) ([]*domainskill.ProfileSkill, error) {
			return nil, boom
		},
		// No ReplaceForOrgFn: a call would panic. Proves the service
		// aborts BEFORE the replace when the list snapshot fails.
	}
	svc := newTestService(catalog, profiles, orgs)

	err := svc.ReplaceProfileSkills(context.Background(), ReplaceProfileSkillsInput{
		OrganizationID: uuid.New(),
		SkillTexts:     []string{"react"},
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, boom)
	assert.Contains(t, err.Error(), "list previous")
}
