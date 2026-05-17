package payment

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/billing"
	milestonedomain "marketplace-backend/internal/domain/milestone"
	domain "marketplace-backend/internal/domain/payment"
	proposaldomain "marketplace-backend/internal/domain/proposal"
	domainuser "marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/port/repository"
	portservice "marketplace-backend/internal/port/service"
	"marketplace-backend/internal/system"
)

// MilestoneStatusReader is the narrow port the wallet uses to resolve
// milestone status in batch when splitting a payment record into the
// EscrowAmount vs AvailableAmount buckets. Defined locally (rather than
// importing the full MilestoneRepository) so the wallet only sees the
// method it actually needs — pure ISP.
//
// Contract:
//   - Returns a map keyed by milestone id, value = current status.
//   - Missing ids (milestone hard-deleted, corrupted FK) are omitted
//     from the result — the caller treats them as "unknown" and keeps
//     the record in escrow (conservative: never mark unverified funds
//     as retire-eligible).
//   - SYSTEM-ACTOR: this lookup is intrinsically cross-org (one org's
//     wallet view may aggregate milestones owned by different proposal
//     parties when the same record references multiple orgs through
//     transfers). The adapter MUST tag ctx with system.WithSystemActor
//     before the SQL hits the RLS-gated table.
type MilestoneStatusReader interface {
	StatusByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]milestonedomain.MilestoneStatus, error)
}

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
	// users is narrowed to UserReader — wallet only resolves the caller
	// + recipient user rows to compute fees and viewer-side flags.
	users   repository.UserReader
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

	// milestones resolves the per-milestone status used to split a
	// payment record between EscrowAmount (client paid, milestone not
	// yet approved) and AvailableAmount (milestone approved, transfer
	// not yet dispatched). Nil when the milestone feature is not wired
	// — wallet then degrades to the conservative "everything escrow,
	// nothing available" mode rather than silently flagging unverified
	// funds as retire-eligible.
	milestones MilestoneStatusReader
}

// WalletServiceDeps groups every dependency NewWalletService needs.
//
// Organizations is narrowed to OrganizationStripeStore — the wallet
// sub-service only reads stripe_account_id off the org row.
type WalletServiceDeps struct {
	Records       repository.PaymentRecordRepository
	Users         repository.UserReader
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

// SetMilestoneStatusReader plugs the milestone status reader used by
// GetWalletOverview to split a payment record between EscrowAmount
// (client paid, milestone not yet approved) and AvailableAmount
// (milestone approved, transfer pending). Passing nil keeps the
// wallet in conservative mode: every paid record sits in escrow and
// AvailableAmount stays zero — the apporteur cannot accidentally see
// unverified funds as retire-eligible.
func (w *WalletService) SetMilestoneStatusReader(r MilestoneStatusReader) {
	w.milestones = r
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
		wallet.Records = append(wallet.Records, recordToDTO(r))
	}

	// Resolve milestone statuses in a single batch to split escrow vs
	// available. Failures (or a nil reader) drop us into the conservative
	// branch: every paid+pending record sits in escrow, AvailableAmount
	// stays zero. Better to under-report retire-eligibility than to
	// flag unverified funds as drainable.
	statuses := w.fetchMilestoneStatuses(ctx, records)
	w.aggregateBuckets(wallet, records, statuses)

	// Commission side — populated only when a referral wallet reader
	// is wired (the referral feature might not be active in every
	// deployment). Failures are swallowed so a broken referral read
	// never takes down the provider-side wallet.
	if w.referralWallet != nil {
		w.populateCommissionSide(ctx, wallet, userID)
	}

	return wallet, nil
}

// recordToDTO is the pure mapping from the domain payment record to
// the wire-facing WalletRecord. Extracted so GetWalletOverview's
// outer loop stays linear and the dispatch logic below can be unit-
// tested independently.
func recordToDTO(r *domain.PaymentRecord) WalletRecord {
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
	return rec
}

// fetchMilestoneStatuses batches every milestone status lookup into a
// SINGLE backend call (no N+1). Returns an empty map and logs a
// warning when the reader is missing or the call errors — the caller
// treats every record as "unknown milestone status" which the
// dispatcher maps to EscrowAmount (conservative).
func (w *WalletService) fetchMilestoneStatuses(ctx context.Context, records []*domain.PaymentRecord) map[uuid.UUID]milestonedomain.MilestoneStatus {
	if w.milestones == nil || len(records) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, 0, len(records))
	seen := make(map[uuid.UUID]struct{}, len(records))
	for _, r := range records {
		if r == nil || r.MilestoneID == uuid.Nil {
			continue
		}
		if _, dup := seen[r.MilestoneID]; dup {
			continue
		}
		seen[r.MilestoneID] = struct{}{}
		ids = append(ids, r.MilestoneID)
	}
	if len(ids) == 0 {
		return nil
	}
	statuses, err := w.milestones.StatusByIDs(ctx, ids)
	if err != nil {
		slog.Warn("payment: milestone status batch lookup failed, degrading to escrow-only",
			"error", err, "milestone_count", len(ids))
		return nil
	}
	return statuses
}

