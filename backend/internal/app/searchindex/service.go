// Package searchindex is the app-layer bridge between the outbox
// worker and the search.Indexer. It owns exactly two responsibilities:
//
//  1. Decode a pending_event payload of type `search.reindex` or
//     `search.delete` into typed arguments.
//  2. Call the right method on the search package (BuildDocument →
//     UpsertDocument, or DeleteDocument) with the appropriate
//     collection name.
//
// Everything else — building the document, talking to Typesense,
// generating embeddings — lives in the search package. This service
// is only 150 lines including tests because it is a pure translation
// layer.
//
// Removing the search feature entirely = delete this folder + the
// three wiring lines in cmd/api/main.go. The outbox worker keeps
// working for every other event type.
package searchindex

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/pendingevent"
	"marketplace-backend/internal/search"
)

// Service is the outbox event handler for search.reindex and
// search.delete. Stateless beyond its dependencies; safe to share
// across goroutines.
type Service struct {
	client   SearchClient
	indexer  DocumentBuilder
	logger   *slog.Logger
	collection string
}

// SearchClient is the narrow port the service needs from the
// Typesense wrapper. Declared here (not in the search package)
// because it's a consumer-side interface — the classic Go pattern
// of defining interfaces at the point of use, which keeps the
// production *search.Client lean.
type SearchClient interface {
	UpsertDocument(ctx context.Context, collection string, doc *search.SearchDocument) error
	DeleteDocument(ctx context.Context, collection, docID string) error
	// DeleteDocumentsByFilter lets the delete handler remove every
	// persona variant of an org in one call, matching the composite
	// ID scheme chosen in phase 1.
	DeleteDocumentsByFilter(ctx context.Context, collection, filterBy string) (int, error)
}

// DocumentBuilder mirrors the subset of search.Indexer this service
// uses. Same rationale as SearchClient — consumer-side interface,
// easy to mock in tests without pulling in the real indexer.
type DocumentBuilder interface {
	BuildDocument(ctx context.Context, orgID uuid.UUID, persona search.Persona) (*search.SearchDocument, error)
}

// Config groups the constructor parameters. Collection is the name
// the service writes to — typically search.AliasName so the worker
// upserts against the alias and cannot pin to a stale `_v1` target
// if the operator swaps the alias mid-run.
type Config struct {
	Client     SearchClient
	Indexer    DocumentBuilder
	Logger     *slog.Logger
	Collection string
}

// NewService constructs the handler service. Every dependency is
// required — we refuse nil values because this runs in a background
// worker where a silent nil would surface as a panic much later.
func NewService(cfg Config) (*Service, error) {
	if cfg.Client == nil {
		return nil, fmt.Errorf("searchindex service: search client is required")
	}
	if cfg.Indexer == nil {
		return nil, fmt.Errorf("searchindex service: indexer is required")
	}
	collection := cfg.Collection
	if collection == "" {
		collection = search.AliasName
	}
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{
		client:     cfg.Client,
		indexer:    cfg.Indexer,
		logger:     logger,
		collection: collection,
	}, nil
}

// ReindexPayload is the typed shape of a `search.reindex` event.
type ReindexPayload struct {
	OrganizationID uuid.UUID       `json:"organization_id"`
	Persona        search.Persona  `json:"persona"`
}

// DeletePayload is the typed shape of a `search.delete` event.
type DeletePayload struct {
	OrganizationID uuid.UUID `json:"organization_id"`
}

// HandleReindex is the EventHandler-compatible entry point for
// search.reindex events. Decodes the payload, builds the document,
// and upserts it into Typesense.
func (s *Service) HandleReindex(ctx context.Context, event *pendingevent.PendingEvent) error {
	if event == nil {
		return fmt.Errorf("searchindex reindex: nil event")
	}
	var payload ReindexPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return fmt.Errorf("searchindex reindex: decode payload: %w", err)
	}
	if payload.OrganizationID == uuid.Nil {
		return fmt.Errorf("searchindex reindex: organization_id is required")
	}
	if !payload.Persona.IsValid() {
		return fmt.Errorf("searchindex reindex: invalid persona %q", payload.Persona)
	}

	doc, err := s.indexer.BuildDocument(ctx, payload.OrganizationID, payload.Persona)
	if err != nil {
		// "Not found" is not a retryable failure — the org may
		// have been deleted between the event being scheduled
		// and the worker running. Log + succeed so the event
		// does not retry forever.
		if isSoftMissing(err) {
			s.logger.Warn("searchindex reindex: org missing, skipping",
				"event_id", event.ID,
				"organization_id", payload.OrganizationID,
				"persona", payload.Persona)
			return nil
		}
		return fmt.Errorf("searchindex reindex: build document: %w", err)
	}

	if err := s.client.UpsertDocument(ctx, s.collection, doc); err != nil {
		return fmt.Errorf("searchindex reindex: upsert: %w", err)
	}
	s.logger.Debug("searchindex reindex: document upserted",
		"organization_id", payload.OrganizationID,
		"persona", payload.Persona)
	return nil
}

// HandleDelete is the EventHandler-compatible entry point for
// search.delete events. Idempotent: the Typesense client swallows
// 404s so a duplicate delete is a no-op.
func (s *Service) HandleDelete(ctx context.Context, event *pendingevent.PendingEvent) error {
	if event == nil {
		return fmt.Errorf("searchindex delete: nil event")
	}
	var payload DeletePayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return fmt.Errorf("searchindex delete: decode payload: %w", err)
	}
	if payload.OrganizationID == uuid.Nil {
		return fmt.Errorf("searchindex delete: organization_id is required")
	}

	// Wipe every persona variant (freelance / agency / referrer)
	// in one call — the composite ID scheme means a per-ID delete
	// would miss the others.
	filter := fmt.Sprintf("organization_id:%s", payload.OrganizationID.String())
	removed, err := s.client.DeleteDocumentsByFilter(ctx, s.collection, filter)
	if err != nil {
		return fmt.Errorf("searchindex delete: %w", err)
	}
	s.logger.Debug("searchindex delete: documents removed",
		"organization_id", payload.OrganizationID,
		"count", removed)
	return nil
}

// isSoftMissing reports whether an error from BuildDocument
// indicates the org no longer exists. We match on the error text
// rather than a typed sentinel because the repository wraps the
// cause with fmt.Errorf — phase 2 will introduce a typed sentinel
// once we have a clearer notion of "soft" vs "hard" failures.
func isSoftMissing(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return containsAny(msg, []string{"not found", "no rows"})
}

// containsAny is a tiny helper kept local to avoid pulling in
// strings.ContainsAny (which matches bytes, not substrings).
func containsAny(s string, needles []string) bool {
	for _, n := range needles {
		if len(n) == 0 {
			continue
		}
		if containsSubstring(s, n) {
			return true
		}
	}
	return false
}

// containsSubstring is a minimal substring check so the soft-miss
// detection has zero external dependencies and stays easy to audit.
func containsSubstring(haystack, needle string) bool {
	return len(haystack) >= len(needle) && indexOfString(haystack, needle) >= 0
}

func indexOfString(haystack, needle string) int {
	// Forward scan; fine for short error messages.
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}

