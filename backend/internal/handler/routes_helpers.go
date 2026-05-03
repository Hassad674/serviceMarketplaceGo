package handler

import (
	"net/http"

	"marketplace-backend/internal/handler/middleware"
)

// idempotencyMiddleware returns the SEC-FINAL-02 idempotency middleware
// pre-bound with the deps cache, or a no-op pass-through when no cache
// has been wired (tests + worktrees without Redis still boot).
//
// The pre-bound closure is the canonical shape `chi.Router.Use` /
// `chi.Router.With` accept (`func(http.Handler) http.Handler`), so each
// of the 6 protected routes wires it with a single `r.With(idem)` call.
func idempotencyMiddleware(deps RouterDeps) func(http.Handler) http.Handler {
	if deps.IdempotencyCache == nil {
		return func(next http.Handler) http.Handler {
			return next
		}
	}
	return middleware.Idempotency(deps.IdempotencyCache)
}
