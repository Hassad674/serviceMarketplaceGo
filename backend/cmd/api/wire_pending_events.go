package main

import (
	"time"

	"marketplace-backend/internal/adapter/postgres"
	"marketplace-backend/internal/adapter/worker"
	"marketplace-backend/internal/adapter/worker/handlers"
	proposalapp "marketplace-backend/internal/app/proposal"
	"marketplace-backend/internal/domain/pendingevent"
	"marketplace-backend/internal/handler"
)

// newPendingEventsWorker builds the phase-6 unified scheduler. Runs
// in a background goroutine alongside the API server, ticks every 30
// seconds, and drives the auto-approval, fund-reminder, and
// auto-close timers. Multiple instances of this binary are safe to
// run side by side — PopDue uses FOR UPDATE SKIP LOCKED so workers
// never claim the same row.
//
// Registers three live handlers (milestone auto-approve, fund
// reminder, proposal auto-close) plus a drain handler for the
// legacy stripe_transfer event. The drain is kept registered so any
// stale rows still sitting in pending_events from a previous deploy
// get marked "done" on the next worker tick.
//
// The search-related event types are registered later, by
// wireSearchIndexer, when Typesense is configured — both helpers
// share the same *worker.Worker instance returned here.
//
// The Stripe webhook handler is registered later by
// registerStripeWebhookWorker (called from main.go after the
// StripeHandler is wired), keeping this constructor free of the
// stripe / handler / app cross-imports.
func newPendingEventsWorker(
	pendingEventsRepo *postgres.PendingEventRepository,
	proposalSvc *proposalapp.Service,
) *worker.Worker {
	w := worker.New(pendingEventsRepo, worker.Config{
		TickInterval: 30 * time.Second,
		BatchSize:    20,
	})
	w.Register(pendingevent.TypeMilestoneAutoApprove, handlers.NewMilestoneAutoApproveHandler(proposalSvc))
	w.Register(pendingevent.TypeMilestoneFundReminder, handlers.NewMilestoneFundReminderHandler(proposalSvc))
	w.Register(pendingevent.TypeProposalAutoClose, handlers.NewProposalAutoCloseHandler(proposalSvc))
	// stripe_transfer is the legacy auto-payout outbox event. Payouts
	// now go through the wallet's manual RequestPayout / Retry path
	// — no new events are enqueued. The drain handler is registered
	// only so any stale rows still sitting in pending_events from a
	// previous deploy get marked "done" on the next worker tick.
	w.Register(pendingevent.TypeStripeTransfer, handlers.NewLegacyStripeTransferDrainHandler())
	return w
}

// registerStripeWebhookWorker attaches the P8 async Stripe webhook
// handler to an already-built worker. Called from main.go after the
// StripeHandler is wired so the dispatcher is non-nil.
//
// When stripeHandler is nil (Stripe not configured for this
// deployment) the registration is a no-op — the webhook HTTP route
// is also absent, so no row of TypeStripeWebhook will ever land in
// the queue.
func registerStripeWebhookWorker(w *worker.Worker, stripeHandler *handler.StripeHandler) {
	if w == nil || stripeHandler == nil {
		return
	}
	w.Register(
		pendingevent.TypeStripeWebhook,
		handlers.NewStripeWebhookHandler(stripeHandler),
	)
}
