package searchindex_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/app/searchindex"
	"marketplace-backend/internal/domain/pendingevent"
	"marketplace-backend/internal/search"
)

// fakeClient records every UpsertDocument + DeleteDocument call so
// tests can assert that the service routed the payload correctly.
type fakeClient struct {
	upserted []*search.SearchDocument
	deleted  []string
	upsertErr error
	deleteErr error
}

func (f *fakeClient) UpsertDocument(_ context.Context, _ string, doc *search.SearchDocument) error {
	f.upserted = append(f.upserted, doc)
	return f.upsertErr
}

func (f *fakeClient) DeleteDocument(_ context.Context, _ string, docID string) error {
	f.deleted = append(f.deleted, docID)
	return f.deleteErr
}

// fakeIndexer builds a trivially-valid SearchDocument for any
// organization, optionally failing with the configured error.
type fakeIndexer struct {
	buildErr error
	calls    int
}

func (f *fakeIndexer) BuildDocument(_ context.Context, orgID uuid.UUID, persona search.Persona) (*search.SearchDocument, error) {
	f.calls++
	if f.buildErr != nil {
		return nil, f.buildErr
	}
	now := time.Now()
	return &search.SearchDocument{
		ID:          orgID.String(),
		Persona:     persona,
		DisplayName: "Test Org",
		IsPublished: true,
		WorkMode:    []string{"remote"},
		CreatedAt:   now.Unix(),
		UpdatedAt:   now.Unix(),
	}, nil
}

func silentLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func mustService(t *testing.T, client searchindex.SearchClient, indexer searchindex.DocumentBuilder) *searchindex.Service {
	t.Helper()
	svc, err := searchindex.NewService(searchindex.Config{
		Client:  client,
		Indexer: indexer,
		Logger:  silentLogger(),
	})
	require.NoError(t, err)
	return svc
}

func mustEvent(t *testing.T, eventType pendingevent.EventType, payload any) *pendingevent.PendingEvent {
	t.Helper()
	raw, err := json.Marshal(payload)
	require.NoError(t, err)
	return &pendingevent.PendingEvent{
		ID:        uuid.New(),
		EventType: eventType,
		Payload:   raw,
		FiresAt:   time.Now(),
		Status:    pendingevent.StatusProcessing,
		Attempts:  1,
	}
}

func TestNewService_Validation(t *testing.T) {
	_, err := searchindex.NewService(searchindex.Config{Indexer: &fakeIndexer{}})
	assert.ErrorContains(t, err, "search client is required")

	_, err = searchindex.NewService(searchindex.Config{Client: &fakeClient{}})
	assert.ErrorContains(t, err, "indexer is required")

	svc, err := searchindex.NewService(searchindex.Config{
		Client:  &fakeClient{},
		Indexer: &fakeIndexer{},
	})
	require.NoError(t, err)
	assert.NotNil(t, svc)
}

func TestHandleReindex_HappyPath(t *testing.T) {
	client := &fakeClient{}
	indexer := &fakeIndexer{}
	svc := mustService(t, client, indexer)

	orgID := uuid.New()
	ev := mustEvent(t, pendingevent.TypeSearchReindex, searchindex.ReindexPayload{
		OrganizationID: orgID,
		Persona:        search.PersonaFreelance,
	})

	require.NoError(t, svc.HandleReindex(context.Background(), ev))
	assert.Equal(t, 1, indexer.calls)
	require.Len(t, client.upserted, 1)
	assert.Equal(t, orgID.String(), client.upserted[0].ID)
	assert.Equal(t, search.PersonaFreelance, client.upserted[0].Persona)
}

func TestHandleReindex_InvalidPayload(t *testing.T) {
	svc := mustService(t, &fakeClient{}, &fakeIndexer{})

	ev := &pendingevent.PendingEvent{
		ID:        uuid.New(),
		EventType: pendingevent.TypeSearchReindex,
		Payload:   []byte("not json"),
	}
	assert.ErrorContains(t, svc.HandleReindex(context.Background(), ev), "decode payload")
}

