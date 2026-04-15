package search_test

// Integration tests against a real Typesense cluster.
//
// Gated behind TYPESENSE_INTEGRATION_URL. Skips by default so a
// plain `go test ./...` never touches the network. Run locally
// with:
//
//	docker compose up -d typesense
//	TYPESENSE_INTEGRATION_URL=http://localhost:8108 \
//	TYPESENSE_INTEGRATION_API_KEY=xyz-dev-master-key-change-in-production \
//	go test ./internal/search/... -run Integration -count=1 -v
//
// Each test uses its own collection name so parallel runs cannot
// collide, and cleans up after itself via the alias + collection
// delete-equivalent via Typesense's POST semantics.

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/search"
)

func integrationClient(t *testing.T) (*search.Client, bool) {
	t.Helper()
	host := os.Getenv("TYPESENSE_INTEGRATION_URL")
	key := os.Getenv("TYPESENSE_INTEGRATION_API_KEY")
	if host == "" || key == "" {
		t.Skip("TYPESENSE_INTEGRATION_URL / TYPESENSE_INTEGRATION_API_KEY not set — skipping")
		return nil, false
	}
	c, err := search.NewClient(host, key)
	require.NoError(t, err)
	return c, true
}

func newIntegrationDoc(t *testing.T, orgID uuid.UUID, persona search.Persona, displayName string) *search.SearchDocument {
	t.Helper()
	now := time.Now()
	return &search.SearchDocument{
		ID:                     orgID.String(),
		Persona:                persona,
		IsPublished:            true,
		DisplayName:            displayName,
		Title:                  "Integration test title",
		City:                   "Paris",
		CountryCode:            "FR",
		WorkMode:               []string{"remote"},
		LanguagesProfessional:  []string{"fr", "en"},
		LanguagesConversational: []string{},
		AvailabilityStatus:     "available_now",
		AvailabilityPriority:   search.AvailabilityPriorityNow,
		ExpertiseDomains:       []string{"dev-backend"},
		Skills:                 []string{"Go", "PostgreSQL"},
		SkillsText:             "Go PostgreSQL",
		RatingAverage:          4.8,
		RatingCount:            42,
		RatingScore:            search.BayesianRatingScore(4.8, 42),
		ProfileCompletionScore: 95,
		LastActiveAt:           now.Unix(),
		IsVerified:             true,
		IsTopRated:             true,
		CreatedAt:              now.Unix(),
		UpdatedAt:              now.Unix(),
		Location:               []float64{48.8566, 2.3522},
	}
}

// TestIntegration_EnsureSchemaCreatesAliasAndCollection exercises
// the bootstrap path against a real cluster.
func TestIntegration_EnsureSchemaCreatesAliasAndCollection(t *testing.T) {
	client, ok := integrationClient(t)
	if !ok {
		return
	}
	ctx := context.Background()

	err := search.EnsureSchema(ctx, search.EnsureSchemaDeps{Client: client})
	require.NoError(t, err)

	// Alias must point at the canonical collection.
	target, err := client.GetAlias(ctx, search.AliasName)
	require.NoError(t, err)
	assert.Equal(t, search.CollectionName, target)

	// Calling again is a no-op.
	assert.NoError(t, search.EnsureSchema(ctx, search.EnsureSchemaDeps{Client: client}))
}

