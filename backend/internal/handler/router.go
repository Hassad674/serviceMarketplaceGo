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

	// ProfileCompletion exposes GET /api/v1/me/profile/completion. Optional —
	// nil disables the route, mirroring the rest of the handler bag.
	ProfileCompletion *ProfileCompletionHandler

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
	Receipt             *ReceiptHandler         // optional — nil disables /receipts routes
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
	Consent             *ConsentHandler          // optional — nil disables POST /consent/log
	AutomatedDecisionAppeal *AutomatedDecisionAppealHandler // optional — nil disables POST /me/automated-decision-appeals (RGPD art. 22)
	Security            *SecurityHandler         // optional — nil disables /me/security/activity route
	Sessions            *SessionsHandler         // optional — nil disables /me/sessions routes (SEC-SESSIONS)
	Stats               *StatsHandler            // optional — nil disables /me/stats/* routes
	StatsRecorder       StatsRecorder            // optional — nil disables view-tracking middleware on public profile reads
	WSHandler           http.HandlerFunc
	Config              *config.Config
	TokenService        service.TokenService
	SessionService      service.SessionService
	// UserRepo is consumed by the Auth middleware as a
	// SessionVersionChecker — only GetSessionVersion is called. Narrowed
	// to UserAuthStore (3 methods) instead of the wide UserRepository.
	//
	// PERF-AUDIT QW2: when SessionVersionChecker is wired below, the
	// middleware uses it instead of UserRepo for the per-request
	// session_version lookup. UserRepo is still kept as the default
	// fallback for tests / legacy wiring that don't pass the cache.
	UserRepo repository.UserAuthStore

	// SessionVersionChecker is the optional Redis-cached front of
	// UserRepo.GetSessionVersion. Production wiring passes a
	// CachedSessionVersionChecker so the per-request session-version
	// lookup costs one Redis GET on the hot path instead of one PG
	// round-trip. Optional — when nil the middleware falls back to
	// UserRepo, preserving the legacy behaviour for tests.
	SessionVersionChecker middleware.SessionVersionChecker

	// OrgOverridesResolver is the read port used by the Auth middleware
	// to compute each caller's effective permissions live on every
	// request — instead of trusting the snapshot baked into the session
	// at login time. Optional for backwards compat (nil in tests that
	// don't exercise permissions); production always wires it.
	OrgOverridesResolver middleware.OrgOverridesResolver

	// UserStateChecker is the read port used by the Auth middleware to
	// fetch the LIVE (is_admin, status) pair on every authenticated
	// request — overriding the snapshot baked into the session/JWT at
	// login. Without this override, an `UPDATE users SET is_admin=true`
	// issued outside the application code path (e.g. operator promotion
	// via SQL) would not propagate until each user logs out and back
	// in. Production wires this to a Redis-cached postgres reader (TTL
	// 30s); tests pass nil to keep the legacy snapshot-trust behaviour.
	UserStateChecker middleware.UserStateChecker

	// Metrics is optional. When non-nil, a Prometheus-format scrape
	// endpoint is exposed at GET /metrics (public, unauthenticated —
	// bind this port to an internal-only network in production).
	Metrics *Metrics

	// RateLimiter is the SEC-11 Redis-backed sliding-window limiter.
	// Optional for tests; production wiring always passes a non-nil
	// instance so the global / mutation / upload classes are enforced.
	RateLimiter *middleware.RateLimiter

	// IdempotencyCache backs the SEC-FINAL-02 Idempotency-Key middleware
	// on the 6 critical mutation POSTs (proposals/create + pay,
	// jobs/create, disputes/open, auth/register, team invitations).
	// Optional — nil disables idempotency wiring (tests + worktrees
	// without Redis still boot). Production always passes a non-nil
	// instance.
	IdempotencyCache middleware.IdempotencyCache
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
	// 5-arg call site is not repeated across the package.
	//
	// F.5 S8 — failClosedInProd routes a DB/Redis incident in the
	// session-version lookup to a 503 in production instead of letting
	// the middleware fall back to the JWT/cookie snapshot. Without
	// this, an attacker who triggered the upstream incident bypassed
	// permission revocation. cfg may be nil in tests; we treat that as
	// dev/test (fail-OPEN preserved).
	failClosedInProd := false
	if deps.Config != nil {
		failClosedInProd = deps.Config.IsProduction()
	}
	// PERF-AUDIT QW2: prefer the Redis-cached session-version checker
	// when wired; fall back to the raw UserRepo so existing tests that
	// don't pass the cache keep working unchanged.
	var sessionVersions middleware.SessionVersionChecker = deps.UserRepo
	if deps.SessionVersionChecker != nil {
		sessionVersions = deps.SessionVersionChecker
	}
	auth := middleware.AuthFromDeps(middleware.AuthDeps{
		TokenService:     deps.TokenService,
		SessionService:   deps.SessionService,
		SessionVersions:  sessionVersions,
		UserState:        deps.UserStateChecker,
		OrgOverrides:     deps.OrgOverridesResolver,
		FailClosedInProd: failClosedInProd,
	})
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
		mountConsentRoutes(r, deps, auth)
		mountAutomatedDecisionAppealRoutes(r, deps, auth)
		mountSecurityRoutes(r, deps, auth)
		mountSessionsRoutes(r, deps, auth)
		mountStatsRoutes(r, deps, auth)
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

	// SEC-11: global per-IP throttle. Default cap is 100 req/min as
	// documented in CLAUDE.md, but PERF-FIX-W-IDLE-CPU made it
	// env-overridable via RATE_LIMIT_GLOBAL_PER_MINUTE so a busy
	// localhost session (multiple browser tabs, polling hooks) can
	// keep iterating without tripping the cap on every poll.
	// Production keeps the default unless explicitly overridden.
	// Routes that need a tighter or per-user policy stack additional
	// limiter middlewares inside the route definition (mutations:
	// 30 req/min/user, uploads: 10 req/min/user). Auth-class
	// throttling is handled by the dedicated BruteForceService
	// inside the auth handler.
	//
	// PERF-FIX-W-IDLE-CPU: /health and /ready are exempted from the
	// throttle. They are unauthenticated, stateless probes — there
	// is no abuse vector to throttle, and a developer (or k8s
	// kube-proxy) needs them to return 200 even when /api/v1/* is
	// rate-limited.
	if deps.RateLimiter != nil {
		r.Use(deps.RateLimiter.Middleware(
			GlobalRateLimitPolicy(deps.Config),
			ExemptHealthIPKey(deps.RateLimiter),
		))
	}
}

