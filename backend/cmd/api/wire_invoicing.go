package main

import (
	"context"
	"database/sql"
	"log/slog"

	emailadapter "marketplace-backend/internal/adapter/email"
	"marketplace-backend/internal/adapter/postgres"
	pdfadapter "marketplace-backend/internal/adapter/pdf"
	redisadapter "marketplace-backend/internal/adapter/redis"
	viesadapter "marketplace-backend/internal/adapter/vies"
	invoicingapp "marketplace-backend/internal/app/invoicing"
	proposalapp "marketplace-backend/internal/app/proposal"
	subscriptionapp "marketplace-backend/internal/app/subscription"
	"marketplace-backend/internal/handler"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
	"marketplace-backend/internal/system"
	"marketplace-backend/pkg/confighelpers"

	"github.com/google/uuid"

	goredis "github.com/redis/go-redis/v9"
)

// invoicingDeps captures the upstream resources needed to bring up
// the invoicing feature: the SQL pool for the Invoice / BillingProfile
// repositories, the Redis pool for the idempotency store + VIES cache,
// the email + storage adapters, and the org/user repos for the billing
// profile flows.
type invoicingDeps struct {
	// Ctx is the long-lived context used by the monthly-consolidation
	// scheduler goroutine. Cancelled by the graceful-shutdown sequence
	// so the goroutine winds down deterministically. The caller must
	// pass a derived ctx that is added to the WorkerCancels bag.
	Ctx context.Context

	DB *sql.DB
	// TxRunner is the routed transaction runner from
	// wireInfrastructure. Forwarded to the InvoiceRepository so
	// invoice writes use the right pool.
	TxRunner        *postgres.TxRunner
	Redis           *goredis.Client
	Email           service.EmailService
	Storage         service.StorageService
	Organizations   repository.OrganizationRepository
	Users           repository.UserRepository
	StripeKYC       service.StripeKYCSnapshotReader
	StripeHandler   *handler.StripeHandler
	WalletHandler   *handler.WalletHandler
	// ProposalHandler carries the proposal payment sub-handler that
	// gets re-bound with the billing-profile gate. Optional — when
	// nil the gate is skipped and the proposal payment flow keeps
	// its prior fail-open behaviour.
	ProposalHandler *handler.ProposalHandler
	SubscriptionSvc *subscriptionapp.Service
	// ProposalSvc receives the per-milestone invoicer adapter so each
	// milestone approval emits a platform_fee invoice synchronously.
	// Optional — when nil the proposal flow stays on the legacy monthly
	// consolidation path.
	ProposalSvc *proposalapp.Service
	// PaymentRecords resolves the payment record from a milestone id —
	// fed to the per-milestone invoicer adapter so the proposal app
	// service never has to import the payment package directly.
	PaymentRecords repository.PaymentRecordRepository
}

// invoicingWiring carries the products of the invoicing feature
// initialisation. Every field stays nil when invoicing is disabled
// (Stripe absent, issuer config invalid, or PDF renderer init fails)
// so the router can keep its `if x != nil` short-circuits in place.
//
// StripeHandler / WalletHandler are returned re-bound: invoicing wires
// itself into both handlers via fluent setters that return new pointer
// values; we must thread those back to main.go so the router uses the
// invoicing-aware variants.
type invoicingWiring struct {
	BillingProfile  *handler.BillingProfileHandler
	Invoice         *handler.InvoiceHandler
	AdminCreditNote *handler.AdminCreditNoteHandler
	AdminInvoice    *handler.AdminInvoiceHandler
	StripeHandler   *handler.StripeHandler // re-bound with WithInvoicing
	WalletHandler   *handler.WalletHandler // re-bound with WithInvoicing
}

