package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"marketplace-backend/internal/domain/organization"
	"marketplace-backend/internal/handler/middleware"
)

// mountProfileRoutes wires the legacy agency-style profile surface
// (/profile/*) plus the public-read counterparts and the persona-
// specific split-profile groups (/freelance-profile, /referrer-profile,
// /organization). Extracted from NewRouter for phase-3-F.
func mountProfileRoutes(r chi.Router, deps RouterDeps, auth func(http.Handler) http.Handler) {
	mountLegacyProfile(r, deps, auth)
	mountSplitProfilePersonas(r, deps, auth)
	mountSkillsCatalog(r, deps, auth)
	mountPublicProfileReads(r, deps)
}

func mountLegacyProfile(r chi.Router, deps RouterDeps, auth func(http.Handler) http.Handler) {
	r.Route("/profile", func(r chi.Router) {
		r.Use(auth)
		r.Use(middleware.NoCache)
		r.Get("/", deps.Profile.GetMyProfile)
		r.With(middleware.RequirePermission(organization.PermOrgProfileEdit)).Put("/", deps.Profile.UpdateMyProfile)
		// Expertise domains — same "edit profile" permission as the
		// main profile fields. The feature is hard-disabled for
		// enterprise orgs at the service layer (403 forbidden).
		r.With(middleware.RequirePermission(organization.PermOrgProfileEdit)).Put("/expertise", deps.Profile.UpdateMyExpertise)
		// Profile skills (authenticated). Same permission as expertise
		// — both are public-profile decorations shared by the whole
		// org. The feature is hard-disabled for enterprise orgs at
		// the service layer (403 forbidden).
		if deps.Skill != nil {
			r.Get("/skills", deps.Skill.GetMyProfileSkills)
			r.With(middleware.RequirePermission(organization.PermOrgProfileEdit)).Put("/skills", deps.Skill.PutMyProfileSkills)
		}
		// Profile Tier 1 completion (migration 083): location,
		// languages, availability blocks. Same edit-profile
		// permission as the main profile fields — all three
		// are public profile decorations shared by the whole org.
		r.With(middleware.RequirePermission(organization.PermOrgProfileEdit)).Put("/location", deps.Profile.UpdateMyLocation)
		r.With(middleware.RequirePermission(organization.PermOrgProfileEdit)).Put("/languages", deps.Profile.UpdateMyLanguages)
		r.With(middleware.RequirePermission(organization.PermOrgProfileEdit)).Put("/availability", deps.Profile.UpdateMyAvailability)

		// Profile pricing (migration 083). Wired through a
		// dedicated handler (ProfilePricingHandler) to preserve
		// the feature-isolation principle — deleting the
		// pricing feature means deleting that file + wiring
		// without touching ProfileHandler.
		if deps.ProfilePricing != nil {
			r.Get("/pricing", deps.ProfilePricing.ListMyPricing)
			r.With(middleware.RequirePermission(organization.PermOrgProfileEdit)).Put("/pricing", deps.ProfilePricing.UpsertMyPricing)
			r.With(middleware.RequirePermission(organization.PermOrgProfileEdit)).Delete("/pricing/{kind}", deps.ProfilePricing.DeleteMyPricingByKind)
		}

		// Client profile (migration 114) — the client-facing facet
		// of the org's public profile. Gated by a dedicated
		// permission (org_client_profile.edit) so an operator can
		// be trusted with the client profile without also having
		// write access to the provider-facing profile.
		if deps.ClientProfile != nil {
			r.With(middleware.RequirePermission(organization.PermOrgClientProfileEdit)).
				Put("/client", deps.ClientProfile.UpdateMyClientProfile)
		}
	})
}

