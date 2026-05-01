package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"marketplace-backend/internal/domain/organization"
	"marketplace-backend/internal/handler/middleware"
)

// mountSocialLinkRoutes wires the three persona-scoped social-link
// surfaces: legacy agency, freelance, and referrer. Each persona has
// its own public read endpoint and authenticated write subtree so the
// frontend can toggle visibility per identity.
func mountSocialLinkRoutes(r chi.Router, deps RouterDeps, auth func(http.Handler) http.Handler) {
	mountAgencySocialLinks(r, deps, auth)
	mountFreelanceSocialLinks(r, deps, auth)
	mountReferrerSocialLinks(r, deps, auth)
}

func mountAgencySocialLinks(r chi.Router, deps RouterDeps, auth func(http.Handler) http.Handler) {
	if deps.SocialLink == nil {
		return
	}
	// Public: read agency social links
	r.Get("/profiles/{orgId}/social-links", deps.SocialLink.ListPublicSocialLinks)

	// Authenticated: manage own agency social links
	r.Route("/profile/social-links", func(r chi.Router) {
		r.Use(auth)
		r.Use(middleware.NoCache)
		r.Get("/", deps.SocialLink.ListMySocialLinks)
		r.With(middleware.RequirePermission(organization.PermOrgProfileEdit)).Put("/", deps.SocialLink.UpsertSocialLink)
		r.With(middleware.RequirePermission(organization.PermOrgProfileEdit)).Delete("/{platform}", deps.SocialLink.DeleteSocialLink)
	})
}

func mountFreelanceSocialLinks(r chi.Router, deps RouterDeps, auth func(http.Handler) http.Handler) {
	if deps.FreelanceSocialLink == nil {
		return
	}
	// Freelance persona social link routes — independent set
	// scoped to the freelance identity of provider_personal users.
	r.Get("/freelance-profiles/{orgId}/social-links", deps.FreelanceSocialLink.ListPublicSocialLinks)

	r.Route("/freelance-profile/social-links", func(r chi.Router) {
		r.Use(auth)
		r.Use(middleware.NoCache)
		r.Get("/", deps.FreelanceSocialLink.ListMySocialLinks)
		r.With(middleware.RequirePermission(organization.PermOrgProfileEdit)).Put("/", deps.FreelanceSocialLink.UpsertSocialLink)
		r.With(middleware.RequirePermission(organization.PermOrgProfileEdit)).Delete("/{platform}", deps.FreelanceSocialLink.DeleteSocialLink)
	})
}

func mountReferrerSocialLinks(r chi.Router, deps RouterDeps, auth func(http.Handler) http.Handler) {
	if deps.ReferrerSocialLink == nil {
		return
	}
	// Referrer persona social link routes — independent set
	// scoped to the apporteur d'affaires identity.
	r.Get("/referrer-profiles/{orgId}/social-links", deps.ReferrerSocialLink.ListPublicSocialLinks)

	r.Route("/referrer-profile/social-links", func(r chi.Router) {
		r.Use(auth)
		r.Use(middleware.NoCache)
		r.Get("/", deps.ReferrerSocialLink.ListMySocialLinks)
		r.With(middleware.RequirePermission(organization.PermOrgProfileEdit)).Put("/", deps.ReferrerSocialLink.UpsertSocialLink)
		r.With(middleware.RequirePermission(organization.PermOrgProfileEdit)).Delete("/{platform}", deps.ReferrerSocialLink.DeleteSocialLink)
	})
}
