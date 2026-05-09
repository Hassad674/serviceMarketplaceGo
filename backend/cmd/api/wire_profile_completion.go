package main

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	"marketplace-backend/internal/adapter/postgres"
	"marketplace-backend/internal/app/profilecompletion"
	"marketplace-backend/internal/domain/freelancepricing"
	"marketplace-backend/internal/domain/freelanceprofile"
	"marketplace-backend/internal/domain/profile"
	"marketplace-backend/internal/domain/referrerpricing"
	"marketplace-backend/internal/domain/referrerprofile"
	"marketplace-backend/internal/handler"
	"marketplace-backend/internal/port/repository"
)

// profileCompletionDeps groups the existing repositories the handler
// composes into the completion report. Most fields are the concrete
// adapter pointers from the infrastructure bag — adapter-level types
// satisfy the narrow reader interfaces below through small typed
// wrappers, so the profilecompletion app package stays decoupled
// from postgres.
//
// The DB handle is the legacy `*sql.DB` pool from infrastructure.DB.
// Adapters that are not exposed at the infra layer (skill, social
// link, portfolio, billing profile, pricings, profile pricing) are
// constructed here on the fly — the constructors are stateless and
// the cost of one allocation per app boot is negligible.
type profileCompletionDeps struct {
	DB                   *sql.DB
	UserRepo             *postgres.UserRepository
	OrganizationRepo     *postgres.OrganizationRepository
	ProfileRepo          *postgres.ProfileRepository
	FreelanceProfileRepo *postgres.FreelanceProfileRepository
	ReferrerProfileRepo  *postgres.ReferrerProfileRepository
}

// wireProfileCompletion builds the completion service + handler. The
// adapters below wrap the existing postgres repositories so the
// service stays decoupled from postgres.
//
// Stateless repository constructors (skills, social links, portfolio,
// billing profile, freelance/referrer/legacy pricing) are invoked
// here rather than threaded from upstream wire functions — the cost
// is one allocation per boot, and inlining them keeps this wire file
// self-contained: deleting the feature is a clean removal of one
// file plus the wiring call.
func wireProfileCompletion(deps profileCompletionDeps) *handler.ProfileCompletionHandler {
	skillRepo := postgres.NewProfileSkillRepository(deps.DB)
	socialRepo := postgres.NewSocialLinkRepository(deps.DB)
	portfolioRepo := postgres.NewPortfolioRepository(deps.DB)
	freelancePricing := postgres.NewFreelancePricingRepository(deps.DB)
	referrerPricing := postgres.NewReferrerPricingRepository(deps.DB)
	legacyPricing := postgres.NewProfilePricingRepository(deps.DB)

	svc, err := profilecompletion.NewService(profilecompletion.Deps{
		Users:            deps.UserRepo,
		Organizations:    deps.OrganizationRepo,
		Shared:           sharedProfileReaderAdapter{repo: deps.OrganizationRepo},
		FreelanceProfile: freelanceProfileReaderAdapter{repo: deps.FreelanceProfileRepo},
		ReferrerProfile:  referrerProfileReaderAdapter{repo: deps.ReferrerProfileRepo},
		LegacyProfile:    legacyProfileReaderAdapter{repo: deps.ProfileRepo},
		Skills:           skillsCounterAdapter{repo: skillRepo},
		SocialLinks:      socialLinksCounterAdapter{repo: socialRepo},
		Portfolio:        portfolioCounterAdapter{repo: portfolioRepo},
		FreelancePricing: freelancePricingExistsAdapter{repo: freelancePricing},
		ReferrerPricing:  referrerPricingExistsAdapter{repo: referrerPricing},
		LegacyPricing:    legacyPricingCounterAdapter{repo: legacyPricing},
	})
	if err != nil {
		// Required readers (Users + Organizations) cannot be nil at
		// production wiring — a nil here is a programming bug, not a
		// runtime concern.
		panic("wireProfileCompletion: " + err.Error())
	}
	return handler.NewProfileCompletionHandler(svc)
}

// ------------------------------------------------------------------
// Adapter shims — each one maps the wider repository interface to
// the narrow reader the profilecompletion service expects. Defined
// here (in cmd/api/) rather than in the app package so the app stays
// independent of postgres types.
// ------------------------------------------------------------------

