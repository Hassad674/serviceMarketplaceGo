package features

import (
	"encoding/json"
	"fmt"
	"testing"
)

// Sample check : feed three documents from the shared Typesense cluster
// (indexed by phase 6B with the 7 new ranking signals) through the
// extractor + print the resulting Features. Run manually with
// `go test -run TestSampleRealDocs -v ./internal/search/features/`.
//
// The test is skipped by default (not a regression check — it's a
// diagnostic tool). The printed output is the "sample Features vector
// for 3 real docs" required by the orchestrator hand-off.
func TestSampleRealDocs(t *testing.T) {
	if testing.Short() {
		t.Skip("skipped in short mode")
	}
	ext := NewDefaultExtractor(DefaultConfig())
	const now = 1_776_300_000
	q := Query{
		Text:             "react aws senior",
		NormalisedTokens: []string{"react", "aws", "senior"},
		Persona:          PersonaFreelance,
	}

	docs := []SearchDocumentLite{
		{
			OrganizationID: "63706cf2-4289-414d-a490-435a8269a95b",
			Persona:        PersonaFreelance,
			Skills:         []string{"react native", "asp.net core", "astro", "aws", "docker"},
			RatingAverage:  3.666, RatingCount: 3, CompletedProjects: 21,
			ProfileCompletionScore: 85, LastActiveAt: 1776258965,
			ResponseRate: 1, IsVerified: true,
			UniqueClientsCount:  3, RepeatClientRate: 0.6667,
			UniqueReviewersCount: 2, MaxReviewerShare: 0.6667,
			ReviewRecencyFactor: 0.9826, LostDisputesCount: 0, AccountAgeDays: 8,
			NowUnix: now, TextMatchBucket: 5,
		},
		{
			OrganizationID: "f0261698-a0fd-4c8f-95af-1e771a267515",
			Persona:        PersonaFreelance,
			RatingAverage:  5, RatingCount: 1, CompletedProjects: 21,
			ProfileCompletionScore: 40, LastActiveAt: 1775919437,
			ResponseRate: 0.6667, IsVerified: false,
			UniqueClientsCount:  1, RepeatClientRate: 1,
			UniqueReviewersCount: 1, MaxReviewerShare: 1,
			ReviewRecencyFactor: 0.9413, LostDisputesCount: 0, AccountAgeDays: 25,
			NowUnix: now, TextMatchBucket: 0,
		},
		{
			OrganizationID: "bfdb7d3d-8114-441a-9143-3d0e0b22eb79",
			Persona:        PersonaFreelance,
			RatingAverage:  0, RatingCount: 0, CompletedProjects: 0,
			ProfileCompletionScore: 80, LastActiveAt: 1776274927,
			ResponseRate: 0, IsVerified: true,
			UniqueClientsCount:  0, RepeatClientRate: 0,
			UniqueReviewersCount: 0, MaxReviewerShare: 0,
			ReviewRecencyFactor: 0, LostDisputesCount: 0, AccountAgeDays: 4,
			NowUnix: now, TextMatchBucket: 3,
		},
	}

	for _, d := range docs {
		f := ext.Extract(q, d)
		b, _ := json.MarshalIndent(map[string]any{
			"org":      d.OrganizationID,
			"features": f,
		}, "", "  ")
		fmt.Println(string(b))
	}
}
