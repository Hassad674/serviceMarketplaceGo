package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	redisadapter "marketplace-backend/internal/adapter/redis"
	"marketplace-backend/internal/config"
	"marketplace-backend/internal/handler"
	"marketplace-backend/internal/handler/middleware"
)

// routerHandlers groups every handler bootstrap built so it can pass
// them to assembleRouter as a single value. Keeps the bootstrap
// function under the project's 600-line ceiling without changing
// behaviour: every field below is a 1:1 copy of a local variable
// from bootstrap and is forwarded verbatim into routerDepsBundle.
type routerHandlers struct {
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
	ProfileCompletion     *handler.ProfileCompletionHandler
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
	Receipt               *handler.ReceiptHandler
	AdminCreditNote       *handler.AdminCreditNoteHandler
	AdminInvoice          *handler.AdminInvoiceHandler
	Admin                 *handler.AdminHandler
	Portfolio             *handler.PortfolioHandler
	ProjectHistory        *handler.ProjectHistoryHandler
	Dispute               *handler.DisputeHandler
	AdminDispute          *handler.AdminDisputeHandler
	GDPR                  *handler.GDPRHandler
	Consent               *handler.ConsentHandler
	AutomatedDecisionAppeal *handler.AutomatedDecisionAppealHandler
	Security              *handler.SecurityHandler
	Sessions              *handler.SessionsHandler
	Skill                 *handler.SkillHandler
	Referral              *handler.ReferralHandler
	Search                *handler.SearchHandler
	AdminSearchStats      *handler.AdminSearchStatsHandler
	Stats                 *handler.StatsHandler
	StatsRecorder         handler.StatsRecorder
}

// bootstrappedRouter bundles everything assembleRouter needs.
type bootstrappedRouter struct {
	Handlers    routerHandlers
	WSHandler   http.HandlerFunc
	Cfg         *config.Config
	Infra       infrastructure
	Metrics     *handler.Metrics
	RateLimiter *middleware.RateLimiter
}

// finalHandlers is the bag of bootstrap-local handlers + small wires
// the routerHandlers builder consumes. Split out so bootstrap.go
// stays under the 600-line ceiling — every field is forwarded
// verbatim into routerHandlers without transformation.
type finalHandlers struct {
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
	ProfileCompletion     *handler.ProfileCompletionHandler
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
	Receipt               *handler.ReceiptHandler
	AdminCreditNote       *handler.AdminCreditNoteHandler
	AdminInvoice          *handler.AdminInvoiceHandler
	Admin                 *handler.AdminHandler
	Portfolio             *handler.PortfolioHandler
	ProjectHistory        *handler.ProjectHistoryHandler
	Dispute               *handler.DisputeHandler
	AdminDispute          *handler.AdminDisputeHandler
	GDPR                  *handler.GDPRHandler
	Consent               *handler.ConsentHandler
	AutomatedDecisionAppeal *handler.AutomatedDecisionAppealHandler
	Security              *handler.SecurityHandler
	Sessions              *handler.SessionsHandler
	Skill                 *handler.SkillHandler
	Referral              *handler.ReferralHandler
	Search                *handler.SearchHandler
	AdminSearchStats      *handler.AdminSearchStatsHandler
	Stats                 *handler.StatsHandler
	StatsRecorder         handler.StatsRecorder
}