func mountSplitProfilePersonas(r chi.Router, deps RouterDeps, auth func(http.Handler) http.Handler) {
	if deps.FreelanceProfile != nil {
		r.Route("/freelance-profile", func(r chi.Router) {
			r.Use(auth)
			r.Use(middleware.NoCache)
			r.Get("/", deps.FreelanceProfile.GetMy)
			r.With(middleware.RequirePermission(organization.PermOrgProfileEdit)).Put("/", deps.FreelanceProfile.UpdateMy)
			r.With(middleware.RequirePermission(organization.PermOrgProfileEdit)).Put("/availability", deps.FreelanceProfile.UpdateMyAvailability)
			r.With(middleware.RequirePermission(organization.PermOrgProfileEdit)).Put("/expertise", deps.FreelanceProfile.UpdateMyExpertise)
			if deps.FreelanceProfileVideo != nil {
				r.With(middleware.RequirePermission(organization.PermOrgProfileEdit)).Post("/video", deps.FreelanceProfileVideo.Upload)
				r.With(middleware.RequirePermission(organization.PermOrgProfileEdit)).Delete("/video", deps.FreelanceProfileVideo.Delete)
			}
			if deps.FreelancePricing != nil {
				r.Get("/pricing", deps.FreelancePricing.GetMy)
				r.With(middleware.RequirePermission(organization.PermOrgProfileEdit)).Put("/pricing", deps.FreelancePricing.UpsertMy)
				r.With(middleware.RequirePermission(organization.PermOrgProfileEdit)).Delete("/pricing", deps.FreelancePricing.DeleteMy)
			}
		})
	}
	if deps.ReferrerProfile != nil {
		r.Route("/referrer-profile", func(r chi.Router) {
			r.Use(auth)
			r.Use(middleware.NoCache)
			r.Get("/", deps.ReferrerProfile.GetMy)
			r.With(middleware.RequirePermission(organization.PermOrgProfileEdit)).Put("/", deps.ReferrerProfile.UpdateMy)
			r.With(middleware.RequirePermission(organization.PermOrgProfileEdit)).Put("/availability", deps.ReferrerProfile.UpdateMyAvailability)
			r.With(middleware.RequirePermission(organization.PermOrgProfileEdit)).Put("/expertise", deps.ReferrerProfile.UpdateMyExpertise)
			if deps.ReferrerProfileVideo != nil {
				r.With(middleware.RequirePermission(organization.PermOrgProfileEdit)).Post("/video", deps.ReferrerProfileVideo.Upload)
				r.With(middleware.RequirePermission(organization.PermOrgProfileEdit)).Delete("/video", deps.ReferrerProfileVideo.Delete)
			}
			if deps.ReferrerPricing != nil {
				r.Get("/pricing", deps.ReferrerPricing.GetMy)
				r.With(middleware.RequirePermission(organization.PermOrgProfileEdit)).Put("/pricing", deps.ReferrerPricing.UpsertMy)
				r.With(middleware.RequirePermission(organization.PermOrgProfileEdit)).Delete("/pricing", deps.ReferrerPricing.DeleteMy)
			}
		})
	}
	if deps.OrganizationShared != nil {
		r.Route("/organization", func(r chi.Router) {
			r.Use(auth)
			r.Use(middleware.NoCache)
			r.Get("/shared", deps.OrganizationShared.GetSharedProfile)
			r.With(middleware.RequirePermission(organization.PermOrgProfileEdit)).Put("/location", deps.OrganizationShared.UpdateLocation)
			r.With(middleware.RequirePermission(organization.PermOrgProfileEdit)).Put("/languages", deps.OrganizationShared.UpdateLanguages)
			r.With(middleware.RequirePermission(organization.PermOrgProfileEdit)).Put("/photo", deps.OrganizationShared.UpdatePhoto)
		})
	}
}

func mountSkillsCatalog(r chi.Router, deps RouterDeps, auth func(http.Handler) http.Handler) {
	if deps.Skill == nil {
		return
	}
	// Public catalog reads — no auth required so the discovery
	// UI can surface skills to anonymous visitors.
	r.Get("/skills/catalog", deps.Skill.GetCuratedByExpertise)
	r.Get("/skills/autocomplete", deps.Skill.Autocomplete)

	// Authenticated: create a new user-contributed skill from
	// the "Create X" autocomplete option. Permission-gated by
	// the same edit-profile grant as the profile skills PUT.
	r.Group(func(r chi.Router) {
		r.Use(auth)
		r.Use(middleware.NoCache)
		r.With(middleware.RequirePermission(organization.PermOrgProfileEdit)).Post("/skills", deps.Skill.CreateUserSkill)
	})
}

func mountPublicProfileReads(r chi.Router, deps RouterDeps) {
	// Public profiles (keyed by organization id since phase R2)
	r.Get("/profiles/search", deps.Profile.SearchProfiles)
	r.Get("/profiles/{orgId}", deps.Profile.GetPublicProfile)

	// Public client profile (migration 114). Keyed on organization
	// id so the URL scheme stays symmetrical with /profiles/{orgId}.
	// Nil ClientProfile handler disables the route entirely —
	// feature-isolation rule.
	if deps.ClientProfile != nil {
		r.Get("/clients/{orgId}", deps.ClientProfile.GetPublicClientProfile)
	}

	if deps.ProjectHistory != nil {
		r.Get("/profiles/{orgId}/project-history", deps.ProjectHistory.ListByOrganization)
	}

	// Public read routes for the split-profile personas
	// (provider_personal orgs only). Keyed by organization_id
	// so the URL scheme stays symmetrical with the legacy
	// /profiles/{orgId} and the frontend's existing routes.
	if deps.FreelanceProfile != nil {
		r.Get("/freelance-profiles/{orgID}", deps.FreelanceProfile.GetPublic)
	}
	if deps.ReferrerProfile != nil {
		r.Get("/referrer-profiles/{orgID}", deps.ReferrerProfile.GetPublic)
		// Apporteur reputation surface — keyed on orgID for URL
		// symmetry with the rest of the referrer-profile read
		// surface. The handler translates internally to the
		// owner user_id because referrals reference users.
		r.Get("/referrer-profiles/{orgID}/reputation", deps.ReferrerProfile.GetReputation)
	}
}
