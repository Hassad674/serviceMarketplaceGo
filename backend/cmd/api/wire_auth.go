package main

import (
	"time"

	redisadapter "marketplace-backend/internal/adapter/redis"
	"marketplace-backend/internal/app/auth"
	organizationapp "marketplace-backend/internal/app/organization"
	"marketplace-backend/internal/config"
	"marketplace-backend/internal/handler"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"

	goredis "github.com/redis/go-redis/v9"
)

// authWiring carries the products of the auth feature initialisation:
// the organization aggregate (consumed by the auth flows when a fresh
// signup is paired with a starter org), the auth service itself, the
// two Redis-backed brute-force guards (login + password reset), and
// the HTTP handler ready to bind onto the router.
type authWiring struct {
	OrganizationSvc       *organizationapp.Service
	AuthSvc               *auth.Service
	LoginBruteForce       service.BruteForceService
	PasswordResetThrottle service.BruteForceService
	AuthHandler           *handler.AuthHandler
}

// authDeps captures the upstream dependencies the auth feature needs:
// the user / reset / organization / audit repositories, the always-on
// output adapters (hasher, JWT issuer, email, session, refresh-token
// blacklist), the cookie configuration shared by the auth handler, the
// Redis pool used by the brute-force services, and the *config.Config
// so the SEC-07 policies stay hooked into the same env vars as the
// rest of the security middleware.
type authDeps struct {
	Cfg                        *config.Config
	Redis                      *goredis.Client
	UserRepo                   repository.UserRepository
	ResetRepo                  repository.PasswordResetRepository
	OrganizationRepo           repository.OrganizationRepository
	OrganizationMemberRepo     repository.OrganizationMemberRepository
	OrganizationInvitationRepo repository.OrganizationInvitationRepository
	AuditRepo                  repository.AuditRepository
	Hasher                     service.HasherService
	TokenSvc                   service.TokenService
	EmailSvc                   service.EmailService
	SessionSvc                 service.SessionService
	RefreshBlacklistSvc        service.RefreshBlacklistService
	CookieCfg                  *handler.CookieConfig
}

// wireAuth brings up the auth feature: the organization aggregate
// (the auth flows pair every fresh signup with a starter org), the
// auth service with its full ServiceDeps wiring, the SEC-07 Redis
// brute-force guards (login + password reset), and the HTTP handler
// ready to bind onto the router. The returned struct is consumed by
// main.go which still owns the cross-feature setter calls that wire
// the moderation orchestrator into the auth service after both have
// been built.
func wireAuth(deps authDeps) authWiring {
	organizationSvc := organizationapp.NewService(deps.OrganizationRepo, deps.OrganizationMemberRepo, deps.OrganizationInvitationRepo)
	// invitationSvc and membershipSvc are constructed below, AFTER the
	// notification feature is set up — they depend on notifSvc so the
	// team events (invitation accepted, role changed, transfer, …) can
	// fire notifications through the same pipeline as the rest of the app.
	authSvc := auth.NewServiceWithDeps(auth.ServiceDeps{
		Users:            deps.UserRepo,
		Resets:           deps.ResetRepo,
		Hasher:           deps.Hasher,
		Tokens:           deps.TokenSvc,
		Email:            deps.EmailSvc,
		Orgs:             organizationSvc,
		Sessions:         deps.SessionSvc,         // SEC-16 — purge sessions on password reset
		RefreshBlacklist: deps.RefreshBlacklistSvc, // SEC-06 — refresh token rotation + replay detection
		Audits:           deps.AuditRepo,          // SEC-13 — emit auth audit events
		FrontendURL:      deps.Cfg.FrontendURL,
	})

	// SEC-07: brute-force protection. Two policies:
	//   - login: 5 per 15-min window per email, 30-min lockout (default)
	//   - password reset: 3 per hour per email/token, 30-min lockout
	loginBruteForce := redisadapter.NewBruteForceService(deps.Redis)
	passwordResetThrottle := redisadapter.NewBruteForceServiceWithPolicy(
		deps.Redis, 3, time.Hour, 30*time.Minute,
	)

	// F.5 S7: brute-force IsLocked must fail-CLOSED in production so
	// a Redis outage cannot bypass the per-email lockout.
	authHandler := handler.NewAuthHandler(authSvc, organizationSvc, deps.SessionSvc, deps.CookieCfg).
		WithBruteForce(loginBruteForce, passwordResetThrottle).
		WithFailClosed(deps.Cfg.IsProduction())

	return authWiring{
		OrganizationSvc:       organizationSvc,
		AuthSvc:               authSvc,
		LoginBruteForce:       loginBruteForce,
		PasswordResetThrottle: passwordResetThrottle,
		AuthHandler:           authHandler,
	}
}