// wireInvoicing brings up the invoicing feature: outbound
// customer-facing invoices for successful subscription payments
// (monthly commission consolidation lives in a follow-up phase). The
// whole block is optional: if any of the issuer env vars are missing
// or the PDF renderer can't initialise, we log + skip and the rest
// of the backend boots.
//
// Pre-condition: deps.StripeHandler must be non-nil. The caller must
// short-circuit on Stripe being un-configured before reaching this
// helper — invoicing has no fallback path without Stripe.
func wireInvoicing(deps invoicingDeps) invoicingWiring {
	zero := invoicingWiring{
		StripeHandler: deps.StripeHandler,
		WalletHandler: deps.WalletHandler,
	}

	issuer, issuerErr := confighelpers.LoadInvoiceIssuer()
	if issuerErr != nil {
		slog.Warn("invoicing feature disabled (issuer config invalid)", "error", issuerErr)
		return zero
	}
	pdfRenderer, pdfErr := pdfadapter.New()
	if pdfErr != nil {
		slog.Warn("invoicing feature disabled (pdf renderer init failed)", "error", pdfErr)
		return zero
	}

	// BUG-NEW-04 path 3/8: invoice is RLS-protected by migration
	// 125 (USING recipient_organization_id = current_setting(
	// 'app.current_org_id', true)). The txRunner wrap makes
	// CreateInvoice / MarkInvoiceCredited / ListInvoicesByOrganization
	// / FindInvoiceByIDForOrg pass under prod NOSUPERUSER NOBYPASSRLS.
	// Stripe webhook lookups (idempotency by stripe_event_id)
	// stay on the legacy direct-db path — production must keep
	// that handler on a privileged DB role. Documented in the
	// repo docstring.
	invoicingTxRunner := deps.TxRunner
	if invoicingTxRunner == nil {
		invoicingTxRunner = postgres.NewTxRunner(deps.DB)
	}
	invoiceRepo := postgres.NewInvoiceRepository(deps.DB).WithTxRunner(invoicingTxRunner)
	billingProfileRepo := postgres.NewBillingProfileRepository(deps.DB)
	invoiceDeliverer := emailadapter.NewDeliverer(deps.Email)
	invoiceIdempotency := redisadapter.NewWebhookIdempotencyStore(
		deps.Redis,
		redisadapter.DefaultWebhookIdempotencyTTL,
	)

	invoicingSvc := invoicingapp.NewService(invoicingapp.ServiceDeps{
		Invoices:    invoiceRepo,
		Profiles:    billingProfileRepo,
		PDF:         pdfRenderer,
		Storage:     deps.Storage,
		Deliverer:   invoiceDeliverer,
		Issuer:      issuer,
		Idempotency: invoiceIdempotency,
	})

	// Phase 6 — wire optional dependencies for the /me/billing-profile
	// flows: stripe KYC pre-fill + VIES validation. Both are best-effort;
	// missing config simply disables the corresponding endpoint.
	viesValidator := viesadapter.NewClient(deps.Redis)
	invoicingSvc.SetBillingProfileDeps(invoicingapp.BillingProfileDeps{
		Organizations: deps.Organizations,
		Users:         deps.Users,
		StripeKYC:     deps.StripeKYC,
		VIESValidator: viesValidator,
	})

	stripeHandler := deps.StripeHandler.WithInvoicing(invoicingSvc)
	// Wire the user → organization resolver so the inline-billing-capture
	// pipeline (payment_intent.succeeded → hydrate billing_profile) can
	// translate the payment record's ClientID (a user_id) into the
	// org_id that owns the billing identity. Same closure shape used by
	// the receipt snapshot resolver — keeps the dependency surface
	// narrow and the wiring identical.
	stripeHandler = stripeHandler.WithUserOrgResolver(handler.UserOrgResolver(userOrgReaderAdapter(deps.Users)))
	walletHandler := deps.WalletHandler.WithInvoicing(invoicingSvc)

	// Proposal payment gate: a client paying a proposal must have a
	// complete billing profile so the receipt snapshot (PR #165)
	// carries the legally required recipient identity. Re-bind the
	// proposal payment sub-handler with the invoicing service the
	// same way we re-bind the wallet handler — fluent setter, no
	// constructor changes. The proposal handler facade keeps a
	// pointer to the same payment sub-handler so the in-place
	// mutation propagates to every router wiring path (including
	// the legacy ProposalHandler facade methods).
	if deps.ProposalHandler != nil {
		deps.ProposalHandler.Payment().WithInvoicing(invoicingSvc)
	}

	// Subscription pre-enriches the Stripe Customer with the billing
	// profile snapshot before creating an Embedded Checkout session,
	// so the inline form has nothing to re-collect. Best-effort: if
	// invoicing is disabled, the reader stays nil and Subscribe still
	// works (Stripe will simply show whatever it already has on the
	// customer).
	if deps.SubscriptionSvc != nil {
		deps.SubscriptionSvc.SetBillingProfileReader(invoicingSvc)
	}

	// Per-milestone platform_fee invoicer — every milestone approval
	// emits a platform_fee invoice synchronously through the proposal
	// service. Best-effort: nil deps fall back to the legacy
	// monthly-consolidation flow.
	if deps.ProposalSvc != nil && deps.PaymentRecords != nil && deps.Organizations != nil {
		orgsReader := orgsOfUserAdapter{orgs: deps.Organizations}
		adapter := invoicingapp.NewPerMilestoneInvoicerAdapter(invoicingSvc, deps.PaymentRecords, orgsReader)
		deps.ProposalSvc.SetPerMilestoneInvoicer(adapter)
		slog.Info("invoicing per-milestone emitter wired into proposal service")
	} else {
		slog.Warn("invoicing per-milestone emitter NOT wired",
			"has_proposal_svc", deps.ProposalSvc != nil,
			"has_payment_records", deps.PaymentRecords != nil,
			"has_organizations", deps.Organizations != nil)
	}

	// Monthly-consolidation scheduler — issues the consolidated
	// commission invoice on the 1st of each month between 02:00 and
	// 04:00 UTC. Runs in-process: V1 ships with a single API instance
	// and the Redis-backed RunMarker is idempotent on retry, so no
	// distributed coordination is required. The CLI binary
	// `cmd/invoice-monthly-run` remains the manual fallback for ad-hoc
	// re-runs and Phase 4 backfill.
	//
	// The scheduler's tick reaches into ListReleasedPaymentRecordsForOrg,
	// which warns when ctx is not tagged as a system actor (RLS bypass);
	// tag the goroutine context here so the warning never fires.
	if deps.Ctx != nil && deps.Organizations != nil && deps.Redis != nil {
		runMarker := redisadapter.NewRunMarker(deps.Redis, redisadapter.DefaultInvoicingRunMarkerTTL)
		scheduler := invoicingapp.NewScheduler(invoicingapp.SchedulerDeps{
			Service: invoicingSvc,
			Orgs:    deps.Organizations,
			Marker:  runMarker,
		})
		scheduler.Start(system.WithSystemActor(deps.Ctx))
		slog.Info("invoicing scheduler started",
			"interval", invoicingapp.DefaultSchedulerInterval,
			"window", "day=1, 02:00-04:00 UTC")
	} else {
		slog.Warn("invoicing scheduler not started — missing ctx / orgs / redis dep",
			"has_ctx", deps.Ctx != nil,
			"has_orgs", deps.Organizations != nil,
			"has_redis", deps.Redis != nil)
	}

	slog.Info("invoicing feature enabled (subscription path + me/billing-profile + me/invoices + monthly scheduler)")

	return invoicingWiring{
		BillingProfile:  handler.NewBillingProfileHandler(invoicingSvc),
		Invoice:         handler.NewInvoiceHandler(invoicingSvc),
		AdminCreditNote: handler.NewAdminCreditNoteHandler(invoicingSvc),
		AdminInvoice:    handler.NewAdminInvoiceHandler(invoicingSvc),
		StripeHandler:   stripeHandler,
		WalletHandler:   walletHandler,
	}
}

// orgsOfUserAdapter satisfies invoicingapp.OrganizationOfUserReader by
// reaching into the wider OrganizationRepository.FindByUserID method
// and unwrapping the returned organization into its bare id. Keeps the
// invoicing app port narrow without forcing it to know the full org
// domain type.
type orgsOfUserAdapter struct {
	orgs repository.OrganizationRepository
}

// FindByUserID implements invoicingapp.OrganizationOfUserReader.
func (a orgsOfUserAdapter) FindByUserID(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	org, err := a.orgs.FindByUserID(ctx, userID)
	if err != nil {
		return uuid.Nil, err
	}
	if org == nil {
		return uuid.Nil, nil
	}
	return org.ID, nil
}
