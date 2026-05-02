package payment

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/billing"
	domain "marketplace-backend/internal/domain/payment"
	proposaldomain "marketplace-backend/internal/domain/proposal"
	domainuser "marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/port/repository"
	portservice "marketplace-backend/internal/port/service"
)

// WalletService is the read-side of the payment feature: wallet
// overview, fee preview, payment-record reads, and the platform-fee
// calculation that the charge service shares.
//
// SRP rationale: every method here is a query that does not mutate
// payment records. The mutation paths (charge / payout) live on
// ChargeService and PayoutService respectively. This split makes the
// dependency surface explicit:
//
//   - Wallet only needs reads from records / users / orgs / Stripe
//     (account capabilities), plus the optional referral-wallet hook
//     for apporteur commission rendering and the optional subscription
//     reader for the Premium fee waiver.
//
//   - Wallet does NOT need: stripe.CreateTransfer, stripe.CreateRefund,
//     stripe.CreatePayout, the proposal-status reader, or the referral
//     distributor / clawback. Those belong to PayoutService /
//     ChargeService.
//
// The legacy Service struct (service.go) embeds *WalletService and
// delegates every wallet method to it, so existing call sites compile
// unchanged.
type WalletService struct {
	records repository.PaymentRecordRepository
	users   repository.UserRepository
	// orgs is narrowed to the Stripe-store child — wallet only ever
	// reads the connected-account id off the organization row.
	orgs   repository.OrganizationStripeStore
	stripe portservice.StripeService

	// referralWallet renders the apporteur side of the wallet (commission
	// totals + recent commissions). Nil when the referral feature is not
	// active in the deployment — the rendering of the commission section
	// degrades to "no data" without erroring.
	referralWallet portservice.ReferralWalletReader

	// subscriptions waives the platform fee on PreviewFee and on
	// computePlatformFee (called by the charge service). Nil when the
	// subscription feature is not wired — every user is then quoted /
	// billed at the full grid rate.
	subscriptions portservice.SubscriptionReader
}

// WalletServiceDeps groups every dependency NewWalletService needs.
//
// Organizations is narrowed to OrganizationStripeStore — the wallet
// sub-service only reads stripe_account_id off the org row.
type WalletServiceDeps struct {
	Records       repository.PaymentRecordRepository
	Users         repository.UserRepository
	Organizations repository.OrganizationStripeStore
	Stripe        portservice.StripeService
}

// NewWalletService wires the read-only wallet sub-service. Optional
// dependencies (referral wallet reader, subscription reader) are wired
// post-construction via setters on the parent Service so the wallet sub-
// service stays bootable in worktrees that don't have those features
// enabled.
func NewWalletService(deps WalletServiceDeps) *WalletService {
	return &WalletService{
		records: deps.Records,
		users:   deps.Users,
		orgs:    deps.Organizations,
		stripe:  deps.Stripe,
	}
}

// SetReferralWalletReader plugs the apporteur commission read path into
// the wallet overview.
func (w *WalletService) SetReferralWalletReader(r portservice.ReferralWalletReader) {
	w.referralWallet = r
}

// SetSubscriptionReader plugs the Premium subscription lookup used by
// PreviewFee and computePlatformFee.
func (w *WalletService) SetSubscriptionReader(r portservice.SubscriptionReader) {
	w.subscriptions = r
}

