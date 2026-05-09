package profilecompletion

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/organization"
	"marketplace-backend/internal/domain/profile"
	"marketplace-backend/internal/domain/user"
)

// labelKeyFor returns the i18n bucket the frontend looks up for the
// given section. Keys live under "profile.completion.section.*" in
// messages/{fr,en}.json (web) and the matching ARB on mobile so the
// label list is centralized — the backend never ships translated
// strings.
func labelKeyFor(k SectionKey) string {
	return "profile.completion.section." + string(k)
}

// completionPathFor returns the in-app URL the frontend opens when
// the user clicks the section in the missing-list modal. Hardcoded
// per persona so the backend stays the single source of truth.
// The paths are relative — the frontend prepends its current locale
// prefix (/fr, /en) when navigating.
func completionPathFor(persona Persona, k SectionKey) string {
	freelancePaths := map[SectionKey]string{
		SectionPhoto:        "/dashboard/profile/edit",
		SectionTitle:        "/dashboard/profile/edit",
		SectionAbout:        "/dashboard/profile/edit",
		SectionExpertises:   "/dashboard/profile/expertise",
		SectionSkills:       "/dashboard/profile/skills",
		SectionPricing:      "/dashboard/profile/pricing",
		SectionAvailability: "/dashboard/profile/availability",
		SectionLocation:     "/dashboard/profile/location",
		SectionLanguages:    "/dashboard/profile/languages",
		SectionVideo:        "/dashboard/profile/video",
		SectionSocialLinks:  "/dashboard/profile/social",
		SectionPortfolio:    "/dashboard/portfolio",
		SectionClientAbout:  "/dashboard/profile/client",
	}
	referrerPaths := map[SectionKey]string{
		SectionPhoto:        "/dashboard/referrer/edit",
		SectionTitle:        "/dashboard/referrer/edit",
		SectionAbout:        "/dashboard/referrer/edit",
		SectionExpertises:   "/dashboard/referrer/expertise",
		SectionPricing:      "/dashboard/referrer/pricing",
		SectionAvailability: "/dashboard/referrer/availability",
		SectionVideo:        "/dashboard/referrer/video",
		SectionSocialLinks:  "/dashboard/referrer/social",
	}
	enterprisePaths := map[SectionKey]string{
		SectionPhoto:       "/dashboard/profile/edit",
		SectionAbout:       "/dashboard/profile/edit",
		SectionClientAbout: "/dashboard/profile/client",
	}

	switch persona {
	case PersonaFreelance, PersonaAgency:
		if v, ok := freelancePaths[k]; ok {
			return v
		}
	case PersonaReferrer:
		if v, ok := referrerPaths[k]; ok {
			return v
		}
	case PersonaEnterprise:
		if v, ok := enterprisePaths[k]; ok {
			return v
		}
	}
	return "/dashboard/profile/edit"
}

// section is a small helper that wraps the per-row construction so
// each builder reads as a flat list.
func section(persona Persona, k SectionKey, filled bool) Section {
	return Section{
		Key:            k,
		Filled:         filled,
		LabelKey:       labelKeyFor(k),
		CompletionPath: completionPathFor(persona, k),
	}
}

// buildSections is the persona dispatch — every persona has its own
// list builder so a future change to one role's checklist cannot
// touch the others.
func (s *Service) buildSections(
	ctx context.Context,
	u *user.User,
	org *organization.Organization,
	persona Persona,
) ([]Section, error) {
	switch persona {
	case PersonaFreelance:
		return s.buildFreelanceSections(ctx, u, org)
	case PersonaReferrer:
		return s.buildReferrerSections(ctx, u, org)
	case PersonaEnterprise:
		return s.buildEnterpriseSections(ctx, u, org)
	case PersonaAgency:
		return s.buildAgencySections(ctx, u, org)
	}
	return s.buildAgencySections(ctx, u, org)
}

// snapshotBundle aggregates the readers' answers in a single pass so
// each builder can compose its list without scattering nil-checks.
// All fields are optional — every nil branch silently maps to "empty".
type snapshotBundle struct {
	Shared           *SharedProfile
	Freelance        *FreelanceProfileSnapshot
	Referrer         *ReferrerProfileSnapshot
	Legacy           *profile.Profile
	SkillCount       int
	SocialFreelance  int
	SocialReferrer   int
	SocialAgency     int
	PortfolioCount   int
	FreelancePricing bool
	ReferrerPricing  bool
	LegacyPricingN   int
}