// ExemptHealthIPKey wraps RateLimiter.IPKey() so that /health and
// /ready short-circuit the limiter (returning ok=false skips the
// Redis hop entirely). PERF-FIX-W-IDLE-CPU.
func ExemptHealthIPKey(rl *middleware.RateLimiter) func(r *http.Request) (string, bool) {
	inner := rl.IPKey()
	return func(r *http.Request) (string, bool) {
		switch r.URL.Path {
		case "/health", "/ready":
			return "", false
		}
		return inner(r)
	}
}

// GlobalRateLimitPolicy returns the per-IP throttling policy with an
// optional env override (RATE_LIMIT_GLOBAL_PER_MINUTE). cfg may be
// nil in tests; we fall back to middleware.DefaultGlobalPolicy.
// PERF-FIX-W-IDLE-CPU.
func GlobalRateLimitPolicy(cfg *config.Config) middleware.RateLimitPolicy {
	policy := middleware.DefaultGlobalPolicy
	if cfg != nil && cfg.RateLimitGlobalPerMinute > 0 {
		policy.Limit = cfg.RateLimitGlobalPerMinute
	}
	return policy
}

// MutationRateLimitPolicy mirrors GlobalRateLimitPolicy for the
// per-user mutation throttle (RATE_LIMIT_MUTATION_PER_MINUTE).
func MutationRateLimitPolicy(cfg *config.Config) middleware.RateLimitPolicy {
	policy := middleware.DefaultMutationPolicy
	if cfg != nil && cfg.RateLimitMutationPerMinute > 0 {
		policy.Limit = cfg.RateLimitMutationPerMinute
	}
	return policy
}

// UploadRateLimitPolicy mirrors the global / mutation helpers for the
// upload class (RATE_LIMIT_UPLOAD_PER_MINUTE).
func UploadRateLimitPolicy(cfg *config.Config) middleware.RateLimitPolicy {
	policy := middleware.DefaultUploadPolicy
	if cfg != nil && cfg.RateLimitUploadPerMinute > 0 {
		policy.Limit = cfg.RateLimitUploadPerMinute
	}
	return policy
}

// AuthLoginRateLimitPolicy returns the per-IP throttle that gates
// POST /auth/login. Default 10/min — see middleware.DefaultAuthLoginPolicy.
// Env override: RATE_LIMIT_AUTH_LOGIN_PER_MINUTE.
func AuthLoginRateLimitPolicy(cfg *config.Config) middleware.RateLimitPolicy {
	policy := middleware.DefaultAuthLoginPolicy
	if cfg != nil && cfg.RateLimitAuthLoginPerMinute > 0 {
		policy.Limit = cfg.RateLimitAuthLoginPerMinute
	}
	return policy
}

// Auth2FAVerifyRateLimitPolicy returns the per-IP throttle that gates
// POST /auth/login/verify-2fa. Default 10/min — see
// middleware.DefaultAuth2FAVerifyPolicy. Env override:
// RATE_LIMIT_AUTH_2FA_VERIFY_PER_MINUTE.
func Auth2FAVerifyRateLimitPolicy(cfg *config.Config) middleware.RateLimitPolicy {
	policy := middleware.DefaultAuth2FAVerifyPolicy
	if cfg != nil && cfg.RateLimitAuth2FAVerifyPerMinute > 0 {
		policy.Limit = cfg.RateLimitAuth2FAVerifyPerMinute
	}
	return policy
}

// Auth2FAEnableRateLimitPolicy returns the per-user_id throttle that
// gates POST /me/two-factor/enable. Default 5/min — see
// middleware.DefaultAuth2FAEnablePolicy. Env override:
// RATE_LIMIT_AUTH_2FA_ENABLE_PER_MINUTE.
func Auth2FAEnableRateLimitPolicy(cfg *config.Config) middleware.RateLimitPolicy {
	policy := middleware.DefaultAuth2FAEnablePolicy
	if cfg != nil && cfg.RateLimitAuth2FAEnablePerMinute > 0 {
		policy.Limit = cfg.RateLimitAuth2FAEnablePerMinute
	}
	return policy
}

// PasswordResetRateLimitPolicy returns the per-email throttle that
// gates POST /auth/forgot-password. Default 3/min — see
// middleware.DefaultPasswordResetPolicy. Env override:
// RATE_LIMIT_PASSWORD_RESET_PER_MINUTE.
func PasswordResetRateLimitPolicy(cfg *config.Config) middleware.RateLimitPolicy {
	policy := middleware.DefaultPasswordResetPolicy
	if cfg != nil && cfg.RateLimitPasswordResetPerMinute > 0 {
		policy.Limit = cfg.RateLimitPasswordResetPerMinute
	}
	return policy
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
		MutationRateLimitPolicy(deps.Config),
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
