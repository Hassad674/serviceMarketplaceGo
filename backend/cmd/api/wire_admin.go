package main

import (
	"context"
	"database/sql"

	"github.com/google/uuid"

	"marketplace-backend/internal/adapter/postgres"
	adminapp "marketplace-backend/internal/app/admin"
	mediaapp "marketplace-backend/internal/app/media"
	organizationapp "marketplace-backend/internal/app/organization"
	"marketplace-backend/internal/app/searchindex"
	"marketplace-backend/internal/handler"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
)

// adminActorSearchIndexer adapts the existing *searchindex.Publisher
// to the narrow adminapp.ActorSearchIndexer port. It deliberately
// reuses the publisher's org-scoped, all-persona primitives so admin
// moderation never re-implements actor document keying:
//
//   - RemoveActor  → PublishDelete: emits a search.delete outbox
//     event whose handler wipes EVERY persona variant of the org via
//     `filter_by organization_id:<id>` (the composite-ID scheme means
//     a single per-ID delete would miss the other personas).
//   - ReindexActor → PublishReindexAllPersonas: emits one
//     search.reindex event per persona; the indexer rebuilds the
//     SearchDocument from the same Indexer.BuildDocument path used
//     everywhere else, so the upsert is keyed identically and never
//     duplicates.
//
// Going through the outbox (not a direct Typesense call) keeps the
// admin action non-blocking and gives us the existing retry / drift
// reconciliation for free.
type adminActorSearchIndexer struct {
	pub *searchindex.Publisher
}

func (a adminActorSearchIndexer) RemoveActor(ctx context.Context, orgID uuid.UUID) error {
	return a.pub.PublishDelete(ctx, orgID)
}

func (a adminActorSearchIndexer) ReindexActor(ctx context.Context, orgID uuid.UUID) error {
	return a.pub.PublishReindexAllPersonas(ctx, orgID)
}

// adminDeps captures the upstream services + repositories that
// adminapp.NewService stitches together. The admin feature touches
// nearly every other aggregate (users, reports, reviews, jobs,
// proposals, media, moderation, organizations, team) so the deps
// list is naturally wide.
type adminDeps struct {
	DB                    *sql.DB
	Users                 repository.UserRepository
	Reports               repository.ReportRepository
	Reviews               repository.ReviewRepository
	Jobs                  repository.JobRepository
	Applications          repository.JobApplicationRepository
	Proposals             repository.ProposalRepository
	Media                 repository.MediaRepository
	ModerationResults     repository.ModerationResultsRepository
	Audit                 repository.AuditRepository
	Storage               service.StorageService
	Session               service.SessionService
	Broadcaster           service.MessageBroadcaster
	AdminNotifier         service.AdminNotifierService
	Organizations         repository.OrganizationRepository
	OrganizationMembers   repository.OrganizationMemberRepository
	OrganizationInvites   repository.OrganizationInvitationRepository
	Membership            *organizationapp.MembershipService
	Invitation            *organizationapp.InvitationService
	MediaSvcForListing    *mediaapp.Service // unused field, kept for forwards compatibility

	// SearchPublisher is the optional outbox publisher used to keep
	// the Typesense actor index consistent with a user's moderation
	// status (deindex on suspend/ban, reindex on unsuspend/unban).
	// Nil when Typesense is not configured — moderation then skips the
	// search sync entirely (the DB status flip is still authoritative).
	SearchPublisher *searchindex.Publisher
}

// wireAdmin brings up the admin app service + handler. The two
// admin-specific repositories (admin_conversations,
// admin_moderation) live here because they are only consumed by the
// admin aggregate.
func wireAdmin(deps adminDeps) *handler.AdminHandler {
	adminConvRepo := postgres.NewAdminConversationRepository(deps.DB)
	adminModerationRepo := postgres.NewAdminModerationRepository(deps.DB)
	adminSvc := adminapp.NewService(adminapp.ServiceDeps{
		Users:              deps.Users,
		Reports:            deps.Reports,
		Reviews:            deps.Reviews,
		Jobs:               deps.Jobs,
		Applications:       deps.Applications,
		Proposals:          deps.Proposals,
		AdminConversations: adminConvRepo,
		MediaRepo:          deps.Media,
		ModerationRepo:     adminModerationRepo,
		ModerationResults:  deps.ModerationResults,
		Audit:              deps.Audit,
		StorageSvc:         deps.Storage,
		SessionSvc:         deps.Session,
		Broadcaster:        deps.Broadcaster,
		AdminNotifier:      deps.AdminNotifier,
		// Phase 6 team admin wiring — these power the GET team
		// detail endpoint and the four force actions. The membership
		// + invitation services already carry notifSvc so team
		// events triggered by force actions still land in the
		// notifications table through the same pipeline as
		// user-driven actions.
		Orgs:           deps.Organizations,
		OrgMembers:     deps.OrganizationMembers,
		OrgInvitations: deps.OrganizationInvites,
		Membership:     deps.Membership,
		Invitation:     deps.Invitation,
	})

	// Moderation ↔ search consistency. Only wired when Typesense is
	// configured (publisher non-nil); otherwise the admin service
	// receives a nil indexer and the sync is a clean no-op. The
	// OrganizationRepository satisfies adminapp's narrow
	// FindByOwnerUserID resolver — the actor document is org-scoped,
	// so a moderated user maps to the org they own.
	if deps.SearchPublisher != nil {
		adminSvc = adminSvc.WithActorSearchIndexer(
			adminActorSearchIndexer{pub: deps.SearchPublisher},
			deps.Organizations,
		)
	}

	return handler.NewAdminHandler(adminSvc)
}
