package searchindex_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/app/searchindex"
	"marketplace-backend/internal/domain/pendingevent"
	"marketplace-backend/internal/search"
)

// jsonImpl is the real decoder, aliased so jsonDecode stays a thin
// helper without pulling encoding/json into its signature.
var jsonImpl = json.Unmarshal

// service_integration_test.go exercises the RGPD delete flow end-
// to-end against a real Typesense cluster. The test:
//
//   1. Indexes two documents under the same organization_id (one
//      freelance, one agency — the composite-id scheme means the
//      two rows coexist in the same collection).
//   2. Fires a search.delete event.
//   3. Asserts the service issued a DeleteDocumentsByFilter request
//      targeting the organization_id, and that Typesense reports
//      zero hits when the collection is queried for the org.
//
// The test is gated by TYPESENSE_INTEGRATION_URL (+ API key) so CI
// runs without a live cluster still pass.

func typesenseFromEnv(t *testing.T) (*search.Client, string) {
	t.Helper()
	host := os.Getenv("TYPESENSE_INTEGRATION_URL")
	if host == "" {
		t.Skip("set TYPESENSE_INTEGRATION_URL to run searchindex integration tests")
	}
	apiKey := os.Getenv("TYPESENSE_API_KEY")
	require.NotEmpty(t, apiKey, "TYPESENSE_API_KEY must be set when TYPESENSE_INTEGRATION_URL is set")

	ts, err := search.NewClient(host, apiKey)
	require.NoError(t, err)
	return ts, search.AliasName
}

// TestIntegration_RGPDDelete_ClearsAllPersonaDocuments indexes two
// documents under the same organization_id and asserts that a
// search.delete event removes all of them.
func TestIntegration_RGPDDelete_ClearsAllPersonaDocuments(t *testing.T) {
	ts, collection := typesenseFromEnv(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	require.NoError(t, search.EnsureSchema(ctx, search.EnsureSchemaDeps{
		Client: ts,
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}))

	orgID := uuid.New()
	docs := []*search.SearchDocument{
		testSearchDoc(orgID, search.PersonaFreelance),
		testSearchDoc(orgID, search.PersonaAgency),
	}
	for _, d := range docs {
		require.NoError(t, ts.UpsertDocument(ctx, collection, d))
	}
	t.Cleanup(func() {
		// Defensive cleanup — the test body deletes the docs but if
		// an assertion fails mid-run we should not leave test rows
		// behind in the shared cluster.
		filter := fmt.Sprintf("organization_id:%s", orgID.String())
		_, _ = ts.DeleteDocumentsByFilter(context.Background(), collection, filter)
	})

	// Sanity: both docs are present.
	count := typesenseCountForOrg(t, ctx, ts, collection, orgID)
	assert.Equal(t, 2, count, "expected two docs before delete")

	// Build the service with the real Typesense client + a no-op
	// indexer (we only exercise the delete path here).
	svc, err := searchindex.NewService(searchindex.Config{
		Client:     ts,
		Indexer:    &noopIndexer{},
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
		Collection: collection,
	})
	require.NoError(t, err)

	ev := mustEvent(t, pendingevent.TypeSearchDelete, searchindex.DeletePayload{
		OrganizationID: orgID,
	})
	require.NoError(t, svc.HandleDelete(ctx, ev))

	// Post-condition: zero docs for the org.
	count = typesenseCountForOrg(t, ctx, ts, collection, orgID)
	assert.Equal(t, 0, count, "expected all docs removed after search.delete")
}

func typesenseCountForOrg(t *testing.T, ctx context.Context, ts *search.Client, collection string, orgID uuid.UUID) int {
	t.Helper()
	raw, err := ts.Query(ctx, collection, search.SearchParams{
		Q:        "*",
		QueryBy:  "display_name",
		FilterBy: fmt.Sprintf("organization_id:%s", orgID.String()),
		PerPage:  0,
	})
	require.NoError(t, err)

	var envelope struct {
		Found int `json:"found"`
	}
	require.NoError(t, jsonDecode(raw, &envelope))
	return envelope.Found
}

func testSearchDoc(orgID uuid.UUID, persona search.Persona) *search.SearchDocument {
	now := time.Now()
	return &search.SearchDocument{
		ID:             orgID.String() + ":" + string(persona),
		OrganizationID: orgID.String(),
		Persona:        persona,
		IsPublished:    true,
		DisplayName:    "RGPD Test Org",
		Title:          "Integration Test Title",
		WorkMode:       []string{"remote"},
		Skills:         []string{"react", "go"},
		SkillsText:     "react go",
		LanguagesProfessional:   []string{"en"},
		LanguagesConversational: []string{},
		ExpertiseDomains:        []string{"dev-backend"},
		RatingScore:             1.0,
		LastActiveAt:            now.Unix(),
		CreatedAt:               now.Unix(),
		UpdatedAt:               now.Unix(),
	}
}

// noopIndexer satisfies the DocumentBuilder port with a never-called
// implementation; the delete flow does not need the indexer.
type noopIndexer struct{}

func (noopIndexer) BuildDocument(_ context.Context, _ uuid.UUID, _ search.Persona) (*search.SearchDocument, error) {
	return nil, fmt.Errorf("noopIndexer.BuildDocument should not be called")
}

// jsonDecode is a tiny wrapper around encoding/json so we avoid
// duplicating the import in multiple helpers.
func jsonDecode(data []byte, v any) error {
	return jsonImpl(data, v)
}
