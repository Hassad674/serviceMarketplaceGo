package main

import (
	"database/sql"
	"log/slog"

	redisadapter "marketplace-backend/internal/adapter/redis"
	notifapp "marketplace-backend/internal/app/notification"
	organizationapp "marketplace-backend/internal/app/organization"
	"marketplace-backend/internal/config"
	"marketplace-backend/internal/handler"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"

	goredis "github.com/redis/go-redis/v9"
)

// teamWiring carries the products of the organization team feature
// initialisation: invitation + membership services (used by both
// admin handlers and the user-facing team handler), the role
// overrides service, and the matching HTTP handlers.
type teamWiring struct {
	InvitationSvc        *organizationapp.InvitationService
	MembershipSvc        *organizationapp.MembershipService
	RoleOverridesSvc     *organizationapp.RoleOverridesService
	InvitationHandler    *handler.InvitationHandler
	TeamHandler          *handler.TeamHandler
	RoleOverridesHandler *handler.RoleOverridesHandler
}

// teamDeps captures the upstream dependencies the team feature needs:
// the organization aggregate (orgs, members, invitations), the user
// repo (for member lookups and the team handler's user batch), the
// hasher + email service (invitation flow), the audit repo (role
// changes are append-only audited), the notification service (team
// events fan out as notifications), and the Redis client used by the
// rate-limiter adapters.
// teamDeps.UserBatch must implement repository.UserBatchReader; the
// production *postgres.UserRepository satisfies both UserRepository
// and UserBatchReader, so the same instance can be passed twice.
type teamDeps struct {
	Cfg                   *config.Config
	DB                    *sql.DB
	Redis                 *goredis.Client
	Orgs                  repository.OrganizationRepository
	Members               repository.OrganizationMemberRepository
	Invitations           repository.OrganizationInvitationRepository
	Users                 repository.UserRepository
	UserBatch             repository.UserBatchReader
	Hasher                service.HasherService
	Email                 service.EmailService
	Audits                repository.AuditRepository
	Notifications         *notifapp.Service
	OrganizationSvc       *organizationapp.Service
	SessionService        service.SessionService
	Cookie                *handler.CookieConfig
	InvitationRateLimiter organizationapp.InvitationRateLimiter
	TokenService          service.TokenService
}

// wireTeam brings up the organization team feature. The invitation
// + membership services need the notification service to fire the
// team_* events (invitation accepted, role changed, transfer, …)
// through the same pipeline as user-driven actions, so this helper
// must run AFTER the notification feature has been wired.
func wireTeam(deps teamDeps) teamWiring {
	invitationSvc := organizationapp.NewInvitationService(organizationapp.InvitationServiceDeps{
		Orgs:          deps.Orgs,
		Members:       deps.Members,
		Invitations:   deps.Invitations,
		Users:         deps.Users,
		Hasher:        deps.Hasher,
		Email:         deps.Email,
		RateLimiter:   deps.InvitationRateLimiter,
		Notifications: deps.Notifications,
		FrontendURL:   deps.Cfg.FrontendURL,
	})
	membershipSvc := organizationapp.NewMembershipService(organizationapp.MembershipServiceDeps{
		Orgs:          deps.Orgs,
		Members:       deps.Members,
		Users:         deps.Users,
		Notifications: deps.Notifications,
	})

	// Role permissions editor (R17 — per-org customization). Uses a
	// dedicated Redis-backed rate limiter so the audit tail and the
	// Owner email notification stay independent from the rest of the
	// invitation rate limit.
	rolePermsRateLimiter := redisadapter.NewRolePermissionsRateLimiter(deps.Redis)
	roleOverridesSvc := organizationapp.NewRoleOverridesService(organizationapp.RoleOverridesServiceDeps{
		Orgs:        deps.Orgs,
		Members:     deps.Members,
		Users:       deps.Users,
		Audits:      deps.Audits,
		Email:       deps.Email,
		RateLimiter: rolePermsRateLimiter,
	})

	invitationHandler := handler.NewInvitationHandler(handler.InvitationHandlerDeps{
		InvitationService: invitationSvc,
		OrgService:        deps.OrganizationSvc,
		TokenService:      deps.TokenService,
		SessionService:    deps.SessionService,
		Cookie:            deps.Cookie,
	})
	teamHandler := handler.NewTeamHandler(handler.TeamHandlerDeps{
		Membership:     membershipSvc,
		OrgService:     deps.OrganizationSvc,
		UserBatch:      deps.UserBatch,
		SessionService: deps.SessionService,
		Cookie:         deps.Cookie,
		Users:          deps.Users,
	})
	roleOverridesHandler := handler.NewRoleOverridesHandler(roleOverridesSvc)

	slog.Info("team feature wired (invitation + membership + role-overrides)")
	return teamWiring{
		InvitationSvc:        invitationSvc,
		MembershipSvc:        membershipSvc,
		RoleOverridesSvc:     roleOverridesSvc,
		InvitationHandler:    invitationHandler,
		TeamHandler:          teamHandler,
		RoleOverridesHandler: roleOverridesHandler,
	}
}

