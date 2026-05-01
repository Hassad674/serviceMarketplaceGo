package main

import (
	"database/sql"
	"log/slog"
	"os"

	"marketplace-backend/internal/adapter/postgres"
	profileapp "marketplace-backend/internal/app/profile"
	profiledomain "marketplace-backend/internal/domain/profile"
	"marketplace-backend/internal/handler"
)

// socialLinkWiring carries the three persona-scoped social-link
// handlers. Each persona has its own service instance (bound at
// construction time so the downstream handler stays unaware of the
// persona dimension) and its own handler.
type socialLinkWiring struct {
	Agency    *handler.SocialLinkHandler
	Freelance *handler.SocialLinkHandler
	Referrer  *handler.SocialLinkHandler
}

// wireSocialLinks brings up the three persona-scoped social-link
// services + handlers. Any construction failure fails the process
// loud with os.Exit(1) — the social link aggregate has no graceful
// degradation path because the persona is invariant per request.
func wireSocialLinks(db *sql.DB) socialLinkWiring {
	socialLinkRepo := postgres.NewSocialLinkRepository(db)
	agencySvc, err := profileapp.NewSocialLinkService(socialLinkRepo, profiledomain.PersonaAgency)
	if err != nil {
		slog.Error("failed to init agency social link service", "error", err)
		os.Exit(1)
	}
	freelanceSvc, err := profileapp.NewSocialLinkService(socialLinkRepo, profiledomain.PersonaFreelance)
	if err != nil {
		slog.Error("failed to init freelance social link service", "error", err)
		os.Exit(1)
	}
	referrerSvc, err := profileapp.NewSocialLinkService(socialLinkRepo, profiledomain.PersonaReferrer)
	if err != nil {
		slog.Error("failed to init referrer social link service", "error", err)
		os.Exit(1)
	}
	return socialLinkWiring{
		Agency:    handler.NewSocialLinkHandler(agencySvc),
		Freelance: handler.NewSocialLinkHandler(freelanceSvc),
		Referrer:  handler.NewSocialLinkHandler(referrerSvc),
	}
}