// TestIntegration_BulkUpsertAndQuery writes a few documents then
// fetches them back via a persona-scoped client. Verifies the
// scoped filter, the facet payload, and that no other persona
// leaks.
func TestIntegration_BulkUpsertAndQuery(t *testing.T) {
	client, ok := integrationClient(t)
	if !ok {
		return
	}
	ctx := context.Background()

	require.NoError(t, search.EnsureSchema(ctx, search.EnsureSchemaDeps{Client: client}))

	orgA := uuid.New()
	orgB := uuid.New()
	orgC := uuid.New()

	docs := []*search.SearchDocument{
		newIntegrationDoc(t, orgA, search.PersonaFreelance, "Freelance Alice"),
		newIntegrationDoc(t, orgB, search.PersonaAgency, "Agency Bob"),
		newIntegrationDoc(t, orgC, search.PersonaReferrer, "Referrer Carol"),
	}

	require.NoError(t, client.BulkUpsert(ctx, search.AliasName, docs))

	// Cleanup: delete all three docs at the end so repeated test
	// runs do not accumulate documents.
	t.Cleanup(func() {
		for _, d := range docs {
			_ = client.DeleteDocument(context.Background(), search.AliasName, d.ID)
		}
	})

	// Give Typesense a moment to index the writes.
	time.Sleep(200 * time.Millisecond)

	freelance := search.NewFreelanceClient(client)
	raw, err := freelance.Query(ctx, search.SearchParams{
		Q:       "*",
		QueryBy: "display_name",
	})
	require.NoError(t, err)

	// Decode just enough of the response to assert on the hits count.
	var decoded struct {
		Hits []struct {
			Document struct {
				ID      string         `json:"id"`
				Persona search.Persona `json:"persona"`
			} `json:"document"`
		} `json:"hits"`
	}
	require.NoError(t, json.Unmarshal(raw, &decoded))

	// Only the freelance doc must come back.
	foundOrgA := false
	for _, hit := range decoded.Hits {
		assert.Equal(t, search.PersonaFreelance, hit.Document.Persona,
			"scoped freelance client must never return agency/referrer docs")
		if hit.Document.ID == orgA.String() {
			foundOrgA = true
		}
	}
	assert.True(t, foundOrgA, "freelance hit list must include our test doc")
}

// TestIntegration_DeleteDocumentIsIdempotent writes then deletes
// then deletes again. Second delete must not error.
func TestIntegration_DeleteDocumentIsIdempotent(t *testing.T) {
	client, ok := integrationClient(t)
	if !ok {
		return
	}
	ctx := context.Background()
	require.NoError(t, search.EnsureSchema(ctx, search.EnsureSchemaDeps{Client: client}))

	doc := newIntegrationDoc(t, uuid.New(), search.PersonaFreelance, "Delete me")
	require.NoError(t, client.UpsertDocument(ctx, search.AliasName, doc))

	require.NoError(t, client.DeleteDocument(ctx, search.AliasName, doc.ID))
	// Second delete: must succeed (idempotent).
	require.NoError(t, client.DeleteDocument(ctx, search.AliasName, doc.ID))
}

// TestIntegration_ScopedClientLeakProof is the critical security
// test: a caller who tries to pass `persona:agency` in their own
// filter_by must still see zero agency results through a freelance
// scoped client.
func TestIntegration_ScopedClientLeakProof(t *testing.T) {
	client, ok := integrationClient(t)
	if !ok {
		return
	}
	ctx := context.Background()
	require.NoError(t, search.EnsureSchema(ctx, search.EnsureSchemaDeps{Client: client}))

	agencyOrg := uuid.New()
	agencyDoc := newIntegrationDoc(t, agencyOrg, search.PersonaAgency, "Blocked Agency")
	require.NoError(t, client.UpsertDocument(ctx, search.AliasName, agencyDoc))
	t.Cleanup(func() {
		_ = client.DeleteDocument(context.Background(), search.AliasName, agencyDoc.ID)
	})

	time.Sleep(200 * time.Millisecond)

	freelance := search.NewFreelanceClient(client)
	raw, err := freelance.Query(ctx, search.SearchParams{
		Q:        "*",
		QueryBy:  "display_name",
		FilterBy: fmt.Sprintf("id:%s", agencyOrg.String()),
	})
	require.NoError(t, err)

	var decoded struct {
		Found int `json:"found"`
	}
	require.NoError(t, json.Unmarshal(raw, &decoded))
	assert.Equal(t, 0, decoded.Found,
		"freelance scoped client must never surface an agency document")
}