type sharedProfileReaderAdapter struct {
	repo *postgres.OrganizationRepository
}

func (a sharedProfileReaderAdapter) GetSharedProfile(ctx context.Context, orgID uuid.UUID) (*profilecompletion.SharedProfile, error) {
	if a.repo == nil {
		return nil, profilecompletion.ErrNotFound
	}
	v, err := a.repo.GetSharedProfile(ctx, orgID)
	if err != nil {
		return nil, err
	}
	if v == nil {
		return nil, profilecompletion.ErrNotFound
	}
	return &profilecompletion.SharedProfile{
		PhotoURL:              v.PhotoURL,
		City:                  v.City,
		CountryCode:           v.CountryCode,
		LanguagesProfessional: v.LanguagesProfessional,
	}, nil
}

type freelanceProfileReaderAdapter struct {
	repo *postgres.FreelanceProfileRepository
}

func (a freelanceProfileReaderAdapter) GetByOrgID(ctx context.Context, orgID uuid.UUID) (*profilecompletion.FreelanceProfileSnapshot, error) {
	if a.repo == nil {
		return nil, profilecompletion.ErrNotFound
	}
	view, err := a.repo.GetByOrgID(ctx, orgID)
	if err != nil {
		if errors.Is(err, freelanceprofile.ErrProfileNotFound) {
			return nil, profilecompletion.ErrNotFound
		}
		return nil, err
	}
	if view == nil || view.Profile == nil {
		return nil, profilecompletion.ErrNotFound
	}
	return &profilecompletion.FreelanceProfileSnapshot{
		ProfileID:          view.Profile.ID,
		Title:              view.Profile.Title,
		About:              view.Profile.About,
		VideoURL:           view.Profile.VideoURL,
		AvailabilityStatus: view.Profile.AvailabilityStatus,
		ExpertiseDomains:   view.Profile.ExpertiseDomains,
	}, nil
}

type referrerProfileReaderAdapter struct {
	repo *postgres.ReferrerProfileRepository
}

// GetByOrgID delegates to GetOrCreateByOrgID because the referrer
// repository deliberately does not expose a strict read — every
// owner-side touch lazily inserts a default row. The completion
// service never calls this adapter for non-referrer personas (the
// dispatch in builders.go gates on persona), so the side effect of
// creating an empty referrer row on the first report read is
// confined to provider_personal orgs whose user has the referrer
// toggle enabled — exactly the same group the existing referrer
// service auto-creates rows for on every other read path.
func (a referrerProfileReaderAdapter) GetByOrgID(ctx context.Context, orgID uuid.UUID) (*profilecompletion.ReferrerProfileSnapshot, error) {
	if a.repo == nil {
		return nil, profilecompletion.ErrNotFound
	}
	view, err := a.repo.GetOrCreateByOrgID(ctx, orgID)
	if err != nil {
		if errors.Is(err, referrerprofile.ErrProfileNotFound) {
			return nil, profilecompletion.ErrNotFound
		}
		return nil, err
	}
	if view == nil || view.Profile == nil {
		return nil, profilecompletion.ErrNotFound
	}
	return &profilecompletion.ReferrerProfileSnapshot{
		ProfileID:          view.Profile.ID,
		Title:              view.Profile.Title,
		About:              view.Profile.About,
		VideoURL:           view.Profile.VideoURL,
		AvailabilityStatus: view.Profile.AvailabilityStatus,
		ExpertiseDomains:   view.Profile.ExpertiseDomains,
	}, nil
}

type legacyProfileReaderAdapter struct {
	repo *postgres.ProfileRepository
}

func (a legacyProfileReaderAdapter) GetByOrgID(ctx context.Context, orgID uuid.UUID) (*profile.Profile, error) {
	if a.repo == nil {
		return nil, profilecompletion.ErrNotFound
	}
	p, err := a.repo.GetByOrganizationID(ctx, orgID)
	if err != nil {
		if errors.Is(err, profile.ErrProfileNotFound) {
			return nil, profilecompletion.ErrNotFound
		}
		return nil, err
	}
	return p, nil
}

type skillsCounterAdapter struct {
	repo *postgres.ProfileSkillRepository
}