// buildRouterHandlers copies a finalHandlers value into the
// routerHandlers shape assembleRouter expects. Trivial 1:1 copy —
// kept as a function so the caller in bootstrap.go can pass a
// single struct + receive a single struct without enumerating 50
// inline bindings.
func buildRouterHandlers(h finalHandlers) routerHandlers {
	return routerHandlers{
		Auth:                  h.Auth,
		Invitation:            h.Invitation,
		Team:                  h.Team,
		RoleOverrides:         h.RoleOverrides,
		Profile:               h.Profile,
		ClientProfile:         h.ClientProfile,
		ProfilePricing:        h.ProfilePricing,
		FreelanceProfile:      h.FreelanceProfile,
		FreelancePricing:      h.FreelancePricing,
		FreelanceProfileVideo: h.FreelanceProfileVideo,
		ReferrerProfile:       h.ReferrerProfile,
		ReferrerPricing:       h.ReferrerPricing,
		ReferrerProfileVideo:  h.ReferrerProfileVideo,
		OrganizationShared:    h.OrganizationShared,
		ProfileCompletion:     h.ProfileCompletion,
		Upload:                h.Upload,
		Health:                h.Health,
		Messaging:             h.Messaging,
		Proposal:              h.Proposal,
		Job:                   h.Job,
		JobApplication:        h.JobApplication,
		Review:                h.Review,
		Report:                h.Report,
		Call:                  h.Call,
		SocialLink:            h.SocialLink,
		FreelanceSocialLink:   h.FreelanceSocialLink,
		ReferrerSocialLink:    h.ReferrerSocialLink,
		Embedded:              h.Embedded,
		Notification:          h.Notification,
		Stripe:                h.Stripe,
		Wallet:                h.Wallet,
		Billing:               h.Billing,
		Subscription:          h.Subscription,
		BillingProfile:        h.BillingProfile,
		Invoice:               h.Invoice,
		Receipt:               h.Receipt,
		AdminCreditNote:       h.AdminCreditNote,
		AdminInvoice:          h.AdminInvoice,
		Admin:                 h.Admin,
		Portfolio:             h.Portfolio,
		ProjectHistory:        h.ProjectHistory,
		Dispute:               h.Dispute,
		AdminDispute:          h.AdminDispute,
		GDPR:                    h.GDPR,
		Consent:                 h.Consent,
		AutomatedDecisionAppeal: h.AutomatedDecisionAppeal,
		Security:                h.Security,
		Sessions:                h.Sessions,
		Skill:                 h.Skill,
		Referral:              h.Referral,
		Search:                h.Search,
		AdminSearchStats:      h.AdminSearchStats,
		Stats:                 h.Stats,
		StatsRecorder:         h.StatsRecorder,
	}
}

