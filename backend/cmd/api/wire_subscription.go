package main

import (
	"database/sql"
	"log/slog"
	"os"

	"marketplace-backend/internal/adapter/postgres"
	redisadapter "marketplace-backend/internal/adapter/redis"
	stripeadapter "marketplace-backend/internal/adapter/stripe"
	paymentapp "marketplace-backend/internal/app/payment"
	subscriptionapp "marketplace-backend/internal/app/subscription"
	webhookidempotencyapp "marketplace-backend/internal/app/webhookidempotency"
	"marketplace-backend/internal/config"
	"marketplace-backend/internal/handler"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"

	goredis "github.com/redis/go-redis/v9"
)

// subscriptionDeps captures the upstream services needed by the
// Premium subscription feature. Stripe must be configured for the
// whole subtree to come up.
type subscriptionDeps struct {
	Cfg            *config.Config
	DB             *sql.DB
	Redis          *goredis.Client
	Users          repository.UserRepository
	Stripe         service.StripeService
	PaymentInfoSvc *paymentapp.Service
	StripeHandler  *handler.StripeHandler // re-bound on output when subscription wires itself in
}

// subscriptionWiring carries the products of the Premium subscription
// feature initialisation. AppSvc + Handler stay nil when Stripe is
// not configured. StripeHandler is returned re-bound (with the
// subscription dispatcher) when the feature comes up.
type subscriptionWiring struct {
	AppSvc        *subscriptionapp.Service
	Handler       *handler.SubscriptionHandler
	StripeHandler *handler.StripeHandler // re-bound with WithSubscription when stripeHandler != nil
}

// wireSubscription brings up the Premium subscription feature: cached
// reader, fee waiver hook, handler, and the Stripe webhook dispatcher
// extension (idempotency-guarded).
//
// Wires the cached reader BEFORE the handlers because
// payment.SetSubscriptionReader must be called so subsequent milestone
// releases see the waiver. The whole block is optional: when Stripe
// is not configured the feature stays off and payment falls back to
// the full grid fee everywhere.
func wireSubscription(deps subscriptionDeps) subscriptionWiring {
	if deps.Stripe == nil {
		slog.Info("subscription feature disabled (stripe not configured)")
		return subscriptionWiring{StripeHandler: deps.StripeHandler}
	}

	cfg := deps.Cfg
	stripeSubSvc := stripeadapter.NewSubscriptionService(cfg.StripeSecretKey)
	subRepo := postgres.NewSubscriptionRepository(deps.DB)
	amountsRepo := postgres.NewProviderMilestoneAmountsRepository(deps.DB)

	appSvc := subscriptionapp.NewService(subscriptionapp.ServiceDeps{
		Subscriptions: subRepo,
		Users:         deps.Users,
		Amounts:       amountsRepo,
		Stripe:        stripeSubSvc,
		LookupKeys:    subscriptionapp.DefaultLookupKeys(),
		URLs: subscriptionapp.URLs{
			// Embedded Checkout uses a single ReturnURL; Stripe
			// substitutes the {CHECKOUT_SESSION_ID} placeholder with
			// the real id so the return page can poll
			// /subscriptions/me until the webhook flips the row to
			// active.
			CheckoutReturn: cfg.FrontendURL + "/subscribe/return?session_id={CHECKOUT_SESSION_ID}",
			PortalReturn:   cfg.FrontendURL + "/billing",
		},
	})

	// The payment feature reads Premium status through the cached
	// reader — the app service answers on cache miss, Redis serves
	// subsequent calls within 60s, and every webhook invalidates the
	// user's entry so state changes surface immediately.
	subscriptionReader := redisadapter.NewCachedSubscriptionReader(
		deps.Redis, appSvc, redisadapter.DefaultSubscriptionCacheTTL,
	)
	deps.PaymentInfoSvc.SetSubscriptionReader(subscriptionReader)

	// As soon as a Premium subscription activates, retroactively zero
	// the platform fee on every still-in-flight payment_record of the
	// org. Hook is best-effort — a failure here logs but does not
	// block the subscription from being persisted.
	appSvc.SetFeeWaiver(deps.PaymentInfoSvc)

	stripeHandler := deps.StripeHandler
	if stripeHandler != nil {
		stripeHandler = wireStripeWebhookSubscription(deps, stripeHandler, appSvc, subscriptionReader)
	}

	slog.Info("subscription feature enabled (premium plan)")
	return subscriptionWiring{
		AppSvc:        appSvc,
		Handler:       handler.NewSubscriptionHandler(appSvc),
		StripeHandler: stripeHandler,
	}
}

// wireStripeWebhookSubscription wires subscription events into the
// Stripe webhook dispatcher along with the composite idempotency
// guard that dedupes Stripe's own retry behaviour. The cache reader
// does double duty as the invalidator the dispatcher flushes on each
// state change.
//
// BUG-10 fix: the idempotency guard combines a Redis fast-path with a
// durable Postgres source of truth, so a Redis outage no longer opens
// a hole through which Stripe can replay the same event. The composite
// path is wired here; the handler treats it as a single black-box
// claimer (IdempotencyClaimer). The sub-components fail loud — when
// both layers are down the handler replies 503 so Stripe retries
// instead of silently dropping the event.
func wireStripeWebhookSubscription(
	deps subscriptionDeps,
	stripeHandler *handler.StripeHandler,
	appSvc *subscriptionapp.Service,
	reader *redisadapter.CachedSubscriptionReader,
) *handler.StripeHandler {
	cacheStore := redisadapter.NewWebhookIdempotencyStore(deps.Redis, redisadapter.DefaultWebhookIdempotencyTTL)
	durableStore := postgres.NewWebhookIdempotencyStore(deps.DB)
	compositeClaimer, claimerErr := webhookidempotencyapp.NewClaimer(durableStore, cacheStore)
	if claimerErr != nil {
		slog.Error("subscription wiring: failed to build composite webhook idempotency claimer", "error", claimerErr)
		os.Exit(1)
	}
	return stripeHandler.WithSubscription(appSvc, reader, compositeClaimer)
}
