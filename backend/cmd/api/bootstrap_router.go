package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"

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
		AdminCreditNote:       h.AdminCreditNote,
		AdminInvoice:          h.AdminInvoice,
		Admin:                 h.Admin,
		Portfolio:             h.Portfolio,
		ProjectHistory:        h.ProjectHistory,
		Dispute:               h.Dispute,
		AdminDispute:          h.AdminDispute,
		GDPR:                  h.GDPR,
		Skill:                 h.Skill,
		Referral:              h.Referral,
		Search:                h.Search,
		AdminSearchStats:      h.AdminSearchStats,
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
		AdminCreditNote:       b.Handlers.AdminCreditNote,
		AdminInvoice:          b.Handlers.AdminInvoice,
		Admin:                 b.Handlers.Admin,
		Portfolio:             b.Handlers.Portfolio,
		ProjectHistory:        b.Handlers.ProjectHistory,
		Dispute:               b.Handlers.Dispute,
		AdminDispute:          b.Handlers.AdminDispute,
		GDPR:                  b.Handlers.GDPR,
		Skill:                 b.Handlers.Skill,
		Referral:              b.Handlers.Referral,
		Search:                b.Handlers.Search,
		AdminSearchStats:      b.Handlers.AdminSearchStats,
		WSHandler:             b.WSHandler,
		Cfg:                   b.Cfg,
		TokenService:          b.Infra.TokenSvc,
		SessionService:        b.Infra.SessionSvc,
		UserRepo:              b.Infra.UserRepo,
		OrgOverridesResolver:  orgOverridesAdapter{repo: b.Infra.OrganizationRepo},
		Metrics:               b.Metrics,
		RateLimiter:           b.RateLimiter,
		// SEC-FINAL-02: idempotency middleware on the 6 critical
		// mutation POSTs (proposals create + pay, jobs create,
		// disputes open, auth/register, team invitations).
		IdempotencyCache: middleware.NewRedisIdempotencyCache(b.Infra.Redis),
	})
}
