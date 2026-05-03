package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"marketplace-backend/internal/config"
	"marketplace-backend/internal/handler/middleware"
	"marketplace-backend/internal/observability"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
)

type RouterDeps struct {
	Auth           *AuthHandler
	Invitation     *InvitationHandler
	Team           *TeamHandler
	RoleOverrides  *RoleOverridesHandler
	Profile        *ProfileHandler
	ClientProfile  *ClientProfileHandler // client-facing profile facet — optional; nil disables the /profile/client + /clients/{orgId} routes
	ProfilePricing *ProfilePricingHandler

	// Split-profile handlers (migrations 096-104). These are the
	// new persona-specific surface for provider_personal orgs; the
	// legacy Profile/ProfilePricing handlers continue to serve the
	// agency path until a follow-up refactor. Every field is
	// optional so a worktree without the split wiring can still
	// boot — only the corresponding routes are registered when the
	// pointer is non-nil.
	FreelanceProfile      *FreelanceProfileHandler
	FreelancePricing      *FreelancePricingHandler
	FreelanceProfileVideo *FreelanceProfileVideoHandler
	ReferrerProfile       *ReferrerProfileHandler
	ReferrerPricing       *ReferrerPricingHandler
	ReferrerProfileVideo  *ReferrerProfileVideoHandler
	OrganizationShared    *OrganizationSharedProfileHandler

	Upload              *UploadHandler
	Health              *HealthHandler
	Messaging           *MessagingHandler
	Proposal            *ProposalHandler
	Job                 *JobHandler
	JobApplication      *JobApplicationHandler
	Review              *ReviewHandler
	Call                *CallHandler
	SocialLink          *SocialLinkHandler // legacy agency-scoped handler
	FreelanceSocialLink *SocialLinkHandler // persona=freelance handler
	ReferrerSocialLink  *SocialLinkHandler // persona=referrer handler
	Embedded            *EmbeddedHandler
	Notification        *NotificationHandler
	Stripe              *StripeHandler
	Report              *ReportHandler
	Wallet              *WalletHandler
	Billing             *BillingHandler
	Subscription        *SubscriptionHandler
	BillingProfile      *BillingProfileHandler  // optional — nil disables /me/billing-profile routes
	Invoice             *InvoiceHandler         // optional — nil disables /me/invoices routes
	AdminCreditNote     *AdminCreditNoteHandler // optional — nil disables admin credit-note correction endpoint
	AdminInvoice        *AdminInvoiceHandler    // optional — nil disables admin "all invoices" listing + PDF redirect
	Admin               *AdminHandler
	Portfolio           *PortfolioHandler
	ProjectHistory      *ProjectHistoryHandler
	Dispute             *DisputeHandler
	AdminDispute        *AdminDisputeHandler
	Skill               *SkillHandler
	Referral            *ReferralHandler
	Search              *SearchHandler           // optional — nil when Typesense is disabled
	AdminSearchStats    *AdminSearchStatsHandler // optional — nil when Typesense is disabled
	GDPR                *GDPRHandler             // optional — nil disables /me/export + /me/account/*-deletion routes
	WSHandler           http.HandlerFunc
	Config              *config.Config
	TokenService        service.TokenService
	SessionService      service.SessionService
	// UserRepo is consumed by the Auth middleware as a
	// SessionVersionChecker — only GetSessionVersion is called. Narrowed
	// to UserAuthStore (3 methods) instead of the wide UserRepository.
	UserRepo repository.UserAuthStore

	// OrgOverridesResolver is the read port used by the Auth middleware
	// to compute each caller's effective permissions live on every
	// request — instead of trusting the snapshot baked into the session
	// at login time. Optional for backwards compat (nil in tests that
	// don't exercise permissions); production always wires it.
	OrgOverridesResolver middleware.OrgOverridesResolver

	// Metrics is optional. When non-nil, a Prometheus-format scrape
	// endpoint is exposed at GET /metrics (public, unauthenticated —
	// bind this port to an internal-only network in production).
	Metrics *Metrics

	// RateLimiter is the SEC-11 Redis-backed sliding-window limiter.
	// Optional for tests; production wiring always passes a non-nil
	// instance so the global / mutation / upload classes are enforced.
	RateLimiter *middleware.RateLimiter
}

// NewRouter assembles the chi router for the marketplace API.
//
// The body delegates each feature's route group to a `mount<Name>`
// helper that lives in a sibling file (routes_*.go). The orchestrator
// kept here is intentionally thin — every entry should read as pure
// composition: build the middleware stack, mount /api/v1 + every
// feature group inside it. Anything more belongs in a routes_*.go.
func NewRouter(deps RouterDeps) chi.Router {
	// Single auth middleware reused on every authenticated route group.
	// Constructed once here and passed down to each mount<Name> so the
	// 4-arg call site is not repeated across the package.
	auth := middleware.Auth(
		deps.TokenService,
		deps.SessionService,
		deps.UserRepo,
		deps.OrgOverridesResolver,
	)
	r := chi.NewRouter()

	mountGlobalMiddleware(r, deps)
	mountTopLevelHealth(r, deps)

	r.Route("/api/v1", func(r chi.Router) {
		mountV1Middleware(r, deps)

		mountAuthRoutes(r, deps, auth)
		mountProfileRoutes(r, deps, auth)
		mountUploadRoutes(r, deps, auth)
		mountSearchRoutes(r, deps, auth)
		mountMessagingRoutes(r, deps, auth)
		mountProposalRoutes(r, deps, auth)
		mountJobRoutes(r, deps, auth)
		mountReviewRoutes(r, deps, auth)
		mountReportRoutes(r, deps, auth)
		mountSocialLinkRoutes(r, deps, auth)
		mountPortfolioRoutes(r, deps, auth)
		mountNotificationRoutes(r, deps, auth)
		mountBillingRoutes(r, deps, auth)
		mountReferralRoutes(r, deps, auth)
		mountDisputeRoutes(r, deps, auth)
		mountGDPRRoutes(r, deps, auth)
		mountWebSocketRoute(r, deps)
		mountAdminRoutes(r, deps, auth)
		mountTestRoutes(r, deps)
	})

	mountOpenAPIRoutes(r)

	return r
}

