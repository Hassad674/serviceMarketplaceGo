package main

import (
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
	Cfg               *config.Config
	Infra             infrastructure
	StripeSvc         service.StripeService
	StripeKYCReader   service.StripeKYCSnapshotReader
	NotifSvc          *notifapp.Service
	ProposalSvc       *proposalapp.Service
	PaymentInfoSvc    *paymentapp.Service
	ReferralSvc       *referralapp.Service
	PendingEventsRepo *postgres.PendingEventRepository
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
	})
	stripeHandler = subscription.StripeHandler
	subscriptionAppSvc := subscription.AppSvc

	var billingProfileHandler *handler.BillingProfileHandler
	var invoiceHandler *handler.InvoiceHandler
	var adminCreditNoteHandler *handler.AdminCreditNoteHandler
	var adminInvoiceHandler *handler.AdminInvoiceHandler
	if stripeHandler != nil {
		invoicing := wireInvoicing(invoicingDeps{
			DB:              deps.Infra.DB,
			Redis:           deps.Infra.Redis,
			Email:           deps.Infra.EmailSvc,
			Storage:         deps.Infra.StorageSvc,
			Organizations:   deps.Infra.OrganizationRepo,
			Users:           deps.Infra.UserRepo,
			StripeKYC:       deps.StripeKYCReader,
			StripeHandler:   stripeHandler,
			WalletHandler:   walletHandler,
			SubscriptionSvc: subscriptionAppSvc,
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