// loadSnapshot fans out to every reader concurrently-safe (sequential
// — readers are expected to be cheap point reads with their own 5s
// context timeouts at the adapter layer). Errors that are not
// ErrNotFound propagate so a transient DB blip surfaces as 500
// instead of producing a misleading 0% report.
func (s *Service) loadSnapshot(
	ctx context.Context,
	u *user.User,
	org *organization.Organization,
) (*snapshotBundle, error) {
	out := &snapshotBundle{}

	if s.deps.Shared != nil {
		shared, err := s.deps.Shared.GetSharedProfile(ctx, org.ID)
		if err != nil && !errors.Is(err, ErrNotFound) {
			return nil, err
		}
		out.Shared = shared
	}
	if s.deps.LegacyProfile != nil {
		legacy, err := s.deps.LegacyProfile.GetByOrgID(ctx, org.ID)
		if err != nil && !errors.Is(err, ErrNotFound) {
			return nil, err
		}
		out.Legacy = legacy
	}
	if s.deps.FreelanceProfile != nil {
		fp, err := s.deps.FreelanceProfile.GetByOrgID(ctx, org.ID)
		if err != nil && !errors.Is(err, ErrNotFound) {
			return nil, err
		}
		out.Freelance = fp
	}
	if s.deps.ReferrerProfile != nil {
		rp, err := s.deps.ReferrerProfile.GetByOrgID(ctx, org.ID)
		if err != nil && !errors.Is(err, ErrNotFound) {
			return nil, err
		}
		out.Referrer = rp
	}
	if err := s.fillCounts(ctx, u, org, out); err != nil {
		return nil, err
	}
	return out, nil
}

// fillCounts populates the count + boolean fields of the snapshot
// bundle. Split off from loadSnapshot to keep both functions under
// the 50-line ceiling.
func (s *Service) fillCounts(
	ctx context.Context,
	_ *user.User,
	org *organization.Organization,
	out *snapshotBundle,
) error {
	if s.deps.Skills != nil {
		n, err := s.deps.Skills.CountByOrg(ctx, org.ID)
		if err != nil {
			return err
		}
		out.SkillCount = n
	}
	if s.deps.SocialLinks != nil {
		nf, err := s.deps.SocialLinks.CountByOrgPersona(ctx, org.ID, profile.PersonaFreelance)
		if err != nil {
			return err
		}
		out.SocialFreelance = nf
		nr, err := s.deps.SocialLinks.CountByOrgPersona(ctx, org.ID, profile.PersonaReferrer)
		if err != nil {
			return err
		}
		out.SocialReferrer = nr
		na, err := s.deps.SocialLinks.CountByOrgPersona(ctx, org.ID, profile.PersonaAgency)
		if err != nil {
			return err
		}
		out.SocialAgency = na
	}
	if s.deps.Portfolio != nil {
		n, err := s.deps.Portfolio.CountByOrganization(ctx, org.ID)
		if err != nil {
			return err
		}
		out.PortfolioCount = n
	}
	if err := s.fillPricing(ctx, out, org.ID); err != nil {
		return err
	}
	return nil
}

// fillPricing populates the per-persona pricing booleans. Pricing
// readers are guarded against a nil freelance/referrer snapshot (no
// profile id to query) — the section then defaults to "empty".
//
// Billing/KYC sections were dropped from every persona checklist —
// billing info is captured inline at first payment and KYC has its
// own dedicated flow, so neither is queried here anymore.
func (s *Service) fillPricing(
	ctx context.Context,
	out *snapshotBundle,
	orgID uuid.UUID,
) error {
	if s.deps.FreelancePricing != nil && out.Freelance != nil &&
		out.Freelance.ProfileID != uuid.Nil {
		ok, err := s.deps.FreelancePricing.ExistsByProfileID(ctx, out.Freelance.ProfileID)
		if err != nil {
			return err
		}
		out.FreelancePricing = ok
	}
	if s.deps.ReferrerPricing != nil && out.Referrer != nil &&
		out.Referrer.ProfileID != uuid.Nil {
		ok, err := s.deps.ReferrerPricing.ExistsByProfileID(ctx, out.Referrer.ProfileID)
		if err != nil {
			return err
		}
		out.ReferrerPricing = ok
	}
	if s.deps.LegacyPricing != nil {
		n, err := s.deps.LegacyPricing.CountByOrgID(ctx, orgID)
		if err != nil {
			return err
		}
		out.LegacyPricingN = n
	}
	return nil
}

// hasLocation reports whether the org has filled the city + country.
// Latitude / longitude / work mode are NOT required — they are
// optional decorations.
func hasLocation(s *SharedProfile) bool {
	if s == nil {
		return false
	}
	return strings.TrimSpace(s.City) != "" && strings.TrimSpace(s.CountryCode) != ""
}

// hasLanguages reports whether at least one professional language is
// declared. Conversational languages are bonus — not required for
// completion.
func hasLanguages(s *SharedProfile) bool {
	if s == nil {
		return false
	}
	return len(s.LanguagesProfessional) > 0
}

// hasPhoto reports whether the shared block has a non-empty photo URL.
func hasPhoto(s *SharedProfile) bool {
	if s == nil {
		return false
	}
	return strings.TrimSpace(s.PhotoURL) != ""
}