// mountOpenAPIRoutes exposes the OpenAPI 3.1 schema describing the
// rest of the router. The handler is registered AFTER every other
// mount so chi.Walk inside ServeOpenAPIHandler sees the complete
// route tree at boot time. The schema is built once and cached for
// the lifetime of the process; consumers (web/admin/mobile generators)
// hit this endpoint to derive their typed client.
//
// Both /api/openapi.json and /api/v1/openapi.json are exposed so
// `npm run generate-api` works regardless of whether the consumer
// expects the v1-prefixed path.
func mountOpenAPIRoutes(r chi.Router) {
	handler := ServeOpenAPIHandler(r)
	r.Get("/api/openapi.json", handler)
	r.Get("/api/v1/openapi.json", handler)
}

// mountGlobalMiddleware installs the request-scoped middlewares that
// run on every endpoint regardless of API version. SecurityHeaders
// runs AFTER Recovery (so even 500s carry the headers) and BEFORE
// CORS (so OPTIONS preflights inherit them too). HSTS inside the
// middleware is gated on cfg.IsProduction() to avoid pinning localhost
// dev environments for a year.
func mountGlobalMiddleware(r chi.Router, deps RouterDeps) {
	r.Use(middleware.RequestID)
	// OpenTelemetry HTTP server middleware. When OTel is disabled
	// (default in dev / CI) the global tracer is the SDK no-op, so
	// the wrap incurs only the fixed cost of the otelhttp wrapper —
	// no spans are recorded, no exporter is dialled. Production
	// deployments with OTEL_EXPORTER_OTLP_ENDPOINT set will see one
	// server span per request, with W3C trace context propagated
	// from the caller and into the response.
	r.Use(observability.HTTPMiddleware("api.http"))
	r.Use(middleware.Logger)
	r.Use(middleware.Recovery)
	r.Use(middleware.SecurityHeaders(deps.Config))
	r.Use(middleware.CORS(deps.Config.AllowedOrigins))

	// SEC-11: global per-IP throttle (100/min). Routes that need a
	// tighter or per-user policy stack additional limiter middlewares
	// inside the route definition (mutations: 30/min/user, uploads:
	// 10/min/user). Auth-class throttling is handled by the dedicated
	// BruteForceService inside the auth handler, NOT here, so the
	// /auth/login + /auth/forgot-password endpoints have their own
	// per-email cap.
	if deps.RateLimiter != nil {
		r.Use(deps.RateLimiter.Middleware(middleware.DefaultGlobalPolicy, deps.RateLimiter.IPKey()))
	}
}

// mountTopLevelHealth registers the unversioned liveness / readiness
// probes and the Prometheus metrics scrape endpoint.
func mountTopLevelHealth(r chi.Router, deps RouterDeps) {
	r.Get("/health", deps.Health.Health)
	r.Get("/ready", deps.Health.Ready)

	// Prometheus metrics — public endpoint by design so a scrape
	// target (Prometheus, Grafana Agent, etc.) does not need
	// credentials. Bind the backend port to an internal network in
	// production OR front this path with a reverse-proxy ACL.
	if deps.Metrics != nil {
		r.Get("/metrics", deps.Metrics.Handler())
	}
}

// mountV1Middleware installs the /api/v1-scoped mutation throttle.
// The MutationOnly key short-circuits read traffic — those are
// governed by the global IP-based limiter above. Anonymous mutations
// (login, register, password reset) fall back to the client IP via
// UserOrIPKey so they still hit the 30/min cap; the brute-force
// service inside the auth handler enforces a tighter per-email cap
// on top of this.
//
// Order matters: the auth middleware that populates the user_id in
// context is installed at each route group's mount site (after this
// middleware in the chain). UserOrIPKey reads from r.Context() at
// request time, AFTER auth has run, so authenticated requests get
// the user_id key and unauthenticated ones get the IP key. This is
// the per-request state, not a build-time check.
func mountV1Middleware(r chi.Router, deps RouterDeps) {
	if deps.RateLimiter == nil {
		return
	}
	r.Use(deps.RateLimiter.Middleware(
		middleware.DefaultMutationPolicy,
		middleware.MutationOnly(middleware.UserOrIPKey(deps.RateLimiter)),
	))
}

// mountWebSocketRoute exposes the /ws upgrade endpoint. Auth is
// handled inside the handler closure (because the WebSocket
// handshake reads the token from a query string the chi auth
// middleware would not parse).
func mountWebSocketRoute(r chi.Router, deps RouterDeps) {
	if deps.WSHandler == nil {
		return
	}
	r.Get("/ws", deps.WSHandler)
}

// mountTestRoutes wires the debug endpoints used for backend & DB
// connectivity checks during development.
func mountTestRoutes(r chi.Router, deps RouterDeps) {
	r.Route("/test", func(r chi.Router) {
		r.Get("/health-check", deps.Health.HealthCheck)
		r.Get("/words", deps.Health.GetWords)
		r.Post("/words", deps.Health.AddWord)
	})
}
