package main

import (
	"database/sql"

	"marketplace-backend/internal/adapter/postgres"
	freelancepricingapp "marketplace-backend/internal/app/freelancepricing"
	freelanceprofileapp "marketplace-backend/internal/app/freelanceprofile"
	profileapp "marketplace-backend/internal/app/profile"
	referrerpricingapp "marketplace-backend/internal/app/referrerpricing"
	referrerprofileapp "marketplace-backend/internal/app/referrerprofile"
	"marketplace-backend/internal/app/searchindex"
	"marketplace-backend/internal/handler"
	"marketplace-backend/internal/port/repository"
	skillReader "marketplace-backend/internal/handler"
)

// personasWiring carries the products of the split-profile constellation
// (freelance + referrer aggregates) introduced by migrations 096-104.
//
// FreelanceProfileSvc is consumed by the referral feature for the thin
// snapshot loader; FreelanceProfileRepo is consumed by the referral
// service constructor. Both are exposed so main.go can thread them
// downstream without re-resolving from the wiring struct.
//
// ProfileSvc is returned re-bound: when the search publisher is wired,
// the legacy agency profile service receives a fluent
// WithSearchIndexPublisher / WithTxRunner setter (Phase 2 carry-over).
// main.go must keep the new pointer so every downstream consumer sees
// the same outbox-aware service.
type personasWiring struct {
	ProfileSvc                  *profileapp.Service // re-bound when searchPublisher != nil
	FreelanceProfileSvc         *freelanceprofileapp.Service
	FreelanceProfileRepo        *postgres.FreelanceProfileRepository
	FreelancePricingSvc         *freelancepricingapp.Service
	ReferrerProfileSvc          *referrerprofileapp.Service
	ReferrerProfileRepo         *postgres.ReferrerProfileRepository
	ReferrerPricingSvc          *referrerpricingapp.Service
	FreelanceProfileHandler     *handler.FreelanceProfileHandler
	FreelancePricingHandler     *handler.FreelancePricingHandler
}

// personasDeps captures the upstream dependencies needed by the
// split-profile wiring. profileSvc is mutated in place (returned
// re-bound) when the search publisher is non-nil — main.go must
// reassign its profileSvc local from the returned wiring.
type personasDeps struct {
	DB              *sql.DB
	ProfileSvc      *profileapp.Service
	SearchPublisher *searchindex.Publisher
	TxRunner        repository.TxRunner
	SkillsReader    skillReader.SkillsReader
}

// wirePersonas brings up the freelance + referrer split-profile
// constellation. Each persona has its own repo / service / handler
// chain so deleting the split is a single-block deletion.
//
// Phase 2 carry-over: the legacy agency profile service also
// publishes reindex events when searchPublisher is wired. Done via a
// fluent setter here (the legacy profileSvc is created upstream for
// other downstream wiring); the helper returns the re-bound pointer
// so main.go can keep using the outbox-aware version everywhere.
func wirePersonas(deps personasDeps) personasWiring {
	freelanceProfileRepo := postgres.NewFreelanceProfileRepository(deps.DB)
	freelanceProfileSvc := freelanceprofileapp.NewService(freelanceProfileRepo)
	profileSvc := deps.ProfileSvc
	if deps.SearchPublisher != nil {
		freelanceProfileSvc = freelanceProfileSvc.
			WithSearchIndexPublisher(deps.SearchPublisher).
			WithTxRunner(deps.TxRunner)
		profileSvc = profileSvc.
			WithSearchIndexPublisher(deps.SearchPublisher).
			WithTxRunner(deps.TxRunner)
	}
	freelancePricingRepo := postgres.NewFreelancePricingRepository(deps.DB)
	freelancePricingSvc := freelancepricingapp.NewService(freelancePricingRepo)
	freelanceProfileHandler := handler.
		NewFreelanceProfileHandler(freelanceProfileSvc).
		WithSkillsReader(deps.SkillsReader).
		WithPricingReader(freelancePricingSvc)
	freelancePricingHandler := handler.NewFreelancePricingHandler(freelancePricingSvc, freelanceProfileSvc)
	if deps.SearchPublisher != nil {
		freelancePricingHandler = freelancePricingHandler.WithSearchIndexPublisher(deps.SearchPublisher)
	}

	referrerProfileRepo := postgres.NewReferrerProfileRepository(deps.DB)
	referrerProfileSvc := referrerprofileapp.NewService(referrerProfileRepo)
	if deps.SearchPublisher != nil {
		referrerProfileSvc = referrerProfileSvc.WithSearchIndexPublisher(deps.SearchPublisher)
	}
	referrerPricingRepo := postgres.NewReferrerPricingRepository(deps.DB)
	referrerPricingSvc := referrerpricingapp.NewService(referrerPricingRepo)

	return personasWiring{
		ProfileSvc:              profileSvc,
		FreelanceProfileSvc:     freelanceProfileSvc,
		FreelanceProfileRepo:    freelanceProfileRepo,
		FreelancePricingSvc:     freelancePricingSvc,
		ReferrerProfileSvc:      referrerProfileSvc,
		ReferrerProfileRepo:     referrerProfileRepo,
		ReferrerPricingSvc:      referrerPricingSvc,
		FreelanceProfileHandler: freelanceProfileHandler,
		FreelancePricingHandler: freelancePricingHandler,
	}
}

// finaliseReferrerHandlers binds the apporteur reputation aggregate
// onto the persona service and produces the referrer-side handlers.
// Split off from wirePersonas because the reputation deps include
// the referral repo, which doesn't exist until wireReferral has run.
//
// referrerPricingHandler optionally publishes reindex events; the
// helper takes searchPublisher to keep the call graph self-contained.
type referrerReputationDeps struct {
	ReferrerProfileSvc *referrerprofileapp.Service
	ReferrerPricingSvc *referrerpricingapp.Service
	Reputation         referrerprofileapp.ReputationDeps
	OrgOwnerLookup     handler.OrgOwnerLookup
	SearchPublisher    *searchindex.Publisher
}

type referrerReputationWiring struct {
	ProfileHandler *handler.ReferrerProfileHandler
	PricingHandler *handler.ReferrerPricingHandler
	Service        *referrerprofileapp.Service // re-bound with reputation deps
}

func finaliseReferrerHandlers(deps referrerReputationDeps) referrerReputationWiring {
	svc := deps.ReferrerProfileSvc.WithReputationDeps(deps.Reputation)
	profileHandler := handler.
		NewReferrerProfileHandler(svc).
		WithPricingReader(deps.ReferrerPricingSvc).
		WithOrgOwnerLookup(deps.OrgOwnerLookup)
	pricingHandler := handler.NewReferrerPricingHandler(deps.ReferrerPricingSvc, svc)
	if deps.SearchPublisher != nil {
		pricingHandler = pricingHandler.WithSearchIndexPublisher(deps.SearchPublisher)
	}
	return referrerReputationWiring{
		ProfileHandler: profileHandler,
		PricingHandler: pricingHandler,
		Service:        svc,
	}
}

// orgRepoForPersonas narrows the *postgres.OrganizationRepository to
// the read methods the persona handlers need. Currently unused as the
// helpers above receive the full repo via positional arg, but kept
// here as a documentation hook.
type orgRepoForPersonas interface {
	repository.OrganizationRepository
}

var _ orgRepoForPersonas = (*postgres.OrganizationRepository)(nil)
