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
		ProposalSummaries: referralapp.NewProposalRepoSummaryResolver(deps.Proposals, deps.Milestones),
	})
	// Setter-based wiring to avoid import cycles between
	// proposal/payment/embedded.
	deps.Proposal.SetReferralAttributor(referralSvc)
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
