package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	goredis "github.com/redis/go-redis/v9"

	"marketplace-backend/internal/adapter/ws"
	"marketplace-backend/internal/app/messaging"
	"marketplace-backend/internal/config"
	"marketplace-backend/internal/handler"
	"marketplace-backend/internal/handler/middleware"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
)

// rateLimiterDeps captures the upstream resources the SEC-11 HTTP
// rate limiter needs: the Redis pool and the *config.Config (the
// trusted proxies allow-list is parsed from cfg.TrustedProxies).
type rateLimiterDeps struct {
	Cfg   *config.Config
	Redis *goredis.Client
}

// wireRateLimiter brings up the SEC-11 Redis-backed sliding-window
// rate limiter. The same instance hosts every quota class — the
// per-route policy and key extractor are passed at the route
// definition site. Calls os.Exit(1) on a malformed
// TRUSTED_PROXIES value (not recoverable at boot).
func wireRateLimiter(deps rateLimiterDeps) *middleware.RateLimiter {
	// SEC-11: Redis-backed sliding-window rate limiter. The same
	// instance hosts every quota class — the per-route policy and key
	// extractor are passed at the route definition site.
	trustedProxies, err := middleware.ParseTrustedProxies(deps.Cfg.TrustedProxies)
	if err != nil {
		slog.Error("invalid TRUSTED_PROXIES", "error", err)
		os.Exit(1)
	}
	// F.5 S7: fail-CLOSED in production. A Redis blip used to silently
	// disable throttling — the predicate switches the policy so a
	// production outage returns 503 to the client instead of leaving
	// the API unprotected.
	return middleware.NewRateLimiterWithPolicy(deps.Redis, trustedProxies, deps.Cfg.IsProduction())
}

// wsHandlerDeps captures the dependencies the WebSocket handler
// reaches into.
type wsHandlerDeps struct {
	Cfg          *config.Config
	WSHub        *ws.Hub
	MessagingSvc *messaging.Service
	TokenSvc     service.TokenService
	SessionSvc   service.SessionService
	PresenceSvc  service.PresenceService
	Broadcaster  service.MessageBroadcaster
}

// wireWSHandler builds the websocket connection handler. The
// allow-list of origins is derived from cfg.AllowedOrigins +
// localhost wildcard for dev — see wsOriginPatterns in
// wire_helpers.go.
func wireWSHandler(deps wsHandlerDeps) http.HandlerFunc {
	return ws.ServeWS(ws.ConnDeps{
		Hub:              deps.WSHub,
		MessagingSvc:     deps.MessagingSvc,
		TokenSvc:         deps.TokenSvc,
		SessionSvc:       deps.SessionSvc,
		PresenceSvc:      deps.PresenceSvc,
		Broadcaster:      deps.Broadcaster,
		AllowedWSOrigins: wsOriginPatterns(deps.Cfg.AllowedOrigins),
	})
}

