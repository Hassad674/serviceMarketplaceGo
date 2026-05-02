package main

import (
	"time"

	"marketplace-backend/internal/adapter/nominatim"
	"marketplace-backend/internal/adapter/postgres"
	clientprofileapp "marketplace-backend/internal/app/clientprofile"
	embeddedapp "marketplace-backend/internal/app/embedded"
	notifapp "marketplace-backend/internal/app/notification"
	paymentapp "marketplace-backend/internal/app/payment"
	profileapp "marketplace-backend/internal/app/profile"
	proposalapp "marketplace-backend/internal/app/proposal"
	referralapp "marketplace-backend/internal/app/referral"
	"marketplace-backend/internal/app/searchindex"
	"marketplace-backend/internal/config"
	"marketplace-backend/internal/handler"
	"marketplace-backend/internal/port/repository"
)

// clientProfileWiring carries the products of the client-profile
// (migration 114) initialisation: the split write/read services and
// the HTTP handler ready to bind onto the router.
type clientProfileWiring struct {
	WriteSvc *profileapp.ClientProfileService
	ReadSvc  *clientprofileapp.Service
	Handler  *handler.ClientProfileHandler
}

// clientProfileDeps captures the upstream repos the client profile
// service reads from. Both write and read paths consult the legacy
// profile + organization repos; the read path also pulls proposal +
// review reads through the dedicated client-profile service.
type clientProfileDeps struct {
	ProfileRepo      repository.ProfileRepository
	OrganizationRepo repository.OrganizationRepository
	ProposalRepo     repository.ProposalRepository
	ReviewRepo       repository.ReviewRepository
}

// wireClientProfile brings up the client-profile (migration 114)
// feature — the client-facing facet of the organization's public
// profile.
//
// Two services orchestrate the feature: the write path
// (ClientProfileService, co-located with the profile aggregate) and
// the read path (clientprofile.Service, its own package). Splitting
// write vs. read keeps each service under the SRP cap and makes the
// feature fully removable by dropping these few lines.
func wireClientProfile(deps clientProfileDeps) clientProfileWiring {
	clientProfileWriteSvc := profileapp.NewClientProfileService(deps.ProfileRepo, deps.OrganizationRepo)
	clientProfileReadSvc := clientprofileapp.NewService(clientprofileapp.ServiceDeps{
		Organizations: deps.OrganizationRepo,
		Profiles:      deps.ProfileRepo,
		Proposals:     deps.ProposalRepo,
		Reviews:       deps.ReviewRepo,
	})
	clientProfileHandler := handler.NewClientProfileHandler(clientProfileWriteSvc, clientProfileReadSvc)
	return clientProfileWiring{
		WriteSvc: clientProfileWriteSvc,
		ReadSvc:  clientProfileReadSvc,
		Handler:  clientProfileHandler,
	}
}

// orgSharedDeps captures the dependencies the organization
// shared-profile handler needs.
type orgSharedDeps struct {
	OrganizationRepo *postgres.OrganizationRepository
	ProfileGeocoder  *nominatim.Geocoder
	SearchPublisher  *searchindex.Publisher
}

// wireOrganizationShared brings up the organization shared-profile
// handler — writes the photo / location / languages columns that both
// personas JOIN at read time. Reuses the optional Nominatim geocoder
// from the legacy profile flow so behaviour stays byte-identical.
func wireOrganizationShared(deps orgSharedDeps) *handler.OrganizationSharedProfileHandler {
	// Organization shared-profile handler — writes the photo /
	// location / languages columns that both personas JOIN at read
	// time. Reuses the optional Nominatim geocoder from the legacy
	// profile flow so behaviour stays byte-identical.
	h := handler.
		NewOrganizationSharedProfileHandler(deps.OrganizationRepo).
		WithGeocoder(deps.ProfileGeocoder)
	if deps.SearchPublisher != nil {
		h = h.WithSearchIndexPublisher(deps.SearchPublisher)
	}
	return h
}

// profileHandlerDeps captures the dependencies of the unified
// profile handler. The handler aggregates write access through the
// legacy profile service plus several read-only adapters used by the
// public profile page (cache + expertise + skills + pricing + client
// stats).
type profileHandlerDeps struct {
	ProfileSvc          *profileapp.Service
	ExpertiseSvc        *profileapp.ExpertiseService
	PublicProfileCache  handler.PublicProfileReader
	ExpertiseCache      handler.ExpertiseReader
	SkillsReader        handler.SkillsReader
	ProfilePricingSvc   handler.PricingReader
	ClientStatsReader   handler.ClientStatsReader
}

