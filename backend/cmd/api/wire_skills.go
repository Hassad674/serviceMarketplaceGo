package main

import (
	"database/sql"

	"marketplace-backend/internal/adapter/postgres"
	profileapp "marketplace-backend/internal/app/profile"
	profilepricingapp "marketplace-backend/internal/app/profilepricing"
	"marketplace-backend/internal/app/searchindex"
	skillapp "marketplace-backend/internal/app/skill"
	"marketplace-backend/internal/handler"
	"marketplace-backend/internal/port/repository"
)

// skillsAndPricingWiring carries the products of the
// expertise/skills/profile-pricing constellation. All three live in
// their own app packages but share an org-type-resolver bridge to
// keep the upstream packages independent of domain/organization.
//
// SkillSvc is exposed (not just the handler) because the freelance
// profile handler decorates its responses through SkillSvc — it
// satisfies the local SkillsReader contract.
type skillsAndPricingWiring struct {
	ExpertiseSvc          *profileapp.ExpertiseService
	SkillSvc              *skillapp.Service
	SkillHandler          *handler.SkillHandler
	ProfilePricingSvc     *profilepricingapp.Service
	ProfilePricingHandler *handler.ProfilePricingHandler
}

// wireSkillsAndPricing brings up the expertise feature, the hybrid
// skill catalog, and the legacy agency profile pricing. Each block
// is wired as repo → service → handler so dropping the feature is a
// single-block deletion. The skill + profile pricing handlers
// optionally publish reindex events when a search publisher is wired.
func wireSkillsAndPricing(
	db *sql.DB,
	organizationRepo repository.OrganizationRepository,
	userRepo repository.UserRepository,
	searchPublisher *searchindex.Publisher,
) skillsAndPricingWiring {
	// Expertise feature (org-scoped domain specializations). Shares
	// the profile application package and is co-located in the
	// profile handler because expertise is part of the org's public
	// profile.
	expertiseRepo := postgres.NewExpertiseRepository(db)
	expertiseSvc := profileapp.NewExpertiseService(expertiseRepo, organizationRepo)

	// Skills feature (hybrid catalog + per-org profile attachments).
	// Uses a small org-type-resolver adapter (org_type_resolver.go)
	// to bridge the existing organization repo to the skill
	// service's dependency contract, keeping the skill package
	// independent of domain/organization.
	skillCatalogRepo := postgres.NewSkillCatalogRepository(db)
	profileSkillRepo := postgres.NewProfileSkillRepository(db)
	skillSvc := skillapp.NewService(
		skillCatalogRepo,
		profileSkillRepo,
		newOrgTypeResolverAdapter(organizationRepo),
	)
	skillHandler := handler.NewSkillHandler(skillSvc)
	if searchPublisher != nil {
		skillHandler = skillHandler.WithSearchIndexPublisher(searchPublisher)
	}

	// Profile pricing feature (migration 083). Uses a local
	// org-info resolver adapter (profile_pricing_org_info_resolver.go)
	// to bridge the existing organization + user repos to the
	// pricing service's dependency contract, keeping the
	// profilepricing package independent of domain/organization
	// and domain/user.
	profilePricingRepo := postgres.NewProfilePricingRepository(db)
	profilePricingSvc := profilepricingapp.NewService(
		profilePricingRepo,
		newProfilePricingOrgInfoResolverAdapter(organizationRepo, userRepo),
	)
	profilePricingHandler := handler.NewProfilePricingHandler(profilePricingSvc)
	if searchPublisher != nil {
		profilePricingHandler = profilePricingHandler.WithSearchIndexPublisher(searchPublisher)
	}

	return skillsAndPricingWiring{
		ExpertiseSvc:          expertiseSvc,
		SkillSvc:              skillSvc,
		SkillHandler:          skillHandler,
		ProfilePricingSvc:     profilePricingSvc,
		ProfilePricingHandler: profilePricingHandler,
	}
}