// GetWalletOverview returns the organization's wallet state. Every
// member of the same org sees the same wallet (Stripe Dashboard model).
// Since phase R5 the Stripe account + KYC bookkeeping live on the org.
func (w *WalletService) GetWalletOverview(ctx context.Context, userID, orgID uuid.UUID) (*WalletOverview, error) {
	stripeAccountID, _, _ := w.orgs.GetStripeAccount(ctx, orgID)
	_ = userID // kept for audit / future per-operator fields
	wallet := &WalletOverview{StripeAccountID: stripeAccountID}

	// Fetch account capabilities from Stripe so wallet shows if charges/payouts are active
	if stripeAccountID != "" && w.stripe != nil {
		acct, err := w.stripe.GetAccount(ctx, stripeAccountID)
		if err == nil && acct != nil {
			wallet.ChargesEnabled = acct.ChargesEnabled
			wallet.PayoutsEnabled = acct.PayoutsEnabled
		}
	}

	records, err := w.records.ListByOrganization(ctx, orgID)
	if err != nil {
		return wallet, nil
	}

	for _, r := range records {
		rec := WalletRecord{
			ID:             r.ID.String(),
			ProposalID:     r.ProposalID.String(),
			ProposalAmount: r.ProposalAmount,
			PlatformFee:    r.PlatformFeeAmount,
			ProviderPayout: r.ProviderPayout,
			PaymentStatus:  string(r.Status),
			TransferStatus: string(r.TransferStatus),
			CreatedAt:      r.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}
		if r.MilestoneID != uuid.Nil {
			rec.MilestoneID = r.MilestoneID.String()
		}
		wallet.Records = append(wallet.Records, rec)

		switch {
		case r.TransferStatus == domain.TransferCompleted:
			wallet.TransferredAmount += r.ProviderPayout
		case r.Status == domain.RecordStatusSucceeded && r.TransferStatus == domain.TransferPending:
			wallet.EscrowAmount += r.ProviderPayout
		}
	}

	wallet.AvailableAmount = wallet.EscrowAmount

	// Commission side — populated only when a referral wallet reader
	// is wired (the referral feature might not be active in every
	// deployment). Failures are swallowed so a broken referral read
	// never takes down the provider-side wallet.
	if w.referralWallet != nil {
		w.populateCommissionSide(ctx, wallet, userID)
	}

	return wallet, nil
}

// populateCommissionSide fills in the apporteur view of the wallet.
// Errors are intentionally swallowed — the provider-side wallet is the
// primary view and must never be taken down by a flaky referral read.
func (w *WalletService) populateCommissionSide(ctx context.Context, wallet *WalletOverview, userID uuid.UUID) {
	if sum, err := w.referralWallet.GetReferrerSummary(ctx, userID); err == nil {
		wallet.Commissions = CommissionWallet{
			PendingCents:    sum.PendingCents,
			PendingKYCCents: sum.PendingKYCCents,
			PaidCents:       sum.PaidCents,
			ClawedBackCents: sum.ClawedBackCents,
			Currency:        sum.Currency,
		}
	}
	recent, err := w.referralWallet.RecentCommissions(ctx, userID, 50)
	if err != nil {
		return
	}
	wallet.CommissionRecords = make([]WalletCommissionRecord, 0, len(recent))
	for _, r := range recent {
		rec := WalletCommissionRecord{
			ID:               r.ID.String(),
			GrossAmountCents: r.GrossAmountCents,
			CommissionCents:  r.CommissionCents,
			Currency:         r.Currency,
			Status:           r.Status,
			StripeTransferID: r.StripeTransferID,
			CreatedAt:        r.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}
		if r.ReferralID != uuid.Nil {
			rec.ReferralID = r.ReferralID.String()
		}
		if r.ProposalID != uuid.Nil {
			rec.ProposalID = r.ProposalID.String()
		}
		if r.MilestoneID != uuid.Nil {
			rec.MilestoneID = r.MilestoneID.String()
		}
		if r.PaidAt != nil {
			rec.PaidAt = r.PaidAt.Format("2006-01-02T15:04:05Z")
		}
		if r.ClawedBackAt != nil {
			rec.ClawedBackAt = r.ClawedBackAt.Format("2006-01-02T15:04:05Z")
		}
		wallet.CommissionRecords = append(wallet.CommissionRecords, rec)
	}
}

// GetPaymentRecord returns the payment record for a proposal.
func (w *WalletService) GetPaymentRecord(ctx context.Context, proposalID uuid.UUID) (*domain.PaymentRecord, error) {
	return w.records.GetByProposalID(ctx, proposalID)
}

