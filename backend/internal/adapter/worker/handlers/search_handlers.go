package handlers

// search_handlers.go wires the search.reindex and search.delete
// outbox events into the existing pending_events worker dispatch
// loop. Tiny adapter — each handler struct carries a reference to
// the searchindex service and forwards the event to the right
// method. The service owns all decoding + business logic.

import (
	"context"

	searchindexapp "marketplace-backend/internal/app/searchindex"
	"marketplace-backend/internal/domain/pendingevent"
)

// SearchIndexService is the narrow contract this adapter needs.
// Declared here (at the point of use) so the handler package does
// not import the service's full public surface when all it cares
// about is HandleReindex / HandleDelete.
type SearchIndexService interface {
	HandleReindex(ctx context.Context, event *pendingevent.PendingEvent) error
	HandleDelete(ctx context.Context, event *pendingevent.PendingEvent) error
}

// Compile-time guard: the real *searchindex.Service must satisfy
// the above interface. If someone removes HandleReindex in the
// future, the build breaks here instead of at registration time.
var _ SearchIndexService = (*searchindexapp.Service)(nil)

// SearchReindexHandler dispatches a search.reindex event to the
// service. Registered under pendingevent.TypeSearchReindex by
// cmd/api/main.go.
type SearchReindexHandler struct {
	svc SearchIndexService
}

// NewSearchReindexHandler builds the handler from the service.
func NewSearchReindexHandler(svc SearchIndexService) *SearchReindexHandler {
	return &SearchReindexHandler{svc: svc}
}

// Handle forwards the event. The service owns error wrapping so
// the worker's backoff/retry semantics kick in on real failures
// (DB down, Typesense unreachable) while swallowing soft misses.
func (h *SearchReindexHandler) Handle(ctx context.Context, event *pendingevent.PendingEvent) error {
	return h.svc.HandleReindex(ctx, event)
}

// SearchDeleteHandler dispatches a search.delete event to the
// service. Registered under pendingevent.TypeSearchDelete.
type SearchDeleteHandler struct {
	svc SearchIndexService
}

// NewSearchDeleteHandler builds the handler from the service.
func NewSearchDeleteHandler(svc SearchIndexService) *SearchDeleteHandler {
	return &SearchDeleteHandler{svc: svc}
}

// Handle forwards the event.
func (h *SearchDeleteHandler) Handle(ctx context.Context, event *pendingevent.PendingEvent) error {
	return h.svc.HandleDelete(ctx, event)
}
