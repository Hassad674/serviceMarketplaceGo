package main

import (
	"database/sql"
	"log/slog"

	emailadapter "marketplace-backend/internal/adapter/email"
	"marketplace-backend/internal/adapter/postgres"
	pdfadapter "marketplace-backend/internal/adapter/pdf"
	redisadapter "marketplace-backend/internal/adapter/redis"
	viesadapter "marketplace-backend/internal/adapter/vies"
	invoicingapp "marketplace-backend/internal/app/invoicing"
	subscriptionapp "marketplace-backend/internal/app/subscription"
	"marketplace-backend/internal/handler"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
	"marketplace-backend/pkg/confighelpers"

	goredis "github.com/redis/go-redis/v9"
)

// invoicingDeps captures the upstream resources needed to bring up
// the invoicing feature: the SQL pool for the Invoice / BillingProfile
// repositories, the Redis pool for the idempotency store + VIES cache,
// the email + storage adapters, and the org/user repos for the billing
// profile flows.
type invoicingDeps struct {
	DB              *sql.DB
	Redis           *goredis.Client
	Email           service.EmailService
	Storage         service.StorageService
	Organizations   repository.OrganizationRepository
	Users           repository.UserRepository
	StripeKYC       service.StripeKYCSnapshotReader
	StripeHandler   *handler.StripeHandler
	WalletHandler   *handler.WalletHandler
	SubscriptionSvc *subscriptionapp.Service
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
	invoiceRepo := postgres.NewInvoiceRepository(deps.DB).WithTxRunner(postgres.NewTxRunner(deps.DB))
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
	walletHandler := deps.WalletHandler.WithInvoicing(invoicingSvc)

	// Subscription pre-enriches the Stripe Customer with the billing
	// profile snapshot before creating an Embedded Checkout session,
	// so the inline form has nothing to re-collect. Best-effort: if
	// invoicing is disabled, the reader stays nil and Subscribe still
	// works (Stripe will simply show whatever it already has on the
	// customer).
	if deps.SubscriptionSvc != nil {
		deps.SubscriptionSvc.SetBillingProfileReader(invoicingSvc)
	}

	slog.Info("invoicing feature enabled (subscription path + me/billing-profile + me/invoices)")

	return invoicingWiring{
		BillingProfile:  handler.NewBillingProfileHandler(invoicingSvc),
		Invoice:         handler.NewInvoiceHandler(invoicingSvc),
		AdminCreditNote: handler.NewAdminCreditNoteHandler(invoicingSvc),
		AdminInvoice:    handler.NewAdminInvoiceHandler(invoicingSvc),
		StripeHandler:   stripeHandler,
		WalletHandler:   walletHandler,
	}
}
