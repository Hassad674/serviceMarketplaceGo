package main

import (
	"context"

	"marketplace-backend/internal/adapter/postgres"
	notifapp "marketplace-backend/internal/app/notification"
	paymentapp "marketplace-backend/internal/app/payment"
	proposalapp "marketplace-backend/internal/app/proposal"
	referralapp "marketplace-backend/internal/app/referral"
	subscriptionapp "marketplace-backend/internal/app/subscription"
	"marketplace-backend/internal/config"
	"marketplace-backend/internal/handler"
	"marketplace-backend/internal/port/service"
)

// billingFeatureDeps captures every input the subscription + invoicing
// + payment-handler stack needs from the broader bootstrap state.
// Defined here (not inline in bootstrap.go) so the orchestrator stays
// under the project's 600-line ceiling without losing any
// behaviour-preservation guarantees.
type billingFeatureDeps struct {
	// InvoicingCtx is the long-lived context handed to the monthly
	// invoicing scheduler. Cancelled at graceful shutdown so the
	// goroutine winds down. May be nil — wireInvoicing logs and skips
	// the scheduler when missing.
	InvoicingCtx context.Context

	Cfg               *config.Config
	Infra             infrastructure
	StripeSvc         service.StripeService
	StripeKYCReader   service.StripeKYCSnapshotReader
	NotifSvc          *notifapp.Service
	ProposalSvc       *proposalapp.Service
	// ProposalHandler is the composition facade we re-bind with the
	// invoicing gate inside wireInvoicing. Optional — when nil, the
	// proposal payment flow keeps its prior un-gated behaviour.
	ProposalHandler   *handler.ProposalHandler
	PaymentInfoSvc    *paymentapp.Service
	ReferralSvc       *referralapp.Service
	PendingEventsRepo *postgres.PendingEventRepository
	// PaymentRecordRepo is forwarded to wireInvoicing so the
	// per-milestone invoice emitter can resolve the platform fee from
	// the milestone id. Optional — when nil the per-milestone hook is
	// disabled and the monthly safety-net path still covers emission.
	PaymentRecordRepo *postgres.PaymentRecordRepository
}

// billingFeature is the bundle of handlers / services the rest of
// bootstrap stitches into the router. The pointers are nil-safe:
// when stripeHandler is nil (Stripe not configured), every invoicing
// handler stays nil and the router skips its routes.
type billingFeature struct {
	StripeHandler         *handler.StripeHandler
	WalletHandler         *handler.WalletHandler
	BillingHandler        *handler.BillingHandler
	SubscriptionHandler   *handler.SubscriptionHandler
	BillingProfileHandler *handler.BillingProfileHandler
	InvoiceHandler        *handler.InvoiceHandler
	AdminCreditNote       *handler.AdminCreditNoteHandler
	AdminInvoice          *handler.AdminInvoiceHandler
	SubscriptionAppSvc    *subscriptionapp.Service
}

// wireBillingFeatures builds the stripe handler, payment handlers,
// subscription wiring, and (when Stripe is configured) the invoicing
// suite. Mirrors the original linear block from bootstrap byte-for-
// byte, just hoisted into a named function so bootstrap.go can stay
// readable.
func wireBillingFeatures(deps billingFeatureDeps) billingFeature {
	stripeHandler := wireStripeHandler(stripeHandlerDeps{
		Cfg:               deps.Cfg,
		PaymentInfoSvc:    deps.PaymentInfoSvc,
		ProposalSvc:       deps.ProposalSvc,
		OrganizationRepo:  deps.Infra.OrganizationRepo,
		Notifications:     deps.NotifSvc,
		ReferralSvc:       deps.ReferralSvc,
		PendingEventsRepo: deps.PendingEventsRepo,
		AnalyticsSvc:      deps.Infra.AnalyticsSvc,
	})

	walletHandler, billingHandler := wirePaymentHandlers(paymentHandlersDeps{
		PaymentInfoSvc: deps.PaymentInfoSvc,
		ProposalSvc:    deps.ProposalSvc,
	})

	subscription := wireSubscription(subscriptionDeps{
		Cfg:            deps.Cfg,
		DB:             deps.Infra.DB,
		Redis:          deps.Infra.Redis,
		Users:          deps.Infra.UserRepo,
		Stripe:         deps.StripeSvc,
		PaymentInfoSvc: deps.PaymentInfoSvc,
		StripeHandler:  stripeHandler,
		Audit:          deps.Infra.AuditRepo,
	})
	stripeHandler = subscription.StripeHandler
	subscriptionAppSvc := subscription.AppSvc

	var billingProfileHandler *handler.BillingProfileHandler
	var invoiceHandler *handler.InvoiceHandler
	var adminCreditNoteHandler *handler.AdminCreditNoteHandler
	var adminInvoiceHandler *handler.AdminInvoiceHandler
	if stripeHandler != nil {
		invoicing := wireInvoicing(invoicingDeps{
			Ctx:             deps.InvoicingCtx,
			DB:              deps.Infra.DB,
			TxRunner:        deps.Infra.TxRunner,
			Redis:           deps.Infra.Redis,
			Email:           deps.Infra.EmailSvc,
			Storage:         deps.Infra.StorageSvc,
			Organizations:   deps.Infra.OrganizationRepo,
			Users:           deps.Infra.UserRepo,
			StripeKYC:       deps.StripeKYCReader,
			StripeHandler:   stripeHandler,
			WalletHandler:   walletHandler,
			ProposalHandler: deps.ProposalHandler,
			SubscriptionSvc: subscriptionAppSvc,
			ProposalSvc:     deps.ProposalSvc,
			PaymentRecords:  deps.PaymentRecordRepo,
		})
		billingProfileHandler = invoicing.BillingProfile
		invoiceHandler = invoicing.Invoice
		adminCreditNoteHandler = invoicing.AdminCreditNote
		adminInvoiceHandler = invoicing.AdminInvoice
		stripeHandler = invoicing.StripeHandler
		walletHandler = invoicing.WalletHandler
	}

	return billingFeature{
		StripeHandler:         stripeHandler,
		WalletHandler:         walletHandler,
		BillingHandler:        billingHandler,
		SubscriptionHandler:   subscription.Handler,
		BillingProfileHandler: billingProfileHandler,
		InvoiceHandler:        invoiceHandler,
		AdminCreditNote:       adminCreditNoteHandler,
		AdminInvoice:          adminInvoiceHandler,
		SubscriptionAppSvc:    subscriptionAppSvc,
	}
}

