package main

import (
	"context"
	"database/sql"
	"log/slog"

	"marketplace-backend/internal/adapter/postgres"
	"marketplace-backend/internal/app/messaging"
	notifapp "marketplace-backend/internal/app/notification"
	paymentapp "marketplace-backend/internal/app/payment"
	proposalapp "marketplace-backend/internal/app/proposal"
	referralapp "marketplace-backend/internal/app/referral"
	"marketplace-backend/internal/handler"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
)

// referralWiring carries the products of the apport d'affaires
// (referral) feature initialisation. The scheduler runs in a
// goroutine started inside wireReferral that stops when the supplied
// context cancels.
type referralWiring struct {
	Service *referralapp.Service
	Handler *handler.ReferralHandler
	Repo    *postgres.ReferralRepository // exposed for the apporteur reputation aggregate
}

// referralDeps captures the upstream dependencies needed by the
// referral feature. The feature plugs into proposal + payment via
// setters because constructor injection would create an import cycle
// (proposal/payment are built before referral).
type referralDeps struct {
	Ctx              context.Context
	DB               *sql.DB
	Users            repository.UserRepository
	Organizations    repository.OrganizationRepository
	OrganizationMems repository.OrganizationMemberRepository
	Proposals        repository.ProposalRepository
	Milestones       repository.MilestoneRepository
	Messaging        *messaging.Service
	Notifications    *notifapp.Service
	Stripe           service.StripeService
	StripeReversals  service.StripeTransferReversalService
	FreelanceProfile *postgres.FreelanceProfileRepository
	Proposal         *proposalapp.Service
	Payment          *paymentapp.Service
	// Conversations is the messaging/conversation persistence adapter
	// queried by the relationship checker — the referral service uses
	// it to refuse intros between two parties that already share a
	// 1:1 conversation (anti-fraud commission gate).
	Conversations *postgres.ConversationRepository
	// Audits is the append-only audit log repository. The referral
	// service writes a row here every time the anti-fraud gate
	// blocks an intro.
	Audits repository.AuditRepository
}

// wireReferral brings up the apport d'affaires feature. The whole
// feature is purely optional: startup with no referral service leaves
// every exposed port nil, and every call site short-circuits on that
// check. Setter wiring on proposal + payment is applied here so the
// caller does not need to remember it.
//
// Referral scheduler — hourly tick running ExpireStaleIntros (14 days
// of silence on pending_* rows) and ExpireMaturedReferrals (active
// rows past expires_at). Runs in its own goroutine; stops when ctx
// is cancelled along with the rest of the background workers.
func wireReferral(deps referralDeps) referralWiring {
	referralRepo := postgres.NewReferralRepository(deps.DB)
	referralSvc := referralapp.NewService(referralapp.ServiceDeps{
		Referrals:         referralRepo,
		Users:             deps.Users,
		Messages:          deps.Messaging,
		Notifications:     deps.Notifications,
		Stripe:            deps.Stripe,
		Reversals:         deps.StripeReversals,
		SnapshotProfiles:  referralapp.NewThinSnapshotLoader(deps.FreelanceProfile),
		StripeAccounts:    referralapp.NewOrgStripeAccountResolver(deps.Organizations),
		OrgMembers:        referralapp.NewOrgDirectoryMemberResolver(deps.Organizations, deps.OrganizationMems),
		// V7 NF-12: pass the referral repo as the attribution lister so
		// the resolver can independently filter requested proposal ids
		// against the referral_attributions table — defence-in-depth
		// against cross-tenant proposal lookups via WithSystemActor.
		ProposalSummaries: referralapp.NewProposalRepoSummaryResolver(deps.Proposals, deps.Milestones, referralRepo),
		// Anti-fraud gate: refuse an intro when the provider party and
		// the client party already share a 1:1 conversation. The
		// adapter is the conversation postgres repository — wired here
		// (and not via a setter) because the dependency cycle the
		// referral / proposal / payment trio fights does not extend
		// to messaging.
		Relationships: referralapp.NewConversationRelationshipChecker(deps.Conversations),
		Audits:        deps.Audits,
		// Run B (WALLET-UNIFY) — projection ports for the unified
		// wallet/summary endpoint. Both adapters are pass-throughs to
		// the underlying repositories.
		MilestonesByProposal: deps.Milestones,
		OrgMembersLister:     deps.OrganizationMems,
		// Apporteur detail page: human-readable provider + client
		// names. Org name wins when the user owns an agency/enterprise
		// org; falls back to the user's FullName otherwise. Wired with
		// the segregated readers so the resolver depends on the
		// smallest possible interfaces.
		PartyDisplayNames: referralapp.NewOrgFirstPartyDisplayNameResolver(
			deps.Users,
			deps.Organizations,
		),
	})
	// Setter-based wiring to avoid import cycles between
	// proposal/payment/embedded.
	deps.Proposal.SetReferralAttributor(referralSvc)
	// CRIT-REF fix: the apporteur commission row used to be created only
	// inside payments.TransferMilestone — which never fires for fresh
	// providers who haven't clicked manual payout yet. The preparer hook
	// creates the pending commission row on milestone APPROVAL so the
	// apporteur wallet is hydrated immediately, independent from the
	// provider auto-transfer eligibility gate.
	deps.Proposal.SetReferralCommissionPreparer(referralSvc)
	deps.Payment.SetReferralDistributor(referralSvc)
	deps.Payment.SetReferralClawback(referralSvc)
	deps.Payment.SetReferralWalletReader(referralSvc)

	scheduler := referralapp.NewScheduler(referralSvc, 0)
	go scheduler.Run(deps.Ctx)
	slog.Info("referral scheduler started")

	return referralWiring{
		Service: referralSvc,
		Handler: handler.NewReferralHandler(referralSvc),
		Repo:    referralRepo,
	}
}
