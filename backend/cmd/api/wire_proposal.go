package main

import (
	"database/sql"

	"marketplace-backend/internal/adapter/postgres"
	"marketplace-backend/internal/adapter/worker"
	"marketplace-backend/internal/app/messaging"
	milestoneapp "marketplace-backend/internal/app/milestone"
	notifapp "marketplace-backend/internal/app/notification"
	paymentapp "marketplace-backend/internal/app/payment"
	proposalapp "marketplace-backend/internal/app/proposal"
	"marketplace-backend/internal/app/searchindex"
	"marketplace-backend/internal/config"
	"marketplace-backend/internal/handler"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
)

// proposalReposWiring carries the early-stage repositories the
// proposal feature owns. They are built before the notification /
// messaging / payment chain so the rest of main.go can plug them
// into any downstream wire helper that reads proposal data
// (project history, dispute, referral, admin, …) — and so the
// SearchPublisher + TxRunner that other features need is exposed in
// a single place.
type proposalReposWiring struct {
	ProposalRepo             *postgres.ProposalRepository
	MilestoneRepo            *postgres.MilestoneRepository
	MilestoneSvc             *milestoneapp.Service
	PaymentRecordRepo        *postgres.PaymentRecordRepository
	BonusLogRepo             *postgres.CreditBonusLogRepository
	PendingEventsRepo        *postgres.PendingEventRepository
	MilestoneTransitionsRepo *postgres.MilestoneTransitionRepository
	SearchPublisher          *searchindex.Publisher
	TxRunner                 repository.TxRunner
}

// proposalReposDeps captures the upstream resources the early-stage
// proposal repositories need: the SQL pool (every repo holds a
// reference) and the *config.Config (used by wireSearchPublisher to
// short-circuit when Typesense is not configured).
type proposalReposDeps struct {
	Cfg *config.Config
	DB  *sql.DB
}

// wireProposalRepos brings up the early-stage proposal repositories
// (proposal, milestone, payment record, bonus log, pending events,
// milestone transitions) plus the search publisher and the outbox
// transaction runner. These products are consumed by features that
// run BEFORE the notification / messaging / payment chain (project
// history, dispute, referral, admin) so they have to live in their
// own helper.
//
// Comments preserved verbatim from the legacy main.go layout: every
// BUG-NEW-04 and phase explanation block stays untouched so the
// audit trail remains intact.
func wireProposalRepos(deps proposalReposDeps) proposalReposWiring {
	db := deps.DB

	// Proposal
	// BUG-NEW-04 path 4/8: proposals is RLS-protected by migration 125
	// (USING client_organization_id = current_org OR provider_organization_id
	// = current_org). The txRunner wrap makes Create / Update /
	// GetByIDForOrg / List* pass under prod NOSUPERUSER NOBYPASSRLS.
	// Legacy GetByID stays for system-actor scheduler paths that run
	// with a privileged DB connection.
	proposalRepo := postgres.NewProposalRepository(db).WithTxRunner(postgres.NewTxRunner(db))

	// Milestone — per-step funding/delivery sub-aggregate of a proposal.
	// The proposal app service consumes milestoneSvc to delegate the
	// Fund/Submit/Approve/Release transitions, and the dispute service
	// (phase 8) delegates OpenDispute/RestoreFromDispute to it as well.
	// BUG-NEW-04 path 5/8: proposal_milestones is RLS-protected by
	// migration 125 — milestones inherit security from the parent
	// proposal via a JOIN on the policy. The txRunner wrap makes
	// CreateBatch / Update / GetByIDForOrg / ListByProposalForOrg pass
	// under prod NOSUPERUSER NOBYPASSRLS. Each operation resolves the
	// parent proposal's stakeholder org via a defensive lookup before
	// opening the tenant tx.
	milestoneRepo := postgres.NewMilestoneRepository(db).WithTxRunner(postgres.NewTxRunner(db))
	milestoneSvc := milestoneapp.NewService(milestoneapp.ServiceDeps{
		Milestones: milestoneRepo,
	})

	// Payment records (custom KYC repos removed — see migration 040/041)
	// BUG-NEW-04 path 7/8: payment_records is RLS-protected by migration
	// 125 (USING organization_id = current_setting('app.current_org_id',
	// true)). The txRunner wrap makes Create / Update / GetByIDForOrg /
	// ListByOrganization pass under prod NOSUPERUSER NOBYPASSRLS. The
	// client's org (resolved from organization_members at INSERT time)
	// is the access boundary; provider-side reads of money received go
	// through the tenant-isolated proposal path instead.
	paymentRecordRepo := postgres.NewPaymentRecordRepository(db).WithTxRunner(postgres.NewTxRunner(db))

	// Credit bonus fraud log
	bonusLogRepo := postgres.NewCreditBonusLogRepository(db)

	// Pending events queue (phase 6 — unified scheduler + Stripe outbox).
	// The proposal service writes events here when a milestone is
	// submitted (auto-approve), released (fund-reminder + auto-close),
	// or released into the Stripe outbox (phase 7).
	pendingEventsRepo := postgres.NewPendingEventRepository(db)

	// Search engine publisher — built once so every service that
	// mutates actor signals (freelance profile, referrer profile,
	// pricing, skills, etc.) can emit a `search.reindex` event on
	// the outbox without re-wiring the whole chain. See wire_search.go.
	searchPublisher := wireSearchPublisher(deps.Cfg, pendingEventsRepo)

	// Outbox transaction runner (BUG-05). Used by the freelance and
	// legacy profile services to commit a profile mutation and the
	// matching `search.reindex` pending event in a single atomic
	// transaction — preventing permanent Postgres / Typesense drift
	// when the publisher Schedule path would otherwise fail after
	// the profile UPDATE has already committed. Cheap to construct:
	// holds only a *sql.DB pointer.
	txRunner := postgres.NewTxRunner(db)

	// Milestone audit trail (phase 9 — append-only). Every successful
	// withMilestoneLock writes one row recording from→to status pair,
	// actor id + org, and an optional reason string. The DB user
	// holds INSERT/SELECT only on this table (Update/Delete are
	// forbidden so the timeline cannot be rewritten).
	milestoneTransitionsRepo := postgres.NewMilestoneTransitionRepository(db)

	return proposalReposWiring{
		ProposalRepo:             proposalRepo,
		MilestoneRepo:            milestoneRepo,
		MilestoneSvc:             milestoneSvc,
		PaymentRecordRepo:        paymentRecordRepo,
		BonusLogRepo:             bonusLogRepo,
		PendingEventsRepo:        pendingEventsRepo,
		MilestoneTransitionsRepo: milestoneTransitionsRepo,
		SearchPublisher:          searchPublisher,
		TxRunner:                 txRunner,
	}
}

