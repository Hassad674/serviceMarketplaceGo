package search

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/search"
	"marketplace-backend/internal/search/features"
)

// document_adapter_test.go pins the Typesense → features.SearchDocumentLite
// projection. A field drift in SearchDocument or SearchDocumentLite must
// show up here before downstream extractors silently ingest a zero value.

func TestTypesenseHit_ToSearchDocumentLite_FullMapping(t *testing.T) {
	doc := search.SearchDocument{
		OrganizationID:         "org-123",
		Persona:                search.PersonaFreelance,
		Skills:                 []string{"react", "go"},
		SkillsText:             "react go microservices",
		RatingAverage:          4.8,
		RatingCount:            23,
		CompletedProjects:      14,
		ProfileCompletionScore: 88,
		LastActiveAt:           1_700_000_000,
		ResponseRate:           0.91,
		IsVerified:             true,
		UniqueClientsCount:     11,
		RepeatClientRate:       0.36,
		UniqueReviewersCount:   18,
		MaxReviewerShare:       0.17,
		ReviewRecencyFactor:    0.82,
		LostDisputesCount:      1,
		AccountAgeDays:         420,
	}
	hit := TypesenseHit{Document: doc, TextMatchBucket: 7}

	lite := hit.ToSearchDocumentLite(1_700_000_500)

	assert.Equal(t, "org-123", lite.OrganizationID)
	assert.Equal(t, features.PersonaFreelance, lite.Persona)
	assert.Equal(t, []string{"react", "go"}, lite.Skills)
	assert.Equal(t, "react go microservices", lite.SkillsText)
	assert.Equal(t, "", lite.About)
	assert.InDelta(t, 4.8, lite.RatingAverage, 1e-9)
	assert.Equal(t, int32(23), lite.RatingCount)
	assert.Equal(t, int32(14), lite.CompletedProjects)
	assert.Equal(t, int32(88), lite.ProfileCompletionScore)
	assert.Equal(t, int64(1_700_000_000), lite.LastActiveAt)
	assert.InDelta(t, 0.91, lite.ResponseRate, 1e-9)
	assert.True(t, lite.IsVerified)
	assert.Equal(t, int32(11), lite.UniqueClientsCount)
	assert.InDelta(t, 0.36, lite.RepeatClientRate, 1e-9)
	assert.Equal(t, int32(18), lite.UniqueReviewersCount)
	assert.InDelta(t, 0.17, lite.MaxReviewerShare, 1e-9)
	assert.InDelta(t, 0.82, lite.ReviewRecencyFactor, 1e-9)
	assert.Equal(t, int32(1), lite.LostDisputesCount)
	assert.Equal(t, int32(420), lite.AccountAgeDays)
	assert.Equal(t, int64(1_700_000_500), lite.NowUnix)
	assert.Equal(t, 7, lite.TextMatchBucket)
}

func TestTypesenseHit_ToSearchDocumentLite_MissingFieldsAreZeroSafe(t *testing.T) {
	// SearchDocumentLite is entirely zero-safe: every extractor must
	// handle the all-zero input without crashing. This test encodes
	// that contract.
	hit := TypesenseHit{
		Document: search.SearchDocument{
			OrganizationID: "org-empty",
			Persona:        search.PersonaAgency,
		},
	}
	lite := hit.ToSearchDocumentLite(0)

	assert.Equal(t, "org-empty", lite.OrganizationID)
	assert.Equal(t, features.PersonaAgency, lite.Persona)
	assert.Empty(t, lite.Skills)
	assert.Empty(t, lite.SkillsText)
	assert.Empty(t, lite.About)
	assert.Zero(t, lite.RatingAverage)
	assert.Equal(t, int32(0), lite.RatingCount)
	assert.Equal(t, int32(0), lite.CompletedProjects)
	assert.Equal(t, int32(0), lite.ProfileCompletionScore)
	assert.Equal(t, int64(0), lite.LastActiveAt)
	assert.Zero(t, lite.ResponseRate)
	assert.False(t, lite.IsVerified)
	assert.Equal(t, int32(0), lite.UniqueClientsCount)
	assert.Zero(t, lite.RepeatClientRate)
	assert.Equal(t, int32(0), lite.UniqueReviewersCount)
	assert.Zero(t, lite.MaxReviewerShare)
	assert.Zero(t, lite.ReviewRecencyFactor)
	assert.Equal(t, int32(0), lite.LostDisputesCount)
	assert.Equal(t, int32(0), lite.AccountAgeDays)
	assert.Equal(t, int64(0), lite.NowUnix)
	assert.Equal(t, 0, lite.TextMatchBucket)
}

func TestTypesenseHit_ToSearchDocumentLite_PersonaRoundTrip(t *testing.T) {
	cases := []struct {
		doc    search.Persona
		expect features.Persona
	}{
		{search.PersonaFreelance, features.PersonaFreelance},
		{search.PersonaAgency, features.PersonaAgency},
		{search.PersonaReferrer, features.PersonaReferrer},
	}
	for _, tc := range cases {
		t.Run(string(tc.doc), func(t *testing.T) {
			hit := TypesenseHit{Document: search.SearchDocument{Persona: tc.doc}}
			lite := hit.ToSearchDocumentLite(0)
			assert.Equal(t, tc.expect, lite.Persona)
			// The Persona type alias means the lite value reports
			// valid for the features package, not just byte equality.
			assert.True(t, lite.Persona.IsValid())
		})
	}
}

func TestTypesenseHit_ToSearchDocumentLite_InjectedNowUnixIsPropagated(t *testing.T) {
	now := time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC).Unix()
	hit := TypesenseHit{Document: search.SearchDocument{OrganizationID: "x"}}
	lite := hit.ToSearchDocumentLite(now)
	assert.Equal(t, now, lite.NowUnix)
}

func TestTypesenseHit_ToSearchDocumentLite_SlicesShareBackingArray(t *testing.T) {
	// Contract note from document_adapter.go: the adapter never
	// allocates a copy — it shares the backing array with the input
	// slice. Ensuring this in a test documents the invariant.
	skills := []string{"a", "b"}
	doc := search.SearchDocument{Skills: skills, Persona: search.PersonaFreelance}
	hit := TypesenseHit{Document: doc}
	lite := hit.ToSearchDocumentLite(0)
	require.Equal(t, 2, len(lite.Skills))
	// Verify shared backing array via direct index comparison after
	// mutating the original.
	skills[0] = "mutated"
	assert.Equal(t, "mutated", lite.Skills[0], "lite should share backing array with input")
}