// aggregateBuckets dispatches every record into its money bucket
// based on (transfer_status, payment_status, milestone.status).
// The full decision matrix is documented below in classifyRecordBucket.
//
// Conservative default: missing milestone status → EscrowAmount. We
// never put money in AvailableAmount unless we can PROVE the milestone
// is approved by the client. The withdraw endpoint reads
// AvailableAmount as the cap on what can be drained — getting this
// wrong would let a provider drain escrowed (un-approved) funds.
func (w *WalletService) aggregateBuckets(
	wallet *WalletOverview,
	records []*domain.PaymentRecord,
	statuses map[uuid.UUID]milestonedomain.MilestoneStatus,
) {
	for _, r := range records {
		if r == nil {
			continue
		}
		bucket := classifyRecordBucket(r, statuses)
		switch bucket {
		case bucketTransferred:
			wallet.TransferredAmount += r.ProviderPayout
		case bucketAvailable:
			wallet.AvailableAmount += r.ProviderPayout
		case bucketEscrow:
			wallet.EscrowAmount += r.ProviderPayout
		}
	}
}

// recordBucket enumerates the three money destinations on the wallet
// overview. Used internally by classifyRecordBucket so callers don't
// reach for stringly-typed comparisons.
type recordBucket int

const (
	bucketSkip recordBucket = iota
	bucketTransferred
	bucketEscrow
	bucketAvailable
)

// classifyRecordBucket is the pure dispatch decision: for a single
// payment record + the milestone-status lookup table, return the
// bucket the record's ProviderPayout should land in.
//
// Decision matrix (internal design note):
//
//	transfer_status=completed                                       → Transferred
//	status=succeeded ∧ transfer=pending ∧ milestone=approved/released → Available (signed off, transfer deferred)
//	status=succeeded ∧ transfer=pending ∧ milestone=funded/submitted/disputed → Escrow
//	status=succeeded ∧ transfer=pending ∧ milestone missing/unknown → Escrow (conservative)
//	any other (failed / refunded / pending payment)                 → Skip
//
// Extracted as a pure func so the matrix is straightforwardly
// table-testable without standing up the wallet service.
func classifyRecordBucket(
	r *domain.PaymentRecord,
	statuses map[uuid.UUID]milestonedomain.MilestoneStatus,
) recordBucket {
	if r.TransferStatus == domain.TransferCompleted {
		return bucketTransferred
	}
	if r.Status != domain.RecordStatusSucceeded || r.TransferStatus != domain.TransferPending {
		return bucketSkip
	}
	// Paid+pending — needs milestone context to decide escrow vs available.
	if r.MilestoneID == uuid.Nil {
		return bucketEscrow
	}
	status, ok := statuses[r.MilestoneID]
	if !ok {
		return bucketEscrow // conservative on missing status
	}
	switch status {
	case milestonedomain.StatusApproved, milestonedomain.StatusReleased:
		// Client signed off. CompleteProposal does m.Approve() THEN
		// m.Release(), so the normal end-of-mission state is Released,
		// not Approved. The ONLY authoritative "money actually left"
		// signal is transfer_status=completed — already handled at the
		// top of this function. A Released milestone whose transfer is
		// still pending means the auto-transfer was deliberately
		// deferred because the provider's KYC or billing profile is
		// incomplete (Volet 3: providerEligibleForAutoTransfer == false).
		// The funds are NOT gone — they are drainable manually via
		// "Retirer", so they belong in Available, never Transferred.
		// Pre-Volet-3 this branch returned Transferred on the
		// "released+pending should not happen" assumption; Volet 3 made
		// it a normal state, so that assumption became wrong and was the
		// root cause of money showing as Transféré instead of Disponible
		// when KYC/billing is incomplete.
		return bucketAvailable
	case milestonedomain.StatusFunded,
		milestonedomain.StatusSubmitted,
		milestonedomain.StatusDisputed:
		return bucketEscrow
	}
	// pending_funding / cancelled / refunded with status=succeeded is
	// data corruption — skip to keep the totals honest.
	return bucketSkip
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
			Paid30dCents:    sum.Paid30dCents,
			LifetimeCents:   sum.LifetimeCents,
			Currency:        sum.Currency,
		}
	}
	recent, err := w.referralWallet.RecentCommissions(ctx, userID, 50)
	if err != nil {
		return
	}
	wallet.CommissionRecords = make([]WalletCommissionRecord, 0, len(recent))
	for _, r := range recent {
		// retire_eligible mirrors the commission retry orchestrator's
		// switch statement: pending_kyc and failed rows can be retried,
		// every other status is terminal or owned by another flow.
		// Keep this in sync with referralapp.Service.RetryCommission.
		retireEligible := r.Status == "pending_kyc" || r.Status == "failed"
		rec := WalletCommissionRecord{
			ID:               r.ID.String(),
			GrossAmountCents: r.GrossAmountCents,
			CommissionCents:  r.CommissionCents,
			Currency:         r.Currency,
			Status:           r.Status,
			StripeTransferID: r.StripeTransferID,
			CreatedAt:        r.CreatedAt.Format("2006-01-02T15:04:05Z"),
			RetireEligible:   retireEligible,
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
//
// SYSTEM-ACTOR: this helper is only used by internal cross-feature
// flows (no HTTP handler calls it directly) AFTER the proposal
// ownership has been confirmed upstream. Tag the read so the
// BYPASSRLS pool serves the lookup.
func (w *WalletService) GetPaymentRecord(ctx context.Context, proposalID uuid.UUID) (*domain.PaymentRecord, error) {
	return w.records.GetByProposalID(system.WithSystemActor(ctx), proposalID)
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