// proposalServiceWiring carries the products of the late-stage
// proposal feature initialisation: the app service, the unified
// pending-events worker (kept exposed so wireSearchIndexer can
// register its own search.reindex / search.delete handlers on the
// same instance), and the HTTP handler ready to bind onto the router.
type proposalServiceWiring struct {
	ProposalSvc         *proposalapp.Service
	PendingEventsWorker *worker.Worker
	ProposalHandler     *handler.ProposalHandler
}

// proposalServiceDeps captures the upstream services + repos the
// proposal app service depends on. Bundled into a struct because the
// service surface touches a wide set of collaborators (notifications,
// messaging, payments, credits, storage, …) that a positional
// argument list would be impossible to read.
type proposalServiceDeps struct {
	Cfg                      *config.Config
	ProposalRepo             repository.ProposalRepository
	MilestoneRepo            repository.MilestoneRepository
	MilestoneTransitionsRepo repository.MilestoneTransitionRepository
	PendingEventsRepo        *postgres.PendingEventRepository
	BonusLogRepo             repository.CreditBonusLogRepository
	UserRepo                 repository.UserRepository
	UserBatch                repository.UserBatchReader
	OrganizationRepo         repository.OrganizationRepository
	JobCreditRepo            repository.JobCreditRepository
	StorageSvc               service.StorageService
	MessagingSvc             *messaging.Service
	NotifSvc                 *notifapp.Service
	PaymentInfoSvc           *paymentapp.Service
}

// wireProposalService brings up the proposal app service, the unified
// pending-events worker, and the HTTP handler. Runs AFTER the
// notification / messaging / payment chain because the service deps
// reach into all three.
//
// Cross-feature setter calls (paymentInfoSvc.SetProposalStatusReader,
// proposalSvc.SetModerationOrchestrator) STAY in main.go — they
// cross multiple wire boundaries so colocating them in this helper
// would defeat the purpose of the split.
func wireProposalService(deps proposalServiceDeps) proposalServiceWiring {
	// Wire services that depend on notifications
	proposalSvc := proposalapp.NewService(proposalapp.ServiceDeps{
		Proposals:            deps.ProposalRepo,
		Milestones:           deps.MilestoneRepo,
		MilestoneTransitions: deps.MilestoneTransitionsRepo,
		PendingEvents:        deps.PendingEventsRepo,
		Users:                deps.UserRepo,
		// Same concrete *postgres.UserRepository — it satisfies both
		// the wide UserRepository contract and the segregated
		// UserBatchReader (GetByIDs). The duplicate field exists so
		// the service can declare the segregated dep without forcing
		// every other caller to bring it in.
		UsersBatch:           deps.UserBatch,
		Organizations:        deps.OrganizationRepo,
		Messages:             deps.MessagingSvc,
		Storage:              deps.StorageSvc,
		Notifications:        deps.NotifSvc,
		Payments:             paymentProcessor(deps.PaymentInfoSvc, deps.Cfg),
		Credits:              deps.JobCreditRepo,
		BonusLog:             deps.BonusLogRepo,
		// Phase 6 timer defaults (override via env in production):
		// 7-day auto-approval, 7-day fund reminder, 14-day auto-close.
	})

	// Phase 6: pending_events worker — see wire_pending_events.go.
	// The worker handles milestone auto-approve, fund reminders, and
	// proposal auto-close; search reindex/delete handlers are added
	// later by wireSearchIndexer when Typesense is configured.
	pendingEventsWorker := newPendingEventsWorker(deps.PendingEventsRepo, proposalSvc)

	proposalHandler := handler.NewProposalHandler(proposalSvc, deps.PaymentInfoSvc)

	return proposalServiceWiring{
		ProposalSvc:         proposalSvc,
		PendingEventsWorker: pendingEventsWorker,
		ProposalHandler:     proposalHandler,
	}
}
