package handler

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	"marketplace-backend/internal/search"
)

// search_index_publisher.go defines the narrow port handlers use
// when they need to trigger a Typesense reindex after a mutation.
//
// It lives at the handler layer (not in port/service or
// app/searchindex) because the consumers — pricing handlers, social
// link handlers, organization shared profile handler — are the only
// places where we have BOTH the org id and the persona at the
// same call site without dragging the search package into every
// app service.
//
// The interface intentionally mirrors searchindex.Publisher so the
// concrete implementation in cmd/api/main.go is a drop-in.

// SearchIndexPublisher is the minimal contract the handlers depend
// on. A nil publisher is silently tolerated by the helper below so
// callers can always invoke it without optional-chain boilerplate.
type SearchIndexPublisher interface {
	PublishReindex(ctx context.Context, orgID uuid.UUID, persona search.Persona) error
}

// publishReindexBestEffort calls the publisher in a best-effort
// fashion: failures are logged at WARN but never returned. We
// intentionally keep this at the handler layer so a degraded search
// engine cannot block any user-facing mutation.
func publishReindexBestEffort(
	ctx context.Context,
	publisher SearchIndexPublisher,
	orgID uuid.UUID,
	persona search.Persona,
	logTag string,
) {
	if publisher == nil {
		return
	}
	if err := publisher.PublishReindex(ctx, orgID, persona); err != nil {
		slog.Warn("search reindex publish failed",
			"source", logTag,
			"org_id", orgID,
			"persona", persona,
			"error", err)
	}
}