func TestHandleReindex_MissingOrg(t *testing.T) {
	svc := mustService(t, &fakeClient{}, &fakeIndexer{})

	ev := mustEvent(t, pendingevent.TypeSearchReindex, searchindex.ReindexPayload{
		Persona: search.PersonaFreelance,
	})
	assert.ErrorContains(t, svc.HandleReindex(context.Background(), ev), "organization_id is required")
}

func TestHandleReindex_InvalidPersona(t *testing.T) {
	svc := mustService(t, &fakeClient{}, &fakeIndexer{})

	ev := mustEvent(t, pendingevent.TypeSearchReindex, searchindex.ReindexPayload{
		OrganizationID: uuid.New(),
		Persona:        "enterprise",
	})
	assert.ErrorContains(t, svc.HandleReindex(context.Background(), ev), "invalid persona")
}

func TestHandleReindex_SoftMissingOrg(t *testing.T) {
	// If the indexer reports "not found", the service swallows
	// the error + succeeds — the org was deleted between schedule
	// and worker run, and retrying forever is pointless.
	client := &fakeClient{}
	indexer := &fakeIndexer{buildErr: errors.New("search repository: freelance profile not found for org abc")}
	svc := mustService(t, client, indexer)

	ev := mustEvent(t, pendingevent.TypeSearchReindex, searchindex.ReindexPayload{
		OrganizationID: uuid.New(),
		Persona:        search.PersonaFreelance,
	})

	require.NoError(t, svc.HandleReindex(context.Background(), ev))
	assert.Len(t, client.upserted, 0, "nothing must be upserted when the org is gone")
}

func TestHandleReindex_HardBuildFailureRetries(t *testing.T) {
	client := &fakeClient{}
	indexer := &fakeIndexer{buildErr: errors.New("postgres down")}
	svc := mustService(t, client, indexer)

	ev := mustEvent(t, pendingevent.TypeSearchReindex, searchindex.ReindexPayload{
		OrganizationID: uuid.New(),
		Persona:        search.PersonaFreelance,
	})

	err := svc.HandleReindex(context.Background(), ev)
	assert.ErrorContains(t, err, "postgres down")
}

func TestHandleReindex_TypesenseError(t *testing.T) {
	client := &fakeClient{upsertErr: errors.New("typesense unavailable")}
	svc := mustService(t, client, &fakeIndexer{})

	ev := mustEvent(t, pendingevent.TypeSearchReindex, searchindex.ReindexPayload{
		OrganizationID: uuid.New(),
		Persona:        search.PersonaFreelance,
	})

	err := svc.HandleReindex(context.Background(), ev)
	assert.ErrorContains(t, err, "typesense unavailable")
}

func TestHandleDelete_HappyPath(t *testing.T) {
	client := &fakeClient{}
	svc := mustService(t, client, &fakeIndexer{})

	orgID := uuid.New()
	ev := mustEvent(t, pendingevent.TypeSearchDelete, searchindex.DeletePayload{
		OrganizationID: orgID,
	})

	require.NoError(t, svc.HandleDelete(context.Background(), ev))
	require.Len(t, client.deleted, 1)
	assert.Equal(t, orgID.String(), client.deleted[0])
}

func TestHandleDelete_MissingPayload(t *testing.T) {
	svc := mustService(t, &fakeClient{}, &fakeIndexer{})

	ev := mustEvent(t, pendingevent.TypeSearchDelete, searchindex.DeletePayload{})
	assert.ErrorContains(t, svc.HandleDelete(context.Background(), ev), "organization_id is required")
}

func TestHandleDelete_InvalidJSON(t *testing.T) {
	svc := mustService(t, &fakeClient{}, &fakeIndexer{})

	ev := &pendingevent.PendingEvent{
		ID:        uuid.New(),
		EventType: pendingevent.TypeSearchDelete,
		Payload:   []byte("{bad"),
	}
	assert.ErrorContains(t, svc.HandleDelete(context.Background(), ev), "decode payload")
}

func TestHandleDelete_NilEvent(t *testing.T) {
	svc := mustService(t, &fakeClient{}, &fakeIndexer{})

	assert.ErrorContains(t, svc.HandleDelete(context.Background(), nil), "nil event")
}