func (a skillsCounterAdapter) CountByOrg(ctx context.Context, orgID uuid.UUID) (int, error) {
	if a.repo == nil {
		return 0, nil
	}
	return a.repo.CountByOrg(ctx, orgID)
}

type socialLinksCounterAdapter struct {
	repo *postgres.SocialLinkRepository
}

// CountByOrgPersona issues ONE query per persona by listing the rows
// and taking len. The legacy repository does not expose a count
// method and adding one would require a migration of every adapter
// + the port interface. Listing here is acceptable: every persona
// holds at most a handful of links (UI caps at 5) so the response is
// cheap. If volume grows the adapter can switch to a SELECT count(*).
func (a socialLinksCounterAdapter) CountByOrgPersona(ctx context.Context, orgID uuid.UUID, persona profile.SocialLinkPersona) (int, error) {
	if a.repo == nil {
		return 0, nil
	}
	links, err := a.repo.ListByOrganizationPersona(ctx, orgID, persona)
	if err != nil {
		return 0, err
	}
	return len(links), nil
}

type portfolioCounterAdapter struct {
	repo *postgres.PortfolioRepository
}

func (a portfolioCounterAdapter) CountByOrganization(ctx context.Context, orgID uuid.UUID) (int, error) {
	if a.repo == nil {
		return 0, nil
	}
	return a.repo.CountByOrganization(ctx, orgID)
}

type freelancePricingExistsAdapter struct {
	repo *postgres.FreelancePricingRepository
}

func (a freelancePricingExistsAdapter) ExistsByProfileID(ctx context.Context, profileID uuid.UUID) (bool, error) {
	if a.repo == nil {
		return false, nil
	}
	row, err := a.repo.FindByProfileID(ctx, profileID)
	if err != nil {
		if errors.Is(err, freelancepricing.ErrPricingNotFound) {
			return false, nil
		}
		return false, err
	}
	return row != nil, nil
}

type referrerPricingExistsAdapter struct {
	repo *postgres.ReferrerPricingRepository
}

func (a referrerPricingExistsAdapter) ExistsByProfileID(ctx context.Context, profileID uuid.UUID) (bool, error) {
	if a.repo == nil {
		return false, nil
	}
	row, err := a.repo.FindByProfileID(ctx, profileID)
	if err != nil {
		if errors.Is(err, referrerpricing.ErrPricingNotFound) {
			return false, nil
		}
		return false, err
	}
	return row != nil, nil
}

type legacyPricingCounterAdapter struct {
	repo *postgres.ProfilePricingRepository
}

func (a legacyPricingCounterAdapter) CountByOrgID(ctx context.Context, orgID uuid.UUID) (int, error) {
	if a.repo == nil {
		return 0, nil
	}
	rows, err := a.repo.FindByOrgID(ctx, orgID)
	if err != nil {
		return 0, err
	}
	return len(rows), nil
}

// Compile-time interface checks — fail fast at build time when an
// adapter signature drifts from the contract it claims to satisfy.
var (
	_ profilecompletion.UserReader             = (*postgres.UserRepository)(nil)
	_ profilecompletion.OrganizationReader     = (*postgres.OrganizationRepository)(nil)
	_ profilecompletion.SharedProfileReader    = sharedProfileReaderAdapter{}
	_ profilecompletion.FreelanceProfileReader = freelanceProfileReaderAdapter{}
	_ profilecompletion.ReferrerProfileReader  = referrerProfileReaderAdapter{}
	_ profilecompletion.LegacyProfileReader    = legacyProfileReaderAdapter{}
	_ profilecompletion.SkillsCounter          = skillsCounterAdapter{}
	_ profilecompletion.SocialLinksCounter     = socialLinksCounterAdapter{}
	_ profilecompletion.PortfolioCounter       = portfolioCounterAdapter{}
	_ profilecompletion.FreelancePricingReader = freelancePricingExistsAdapter{}
	_ profilecompletion.ReferrerPricingReader  = referrerPricingExistsAdapter{}
	_ profilecompletion.LegacyPricingCounter   = legacyPricingCounterAdapter{}

	// Ensure repository.OrganizationRepository continues to satisfy
	// FindByID even when a future signature change reorders return
	// values — the assertion costs nothing and catches drift early.
	_ repository.OrganizationRepository = (*postgres.OrganizationRepository)(nil)
)
