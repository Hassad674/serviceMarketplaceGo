package search

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/search"
)

// integration_test.go exercises the query service against a real
// Typesense container. Gated by TYPESENSE_INTEGRATION_URL +
// TYPESENSE_INTEGRATION_API_KEY so unit-test runs stay hermetic.
//
// The test seeds a tiny set of documents, hits the live cluster
// through the scoped client, and verifies the parsed result.

func integrationClient(t *testing.T) *search.Client {
	t.Helper()
	host := os.Getenv("TYPESENSE_INTEGRATION_URL")
	apiKey := os.Getenv("TYPESENSE_INTEGRATION_API_KEY")
	if host == "" || apiKey == "" {
		t.Skip("TYPESENSE_INTEGRATION_URL / TYPESENSE_INTEGRATION_API_KEY not set — skipping")
	}
	c, err := search.NewClient(host, apiKey, search.WithHTTPClient(&http.Client{Timeout: 10 * time.Second}))
	require.NoError(t, err)
	return c
}

func TestIntegration_QueryServiceWithRealTypesense(t *testing.T) {
	client := integrationClient(t)
	ctx := context.Background()

	// Make sure the schema + synonyms are in place.
	require.NoError(t, search.EnsureSchema(ctx, search.EnsureSchemaDeps{Client: client}))

	// Index a tiny set of documents under a freelance persona so
	// the scoped client surfaces them. The composite-ID scheme
	// requires organization_id be set — we use the raw UUID as both
	// the org id and the ID prefix for readability.
	orgA := uuid.New()
	orgB := uuid.New()
	docs := []*search.SearchDocument{
		{
			ID:                      orgA.String() + ":freelance",
			OrganizationID:          orgA.String(),
			Persona:                 search.PersonaFreelance,
			IsPublished:             true,
			DisplayName:             "Phase2 Alice",
			Title:                   "Senior Go Developer",
			Skills:                  []string{"go", "react"},
			SkillsText:              "go react",
			LanguagesProfessional:   []string{"fr", "en"},
			LanguagesConversational: []string{},
			ExpertiseDomains:        []string{"dev"},
			AvailabilityStatus:      "available_now",
			AvailabilityPriority:    3,
			RatingAverage:           4.8,
			RatingCount:             12,
			RatingScore:             12.5,
			ProfileCompletionScore:  90,
			WorkMode:                []string{"remote"},
			CreatedAt:               time.Now().Unix(),
			UpdatedAt:               time.Now().Unix(),
		},
		{
			ID:                      orgB.String() + ":freelance",
			OrganizationID:          orgB.String(),
			Persona:                 search.PersonaFreelance,
			IsPublished:             true,
			DisplayName:             "Phase2 Bob",
			Title:                   "UX Designer",
			Skills:                  []string{"figma", "sketch"},
			SkillsText:              "figma sketch",
			LanguagesProfessional:   []string{"en"},
			LanguagesConversational: []string{},
			ExpertiseDomains:        []string{"design"},
			AvailabilityStatus:      "available_soon",
			AvailabilityPriority:    2,
			RatingAverage:           4.2,
			RatingCount:             5,
			RatingScore:             7.5,
			ProfileCompletionScore:  80,
			WorkMode:                []string{"hybrid"},
			CreatedAt:               time.Now().Unix(),
			UpdatedAt:               time.Now().Unix(),
		},
	}
	require.NoError(t, client.BulkUpsert(ctx, search.AliasName, docs))
	defer func() {
		for _, d := range docs {
			_ = client.DeleteDocument(ctx, search.AliasName, d.ID)
		}
	}()

	// Wait briefly for Typesense to make the new docs queryable.
	time.Sleep(150 * time.Millisecond)

	svc := NewService(ServiceDeps{
		Freelance: search.NewFreelanceClient(client),
		Agency:    search.NewAgencyClient(client),
		Referrer:  search.NewReferrerClient(client),
	})

	t.Run("matches by display name", func(t *testing.T) {
		got, err := svc.Query(ctx, QueryInput{
			Persona: search.PersonaFreelance,
			Query:   "Phase2 Alice",
		})
		require.NoError(t, err)
		require.NotNil(t, got)
		require.GreaterOrEqual(t, got.Found, 1)
	})

	t.Run("filter by skill", func(t *testing.T) {
		got, err := svc.Query(ctx, QueryInput{
			Persona: search.PersonaFreelance,
			Query:   "*",
			Filters: FilterInput{Skills: []string{"figma"}},
		})
		require.NoError(t, err)
		require.GreaterOrEqual(t, got.Found, 1)
		// Only Bob has figma in our fixture.
		var bobFound bool
		for _, d := range got.Documents {
			if d.DisplayName == "Phase2 Bob" {
				bobFound = true
			}
		}
		require.True(t, bobFound, "filter by skill should surface Bob")
	})

	t.Run("facet counts present", func(t *testing.T) {
		got, err := svc.Query(ctx, QueryInput{
			Persona: search.PersonaFreelance,
			Query:   "*",
		})
		require.NoError(t, err)
		require.NotNil(t, got.FacetCounts)
		require.NotEmpty(t, got.FacetCounts, "facet counts must be populated")
	})

	t.Run("highlights returned", func(t *testing.T) {
		got, err := svc.Query(ctx, QueryInput{
			Persona: search.PersonaFreelance,
			Query:   "Phase2",
		})
		require.NoError(t, err)
		require.GreaterOrEqual(t, got.Found, 2)
		// At least one hit must have a highlight on display_name.
		var anyHL bool
		for _, hl := range got.Highlights {
			if _, ok := hl["display_name"]; ok {
				anyHL = true
				break
			}
		}
		require.True(t, anyHL, "highlight on display_name must be returned for matching docs")
	})

	t.Run("scoped key roundtrip", func(t *testing.T) {
		key, err := client.GenerateScopedSearchKey(os.Getenv("TYPESENSE_INTEGRATION_API_KEY"),
			search.EmbeddedSearchParams{
				FilterBy:  "persona:freelance && is_published:true",
				ExpiresAt: time.Now().Add(time.Hour).Unix(),
			})
		require.NoError(t, err)
		require.NotEmpty(t, key)
	})
}
