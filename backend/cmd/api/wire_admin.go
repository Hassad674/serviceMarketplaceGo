package main

import (
	"database/sql"

	"marketplace-backend/internal/adapter/postgres"
	adminapp "marketplace-backend/internal/app/admin"
	mediaapp "marketplace-backend/internal/app/media"
	organizationapp "marketplace-backend/internal/app/organization"
	"marketplace-backend/internal/handler"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
)

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
	return handler.NewAdminHandler(adminSvc)
}