// wireProfileHandler builds the unified legacy profile handler with
// every cache + read adapter wired in. The fluent setters accept any
// implementation of the matching interface so the underlying caches
// + service can swap independently.
func wireProfileHandler(deps profileHandlerDeps) *handler.ProfileHandler {
	return handler.
		NewProfileHandler(deps.ProfileSvc, deps.ExpertiseSvc).
		WithPublicReader(deps.PublicProfileCache).
		WithExpertiseReader(deps.ExpertiseCache).
		WithSkillsReader(deps.SkillsReader).
		WithPricingReader(deps.ProfilePricingSvc).
		WithClientStatsReader(deps.ClientStatsReader)
}

// stripeHandlerDeps captures the upstream services the Stripe HTTP
// handler reaches into. The embedded notifier (account.* webhooks)
// fans out diff-based multi-channel notifications when org KYC
// fields change; the referral KYC listener drains parked
// pending_kyc commissions the moment the referrer becomes payable.
//
// PendingEventsRepo is the P8 async-dispatch queue: when set, the
// webhook HTTP handler enqueues onto it and replies 200 in <50ms,
// and the dispatch chain runs in a background worker (registered in
// adapter/worker/handlers/stripe_handlers.go). nil disables the
// async path and HandleWebhook falls back to inline dispatch.
type stripeHandlerDeps struct {
	Cfg               *config.Config
	PaymentInfoSvc    *paymentapp.Service
	ProposalSvc       *proposalapp.Service
	OrganizationRepo  *postgres.OrganizationRepository
	Notifications     *notifapp.Service
	ReferralSvc       *referralapp.Service
	PendingEventsRepo *postgres.PendingEventRepository
}

// wireStripeHandler builds the optional Stripe HTTP handler when
// StripeConfigured() returns true. Returns nil otherwise — the
// router has a `if x != nil` short-circuit so a nil handler simply
// omits the /stripe/* routes.
func wireStripeHandler(deps stripeHandlerDeps) *handler.StripeHandler {
	// Stripe handler (optional)
	if !deps.Cfg.StripeConfigured() {
		return nil
	}
	stripeHandler := handler.NewStripeHandler(deps.PaymentInfoSvc, deps.ProposalSvc, deps.Cfg.StripePublishableKey)

	// Embedded Components notifier — diff-based multi-channel notifications
	// for Stripe account.* webhooks (activation, requirements, docs rejected).
	// Backed by the organizations table since phase R5 — the Stripe
	// Connect account lives on the org (the merchant of record).
	embeddedNotifier := embeddedapp.NewNotifier(
		embeddedapp.NewNotificationSenderAdapter(deps.Notifications),
		deps.OrganizationRepo,
		5*time.Minute,
	)
	// Wire the referral KYC listener on the embedded notifier so parked
	// pending_kyc commissions are drained the moment the referrer's
	// Stripe account becomes payable.
	embeddedNotifier.SetReferralKYCListener(deps.ReferralSvc)
	stripeHandler = stripeHandler.WithEmbeddedNotifier(embeddedNotifier)

	// P8 — async dispatch via pending_events. With the queue wired,
	// HandleWebhook verifies the signature, enqueues a TypeStripeWebhook
	// row (ON CONFLICT (stripe_event_id) DO NOTHING for retries), and
	// replies 200 in <50ms. The dispatch chain runs in the background
	// worker registered by wirePendingEventsStripeHandler in main.go.
	if deps.PendingEventsRepo != nil {
		stripeHandler = stripeHandler.WithPendingEventsQueue(deps.PendingEventsRepo)
	}
	return stripeHandler
}

// embeddedHandlerDeps captures the dependencies the
// /api/v1/embedded entrypoint reaches into.
type embeddedHandlerDeps struct {
	OrganizationRepo *postgres.OrganizationRepository
	FrontendURL      string
}

// wireEmbeddedHandler builds the embedded entrypoint handler. Thin
// helper kept here for symmetry with the rest of the late wires.
func wireEmbeddedHandler(deps embeddedHandlerDeps) *handler.EmbeddedHandler {
	return handler.NewEmbeddedHandler(deps.OrganizationRepo, deps.FrontendURL)
}