// routerDepsBundle bundles every handler the router consumes. The
// struct mirrors handler.RouterDeps field-for-field so main.go can
// pass a single locally-built bundle to wireRouter without enumerating
// every handler at the call site twice.
type routerDepsBundle struct {
	Auth                  *handler.AuthHandler
	Invitation            *handler.InvitationHandler
	Team                  *handler.TeamHandler
	RoleOverrides         *handler.RoleOverridesHandler
	Profile               *handler.ProfileHandler
	ClientProfile         *handler.ClientProfileHandler
	ProfilePricing        *handler.ProfilePricingHandler
	FreelanceProfile      *handler.FreelanceProfileHandler
	FreelancePricing      *handler.FreelancePricingHandler
	FreelanceProfileVideo *handler.FreelanceProfileVideoHandler
	ReferrerProfile       *handler.ReferrerProfileHandler
	ReferrerPricing       *handler.ReferrerPricingHandler
	ReferrerProfileVideo  *handler.ReferrerProfileVideoHandler
	OrganizationShared    *handler.OrganizationSharedProfileHandler
	Upload                *handler.UploadHandler
	Health                *handler.HealthHandler
	Messaging             *handler.MessagingHandler
	Proposal              *handler.ProposalHandler
	Job                   *handler.JobHandler
	JobApplication        *handler.JobApplicationHandler
	Review                *handler.ReviewHandler
	Report                *handler.ReportHandler
	Call                  *handler.CallHandler
	SocialLink            *handler.SocialLinkHandler
	FreelanceSocialLink   *handler.SocialLinkHandler
	ReferrerSocialLink    *handler.SocialLinkHandler
	Embedded              *handler.EmbeddedHandler
	Notification          *handler.NotificationHandler
	Stripe                *handler.StripeHandler
	Wallet                *handler.WalletHandler
	Billing               *handler.BillingHandler
	Subscription          *handler.SubscriptionHandler
	BillingProfile        *handler.BillingProfileHandler
	Invoice               *handler.InvoiceHandler
	AdminCreditNote       *handler.AdminCreditNoteHandler
	AdminInvoice          *handler.AdminInvoiceHandler
	Admin                 *handler.AdminHandler
	Portfolio             *handler.PortfolioHandler
	ProjectHistory        *handler.ProjectHistoryHandler
	Dispute               *handler.DisputeHandler
	AdminDispute          *handler.AdminDisputeHandler
	GDPR                  *handler.GDPRHandler
	Skill                 *handler.SkillHandler
	Referral              *handler.ReferralHandler
	Search                *handler.SearchHandler
	AdminSearchStats      *handler.AdminSearchStatsHandler
	WSHandler             http.HandlerFunc
	Cfg                   *config.Config
	TokenService          service.TokenService
	SessionService        service.SessionService
	UserRepo              repository.UserRepository
	OrgOverridesResolver  middleware.OrgOverridesResolver
	Metrics               *handler.Metrics
	RateLimiter           *middleware.RateLimiter
	IdempotencyCache      middleware.IdempotencyCache
}

// wireRouter forwards the bundled handlers to handler.NewRouter. A
// thin facade kept so the RouterDeps literal lives outside main.go
// — the field map is enormous and re-listing every binding inline
// makes the orchestration intent in main.go unreadable.
func wireRouter(b routerDepsBundle) chi.Router {
	return handler.NewRouter(handler.RouterDeps{
		Auth:           b.Auth,
		Invitation:     b.Invitation,
		Team:           b.Team,
		RoleOverrides:  b.RoleOverrides,
		Profile:        b.Profile,
		ClientProfile:  b.ClientProfile,
		ProfilePricing: b.ProfilePricing,

		// Split-profile handlers (migrations 096-104).
		FreelanceProfile:      b.FreelanceProfile,
		FreelancePricing:      b.FreelancePricing,
		FreelanceProfileVideo: b.FreelanceProfileVideo,
		ReferrerProfile:       b.ReferrerProfile,
		ReferrerPricing:       b.ReferrerPricing,
		ReferrerProfileVideo:  b.ReferrerProfileVideo,
		OrganizationShared:    b.OrganizationShared,

		Upload:               b.Upload,
		Health:               b.Health,
		Messaging:            b.Messaging,
		Proposal:             b.Proposal,
		Job:                  b.Job,
		JobApplication:       b.JobApplication,
		Review:               b.Review,
		Report:               b.Report,
		Call:                 b.Call,
		SocialLink:           b.SocialLink,
		FreelanceSocialLink:  b.FreelanceSocialLink,
		ReferrerSocialLink:   b.ReferrerSocialLink,
		Embedded:             b.Embedded,
		Notification:         b.Notification,
		Stripe:               b.Stripe,
		Wallet:               b.Wallet,
		Billing:              b.Billing,
		Subscription:         b.Subscription,
		BillingProfile:       b.BillingProfile,
		Invoice:              b.Invoice,
		AdminCreditNote:      b.AdminCreditNote,
		AdminInvoice:         b.AdminInvoice,
		Admin:                b.Admin,
		Portfolio:            b.Portfolio,
		ProjectHistory:       b.ProjectHistory,
		Dispute:              b.Dispute,
		AdminDispute:         b.AdminDispute,
		GDPR:                 b.GDPR,
		Skill:                b.Skill,
		Referral:             b.Referral,
		Search:               b.Search,
		AdminSearchStats:     b.AdminSearchStats,
		WSHandler:            b.WSHandler,
		Config:               b.Cfg,
		TokenService:         b.TokenService,
		SessionService:       b.SessionService,
		UserRepo:             b.UserRepo,
		OrgOverridesResolver: b.OrgOverridesResolver,
		Metrics:              b.Metrics,
		RateLimiter:          b.RateLimiter,
		IdempotencyCache:     b.IdempotencyCache,
	})
}