// PreviewFee returns the fee schedule for the authenticated user
// alongside a permission flag that tells the UI whether the caller
// would actually pay the fee on a hypothetical proposal against
// recipientID. Used by the web/mobile proposal creation flow to render
// the live simulator.
//
// Subscription-aware: when the caller has an active Premium
// subscription, the FeeCents is zeroed (and NetCents equals
// AmountCents). The tier grid is still returned so the UI can explain
// "you would pay X without Premium" if it wants — the caller decides
// how to present the waiver visually.
//
// Visibility rule (single source of truth = proposal.DetermineRoles):
//   - recipientID nil: fallback to role-based default. Enterprise is
//     ALWAYS client so ViewerIsProvider=false. Provider is ALWAYS
//     provider so ViewerIsProvider=true. Agency defaults to true
//     (proposal against an enterprise is the common case); callers that
//     need precise agency resolution MUST pass recipientID.
//   - recipientID set: run DetermineRoles(caller, recipient) and set
//     ViewerIsProvider from the computed provider_id. Invalid
//     combinations (agency+agency, enterprise+enterprise) set the flag
//     to false defensively — the UI must never show fees when the
//     backend cannot confirm the caller is the prestataire.
func (w *WalletService) PreviewFee(ctx context.Context, userID uuid.UUID, amountCents int64, recipientID *uuid.UUID) (*FeePreviewResult, error) {
	u, err := w.users.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("fetch user: %w", err)
	}
	billingRole := billing.RoleFromUser(string(u.Role))
	calc := billing.Calculate(billingRole, amountCents)

	viewerIsProvider := defaultViewerIsProvider(u.Role)
	if recipientID != nil {
		recipient, rErr := w.users.GetByID(ctx, *recipientID)
		if rErr != nil || recipient == nil {
			// Unknown recipient — fail closed rather than leak the fee
			// to a potentially client-side viewer. The UI hides the
			// preview.
			viewerIsProvider = false
		} else {
			_, providerID, drErr := proposaldomain.DetermineRoles(
				userID, string(u.Role),
				*recipientID, string(recipient.Role),
			)
			if drErr != nil {
				viewerIsProvider = false
			} else {
				viewerIsProvider = providerID == userID
			}
		}
	}

	// Waive the fee for Premium subscribers. The tier grid (calc.Tiers
	// and calc.ActiveTierIndex) is kept intact so the UI can still show
	// "Premium → 0 €, normal price would be X" if it chooses.
	viewerIsSubscribed := false
	if w.subscriptions != nil && viewerIsProvider {
		active, sErr := w.subscriptions.IsActive(ctx, userID)
		if sErr != nil {
			slog.Warn("payment: subscription lookup in PreviewFee failed",
				"user_id", userID, "error", sErr)
		} else if active {
			viewerIsSubscribed = true
			calc.FeeCents = 0
			calc.NetCents = calc.AmountCents
		}
	}

	return &FeePreviewResult{
		Billing:            calc,
		ViewerIsProvider:   viewerIsProvider,
		ViewerIsSubscribed: viewerIsSubscribed,
	}, nil
}

// computePlatformFee looks up the provider's role and returns the flat
// fee from the billing schedule, waived to zero when the provider is a
// Premium subscriber. Returns an error if the provider cannot be
// resolved — creating a payment record without knowing which grid
// applies would skew either the platform (under-charge) or the provider
// (over-charge), so we fail fast on user lookup failure.
//
// Subscription reader failures, by contrast, do NOT fail the payment:
// the Redis-backed cache can degrade, the database can blip, and we
// must not block a live checkout over a cache miss. When the reader
// errors we log + fall back to the full grid fee (the conservative
// choice: the platform keeps its revenue, the user sees the normal
// fee). A genuinely subscribed user affected by this edge case will be
// refunded the milestone fee via support.
//
// Lives on WalletService (not ChargeService) because the same helper is
// called from PreviewFee — co-locating it here means a single source of
// truth for "how much is this provider's fee right now?". ChargeService
// reuses it via the parent Service composition.
func (w *WalletService) computePlatformFee(ctx context.Context, providerID uuid.UUID, amountCents int64) (int64, error) {
	u, err := w.users.GetByID(ctx, providerID)
	if err != nil {
		return 0, fmt.Errorf("fetch provider: %w", err)
	}
	billingRole := billing.RoleFromUser(string(u.Role))
	fee := billing.Calculate(billingRole, amountCents).FeeCents

	if w.subscriptions != nil {
		active, subErr := w.subscriptions.IsActive(ctx, providerID)
		if subErr != nil {
			slog.Warn("payment: subscription lookup failed, applying full fee",
				"provider_id", providerID, "error", subErr)
			return fee, nil
		}
		if active {
			return 0, nil
		}
	}
	return fee, nil
}

// defaultViewerIsProvider is the role-only fallback when no recipient
// is known. Enterprise is ALWAYS the client; everyone else is (likely)
// the provider. Agency defaults to true for the happy path (agency
// pitching an enterprise); edge cases MUST be disambiguated by passing
// recipientID.
func defaultViewerIsProvider(role domainuser.Role) bool {
	return role != domainuser.RoleEnterprise
}
