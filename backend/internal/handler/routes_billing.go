package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"marketplace-backend/internal/domain/organization"
	"marketplace-backend/internal/handler/middleware"
)

// mountBillingRoutes wires every commerce-side surface: payment-info
// (Embedded KYC), billing fee preview, subscription lifecycle, billing
// profile, invoice listing/PDF, wallet, and the Stripe webhook + config
// endpoint. Each block is self-skipping when the corresponding handler
// is nil so the feature stays fully removable.
func mountBillingRoutes(r chi.Router, deps RouterDeps, auth func(http.Handler) http.Handler) {
	mountPaymentInfoRoutes(r, deps, auth)
	mountFeePreviewRoutes(r, deps, auth)
	mountSubscriptionRoutes(r, deps, auth)
	mountBillingProfileRoutes(r, deps, auth)
	mountInvoiceRoutes(r, deps, auth)
	mountWalletRoutes(r, deps, auth)
	mountStripeRoutes(r, deps, auth)
}

func mountPaymentInfoRoutes(r chi.Router, deps RouterDeps, auth func(http.Handler) http.Handler) {
	if deps.Embedded == nil {
		return
	}
	// Payment info routes — all served by Embedded Components now.
	r.Route("/payment-info", func(r chi.Router) {
		r.Use(auth)
		r.Use(middleware.NoCache)
		r.With(middleware.RequirePermission(organization.PermBillingView)).Get("/account-status", deps.Embedded.GetAccountStatus)
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequirePermission(organization.PermKYCManage))
			r.Post("/account-session", deps.Embedded.CreateAccountSession)
			r.Delete("/account-session", deps.Embedded.ResetAccount)
		})
	})
}

func mountFeePreviewRoutes(r chi.Router, deps RouterDeps, auth func(http.Handler) http.Handler) {
	if deps.Billing == nil {
		return
	}
	// Billing — read-only fee preview for the proposal creation flow.
	// Auth required; the role is resolved from the JWT so a client
	// cannot forge it via query string. No permission gate: every
	// authenticated user can see their own applicable fee grid.
	r.Route("/billing", func(r chi.Router) {
		r.Use(auth)
		r.Use(middleware.NoCache)
		r.Get("/fee-preview", deps.Billing.GetFeePreview)
	})
}

func mountSubscriptionRoutes(r chi.Router, deps RouterDeps, auth func(http.Handler) http.Handler) {
	if deps.Subscription == nil {
		return
	}
	// Subscription — Premium plan lifecycle endpoints. Every handler
	// requires auth; the role-based access (enterprise can't pay a
	// prestataire fee) is enforced inside the subscription service
	// when it rejects invalid plans, so no per-route role gate here.
	r.Route("/subscriptions", func(r chi.Router) {
		r.Use(auth)
		r.Use(middleware.NoCache)
		r.Post("/", deps.Subscription.Subscribe)
		r.Get("/me", deps.Subscription.GetMine)
		r.Patch("/me/auto-renew", deps.Subscription.ToggleAutoRenew)
		r.Patch("/me/billing-cycle", deps.Subscription.ChangeCycle)
		r.Get("/me/stats", deps.Subscription.GetStats)
		r.Get("/me/cycle-preview", deps.Subscription.PreviewCycleChange)
		r.Get("/portal", deps.Subscription.GetPortal)
	})
}

func mountBillingProfileRoutes(r chi.Router, deps RouterDeps, auth func(http.Handler) http.Handler) {
	if deps.BillingProfile == nil {
		return
	}
	// Invoicing — billing profile + invoices for the caller's org.
	// Mounted on a single /me prefix so the URLs stay symmetrical
	// with the rest of the org-scoped self routes. Each handler is
	// optional — nil pointer means "feature not wired" and the
	// routes simply do not exist.
	r.Route("/me/billing-profile", func(r chi.Router) {
		r.Use(auth)
		r.Use(middleware.NoCache)
		// Read is open to any org member — completeness is part
		// of the wallet/subscribe self-service UX.
		r.With(middleware.RequirePermission(organization.PermBillingView)).Get("/", deps.BillingProfile.GetMine)
		// Mutations require billing.manage so a Viewer cannot
		// edit the recipient identity that ends up on every
		// invoice.
		r.With(middleware.RequirePermission(organization.PermBillingManage)).Put("/", deps.BillingProfile.Update)
		r.With(middleware.RequirePermission(organization.PermBillingManage)).Post("/sync-from-stripe", deps.BillingProfile.SyncFromStripe)
		r.With(middleware.RequirePermission(organization.PermBillingManage)).Post("/validate-vat", deps.BillingProfile.ValidateVAT)
	})
}

func mountInvoiceRoutes(r chi.Router, deps RouterDeps, auth func(http.Handler) http.Handler) {
	if deps.Invoice == nil {
		return
	}
	r.Route("/me", func(r chi.Router) {
		r.Use(auth)
		r.Use(middleware.NoCache)
		r.Use(middleware.RequirePermission(organization.PermBillingView))
		r.Get("/invoices", deps.Invoice.List)
		r.Get("/invoices/{id}/pdf", deps.Invoice.GetPDF)
		r.Get("/invoicing/current-month", deps.Invoice.CurrentMonth)
	})
}

func mountWalletRoutes(r chi.Router, deps RouterDeps, auth func(http.Handler) http.Handler) {
	if deps.Wallet == nil {
		return
	}
	// Wallet routes (authenticated, permission-gated)
	r.Route("/wallet", func(r chi.Router) {
		r.Use(auth)
		r.Use(middleware.NoCache)
		r.With(middleware.RequirePermission(organization.PermWalletView)).Get("/", deps.Wallet.GetWallet)
		r.With(middleware.RequirePermission(organization.PermWalletWithdraw)).Post("/payout", deps.Wallet.RequestPayout)
		r.With(middleware.RequirePermission(organization.PermWalletWithdraw)).Post("/transfers/{record_id}/retry", deps.Wallet.RetryFailedTransfer)
	})
}

func mountStripeRoutes(r chi.Router, deps RouterDeps, auth func(http.Handler) http.Handler) {
	if deps.Stripe == nil {
		return
	}
	// Stripe routes
	r.Route("/stripe", func(r chi.Router) {
		r.Use(auth)
		r.Use(middleware.NoCache)
		r.With(middleware.RequirePermission(organization.PermBillingView)).Get("/config", deps.Stripe.GetConfig)
	})
	// Webhook: NO auth — Stripe sends directly, verified by signature
	r.Post("/stripe/webhook", deps.Stripe.HandleWebhook)
}