// assembleRouter composes the chi router from the bootstrapped
// handlers + infra. Extracted from bootstrap so the orchestration
// function stays under the project's 600-line ceiling; behaviour is
// byte-identical with the original inline call.
func assembleRouter(b bootstrappedRouter) chi.Router {
	return wireRouter(routerDepsBundle{
		Auth:                  b.Handlers.Auth,
		Invitation:            b.Handlers.Invitation,
		Team:                  b.Handlers.Team,
		RoleOverrides:         b.Handlers.RoleOverrides,
		Profile:               b.Handlers.Profile,
		ClientProfile:         b.Handlers.ClientProfile,
		ProfilePricing:        b.Handlers.ProfilePricing,
		FreelanceProfile:      b.Handlers.FreelanceProfile,
		FreelancePricing:      b.Handlers.FreelancePricing,
		FreelanceProfileVideo: b.Handlers.FreelanceProfileVideo,
		ReferrerProfile:       b.Handlers.ReferrerProfile,
		ReferrerPricing:       b.Handlers.ReferrerPricing,
		ReferrerProfileVideo:  b.Handlers.ReferrerProfileVideo,
		OrganizationShared:    b.Handlers.OrganizationShared,
		ProfileCompletion:     b.Handlers.ProfileCompletion,
		Upload:                b.Handlers.Upload,
		Health:                b.Handlers.Health,
		Messaging:             b.Handlers.Messaging,
		Proposal:              b.Handlers.Proposal,
		Job:                   b.Handlers.Job,
		JobApplication:        b.Handlers.JobApplication,
		Review:                b.Handlers.Review,
		Report:                b.Handlers.Report,
		Call:                  b.Handlers.Call,
		SocialLink:            b.Handlers.SocialLink,
		FreelanceSocialLink:   b.Handlers.FreelanceSocialLink,
		ReferrerSocialLink:    b.Handlers.ReferrerSocialLink,
		Embedded:              b.Handlers.Embedded,
		Notification:          b.Handlers.Notification,
		Stripe:                b.Handlers.Stripe,
		Wallet:                b.Handlers.Wallet,
		Billing:               b.Handlers.Billing,
		Subscription:          b.Handlers.Subscription,
		BillingProfile:        b.Handlers.BillingProfile,
		Invoice:               b.Handlers.Invoice,
		Receipt:               b.Handlers.Receipt,
		AdminCreditNote:       b.Handlers.AdminCreditNote,
		AdminInvoice:          b.Handlers.AdminInvoice,
		Admin:                 b.Handlers.Admin,
		Portfolio:             b.Handlers.Portfolio,
		ProjectHistory:        b.Handlers.ProjectHistory,
		Dispute:               b.Handlers.Dispute,
		AdminDispute:          b.Handlers.AdminDispute,
		GDPR:                    b.Handlers.GDPR,
		Consent:                 b.Handlers.Consent,
		AutomatedDecisionAppeal: b.Handlers.AutomatedDecisionAppeal,
		Security:                b.Handlers.Security,
		Sessions:                b.Handlers.Sessions,
		Skill:                 b.Handlers.Skill,
		Referral:              b.Handlers.Referral,
		Search:                b.Handlers.Search,
		AdminSearchStats:      b.Handlers.AdminSearchStats,
		Stats:                 b.Handlers.Stats,
		StatsRecorder:         b.Handlers.StatsRecorder,
		WSHandler:             b.WSHandler,
		Cfg:                   b.Cfg,
		TokenService:          b.Infra.TokenSvc,
		SessionService:        b.Infra.SessionSvc,
		UserRepo: b.Infra.UserRepo,
		// PERF-AUDIT QW1: the role-overrides resolver is consulted on
		// every authenticated request to compute the caller's effective
		// permissions live. Without the cache, every request hit
		// Postgres for the full organizations row (~10-15 ms RTT on
		// Neon + ~2-3 ms planning). The 30s Redis cache cuts that to a
		// single Redis GET — the role-permissions editor explicitly
		// Invalidates the cache after a write so the propagation lag
		// stays bounded to "next request" for live edits and to the
		// TTL for direct-SQL operator edits.
		//
		// QW-HARDENING: the cache is now constructed in
		// wire_infra.go so the role-overrides app service can reach
		// the same instance via b.Infra.OrgOverridesCache and call
		// Invalidate after a SaveRoleOverrides write — closing the
		// 30s revocation window the original wiring left open.
		OrgOverridesResolver: b.Infra.OrgOverridesCache,
		// is_admin propagation fix — the auth middleware reads the live
		// (is_admin, status) pair on every request and overrides the
		// session/JWT snapshot. Postgres adapter behind a 30s Redis
		// cache so the per-request DB cost is amortised.
		UserStateChecker: redisadapter.NewCachedUserStateChecker(
			b.Infra.Redis,
			userStateAdapter{repo: b.Infra.UserRepo},
			redisadapter.DefaultUserStateCacheTTL,
		),
		// PERF-AUDIT QW2: session-version checker behind a 30s Redis
		// cache. Without it, every authenticated request paid a fresh
		// SELECT session_version FROM users WHERE id = $1 against
		// Postgres (~10-15 ms RTT on Neon). With the cache, the
		// per-request cost collapses to one Redis GET. Revocation
		// paths (BumpSessionVersion / logout-all) must call
		// Invalidate on the cache so the new version propagates
		// immediately instead of waiting for the TTL.
		//
		// QW-HARDENING: the cache is now constructed in
		// wire_infra.go so the InvalidatingUserRepo wrapper (passed
		// to every service that calls BumpSessionVersion) can call
		// Invalidate on the same instance after each successful bump
		// — closing the 30s revocation window.
		SessionVersionChecker: b.Infra.SessionVersionCache,
		Metrics:     b.Metrics,
		RateLimiter: b.RateLimiter,
		// SEC-FINAL-02: idempotency middleware on the 6 critical
		// mutation POSTs (proposals create + pay, jobs create,
		// disputes open, auth/register, team invitations).
		IdempotencyCache: middleware.NewRedisIdempotencyCache(b.Infra.Redis),
	})
}
