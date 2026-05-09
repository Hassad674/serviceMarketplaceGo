// Package profilecompletion computes the "profile filled at X%" report
// surfaced on the /me/profile/completion endpoint and rendered as a
// progress bar on the web sidebar, mobile account screen, and every
// profile page header. The report enumerates every required section
// for the org's role, marks each as filled / empty, and ships the
// machine key + i18n label key so the frontend can render targeted
// "fill me" prompts without duplicating the role-aware section list.
//
// The package is fully independent — it does not import any other
// feature service. Every read goes through narrow reader interfaces
// the wiring layer satisfies with the existing repositories. Removing
// the feature is a pure deletion: drop this package, drop the handler,
// drop the wire_profile_completion.go file and the route line in
// routes_profile.go.
package profilecompletion

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/organization"
	"marketplace-backend/internal/domain/profile"
	"marketplace-backend/internal/domain/user"
)

// SectionKey is the machine identifier for a completion section. Used
// as the `key` field in the API response and as the lookup key for
// i18n labels on the frontend. Stable across releases — renaming a
// key is a breaking API change.
type SectionKey string

const (
	SectionPhoto        SectionKey = "photo"
	SectionTitle        SectionKey = "title"
	SectionAbout        SectionKey = "about"
	SectionExpertises   SectionKey = "expertises"
	SectionSkills       SectionKey = "skills"
	SectionPricing      SectionKey = "pricing"
	SectionAvailability SectionKey = "availability"
	SectionLocation     SectionKey = "location"
	SectionLanguages    SectionKey = "languages"
	SectionVideo        SectionKey = "video"
	SectionSocialLinks  SectionKey = "social_links"
	SectionPortfolio    SectionKey = "portfolio"
	SectionClientAbout  SectionKey = "client_about"
)

// Section is one row of the completion report. The label_key is the
// frontend i18n bucket (e.g. "profile.completion.section.title"); the
// completion_path is the in-app URL the frontend opens when the user
// clicks the section in the missing-list modal.
type Section struct {
	Key            SectionKey `json:"key"`
	Filled         bool       `json:"filled"`
	LabelKey       string     `json:"label_key"`
	CompletionPath string     `json:"completion_path"`
}

// Report is the response payload. Sections are ordered by domain
// precedence (identity -> presentation -> offer -> compliance) so the
// frontend can render the missing list in the same intuitive order
// without any client-side sort.
type Report struct {
	Role            string    `json:"role"`
	Persona         string    `json:"persona"`
	Percent         int       `json:"percent"`
	TotalSections   int       `json:"total_sections"`
	FilledSections  int       `json:"filled_sections"`
	Sections        []Section `json:"sections"`
}

// Persona enumerates the offering facet a completion report scopes
// to. provider_personal orgs surface the freelance persona by default;
// the referrer persona is computed when the user has the referrer
// toggle enabled (then a second computation can be requested by
// passing PersonaReferrer to ComputeWithPersona — kept available for
// future product surface even though v1 only exposes the freelance
// view at /me/profile/completion).
type Persona string

const (
	PersonaFreelance  Persona = "freelance"
	PersonaReferrer   Persona = "referrer"
	PersonaAgency     Persona = "agency"
	PersonaEnterprise Persona = "enterprise"
)

// Inputs

// UserReader is the narrow read contract the service needs to resolve
// the caller's role + referrer toggle. Defined locally so the package
// stays independent of the wider UserRepository surface.
type UserReader interface {
	GetByID(ctx context.Context, id uuid.UUID) (*user.User, error)
}

// OrganizationReader is the narrow read contract the service needs to
// resolve the org type + KYC state.
type OrganizationReader interface {
	FindByID(ctx context.Context, id uuid.UUID) (*organization.Organization, error)
}

// SharedProfileReader returns the org's shared-profile block (photo,
// city/country, languages). Used by every persona — the photo is
// shared across freelance, referrer, and agency views.
type SharedProfileReader interface {
	GetSharedProfile(ctx context.Context, orgID uuid.UUID) (*SharedProfile, error)
}

// SharedProfile is a transport-only bundle that mirrors the relevant
// columns of port/repository/OrganizationSharedProfile WITHOUT
// importing that package — the wiring layer adapts the wider port
// type to this narrow shape so the service stays decoupled from the
// repository interface surface. The fields are limited to what the
// completion report actually inspects.
type SharedProfile struct {
	PhotoURL              string
	City                  string
	CountryCode           string
	LanguagesProfessional []string
}

// FreelanceProfileReader returns the freelance persona's editable
// fields for an org. Returns ErrNotFound when no row exists; the
// service treats that case as "every freelance section empty".
type FreelanceProfileReader interface {
	GetByOrgID(ctx context.Context, orgID uuid.UUID) (*FreelanceProfileSnapshot, error)
}

// FreelanceProfileSnapshot is a transport-only bundle of the freelance
// columns the completion report inspects. Wiring layer maps the port
// repository.FreelanceProfileView onto this narrow shape.
type FreelanceProfileSnapshot struct {
	ProfileID          uuid.UUID
	Title              string
	About              string
	VideoURL           string
	AvailabilityStatus profile.AvailabilityStatus
	ExpertiseDomains   []string
}

// ReferrerProfileReader returns the referrer persona's editable
// fields. Same lazy-or-empty semantics as FreelanceProfileReader.
type ReferrerProfileReader interface {
	GetByOrgID(ctx context.Context, orgID uuid.UUID) (*ReferrerProfileSnapshot, error)
}

// ReferrerProfileSnapshot is the referrer-side counterpart of
// FreelanceProfileSnapshot.
type ReferrerProfileSnapshot struct {
	ProfileID          uuid.UUID
	Title              string
	About              string
	VideoURL           string
	AvailabilityStatus profile.AvailabilityStatus
	ExpertiseDomains   []string
}

// LegacyProfileReader returns the agency / enterprise legacy profile
// row. Returns nil with a nil error when the row is missing — every
// section is then computed as empty.
type LegacyProfileReader interface {
	GetByOrgID(ctx context.Context, orgID uuid.UUID) (*profile.Profile, error)
}

// SkillsCounter returns the count of skills attached to an org.
type SkillsCounter interface {
	CountByOrg(ctx context.Context, orgID uuid.UUID) (int, error)
}

// SocialLinksCounter returns the count of social links attached to an
// org for a given persona. The freelance / referrer / agency persona
// keep independent sets so the count must be persona-aware.
type SocialLinksCounter interface {
	CountByOrgPersona(ctx context.Context, orgID uuid.UUID, persona profile.SocialLinkPersona) (int, error)
}

// PortfolioCounter returns the count of portfolio items an org has
// published. Used only by the agency role mapping.
type PortfolioCounter interface {
	CountByOrganization(ctx context.Context, organizationID uuid.UUID) (int, error)
}

// FreelancePricingReader returns whether a freelance pricing row
// exists for the given freelance profile id. Implementations return
// (false, nil) when no row exists — never an error.
type FreelancePricingReader interface {
	ExistsByProfileID(ctx context.Context, profileID uuid.UUID) (bool, error)
}

// ReferrerPricingReader is the symmetric counterpart for the referrer
// persona.
type ReferrerPricingReader interface {
	ExistsByProfileID(ctx context.Context, profileID uuid.UUID) (bool, error)
}

// LegacyPricingCounter returns the count of pricing rows attached to
// the org's legacy profile (0..2). Used by the agency role mapping
// only.
type LegacyPricingCounter interface {
	CountByOrgID(ctx context.Context, orgID uuid.UUID) (int, error)
}

// Errors

// ErrNotFound is the sentinel returned by readers when the requested
// row is absent. The service does NOT propagate it — every "row
// missing" case maps to "section empty" silently. Defined here so
// adapters can wire return values to a stable sentinel.
var ErrNotFound = errors.New("profile completion: row not found")

// Service computes the completion report. Construction takes a
// dependencies bag — Go allows nil for optional readers, in which
// case the corresponding sections collapse to "empty" rather than
// panicking. This is the resilience contract the handler relies on
// when a feature (e.g. portfolio) is disabled in a given environment.
type Service struct {
	deps Deps
}

// Deps groups every reader the service needs. Each field is optional
// at construction time so the wiring layer can omit a feature without
// breaking the completion endpoint — consult the rules in Compute for
// the exact fallback per role.
type Deps struct {
	Users            UserReader
	Organizations    OrganizationReader
	Shared           SharedProfileReader
	FreelanceProfile FreelanceProfileReader
	ReferrerProfile  ReferrerProfileReader
	LegacyProfile    LegacyProfileReader
	Skills           SkillsCounter
	SocialLinks      SocialLinksCounter
	Portfolio        PortfolioCounter
	FreelancePricing FreelancePricingReader
	ReferrerPricing  ReferrerPricingReader
	LegacyPricing    LegacyPricingCounter
}

// NewService constructs the service. Required fields are Users and
// Organizations — the other readers are role-conditional and may be
// nil for environments where a given feature is disabled.
func NewService(deps Deps) (*Service, error) {
	if deps.Users == nil {
		return nil, fmt.Errorf("profile completion: Users reader is required")
	}
	if deps.Organizations == nil {
		return nil, fmt.Errorf("profile completion: Organizations reader is required")
	}
	return &Service{deps: deps}, nil
}

// Compute returns the completion report for the authenticated user's
// current organization. The persona is auto-selected from the user's
// role + organization type — provider_personal orgs default to the
// freelance persona at this endpoint. Use ComputeWithPersona to force
// the referrer persona for provider_personal orgs (the second surface
// rendered when the user toggles the referrer workspace).
func (s *Service) Compute(ctx context.Context, userID, orgID uuid.UUID) (*Report, error) {
	return s.ComputeWithPersona(ctx, userID, orgID, "")
}

// ComputeWithPersona returns the completion report scoped to the given
// persona override. When override is empty, the persona auto-selects
// from the org type. The override is honoured only when it is a
// supported persona for the org type — provider_personal orgs accept
// PersonaFreelance and PersonaReferrer; every other override falls
// back to the default persona for the org. Untrusted overrides cannot
// surface a persona that does not match the user's role.
func (s *Service) ComputeWithPersona(
	ctx context.Context,
	userID, orgID uuid.UUID,
	override Persona,
) (*Report, error) {
	u, err := s.deps.Users.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("profile completion: load user: %w", err)
	}
	org, err := s.deps.Organizations.FindByID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("profile completion: load org: %w", err)
	}

	persona := resolvePersona(org.Type, override)
	sections, err := s.buildSections(ctx, u, org, persona)
	if err != nil {
		return nil, err
	}

	filled := 0
	for _, sec := range sections {
		if sec.Filled {
			filled++
		}
	}
	total := len(sections)
	pct := 0
	if total > 0 {
		pct = (filled * 100) / total
	}

	return &Report{
		Role:           string(u.Role),
		Persona:        string(persona),
		Percent:        pct,
		TotalSections:  total,
		FilledSections: filled,
		Sections:       sections,
	}, nil
}

// personaForOrg maps an organization type to the persona whose
// completion the report describes. provider_personal currently always
// surfaces the freelance persona; agency and enterprise orgs surface
// their own personas. Unknown types fall back to the agency mapping
// so the endpoint never returns a 500 on a misconfigured row.
func personaForOrg(t organization.OrgType) Persona {
	switch t {
	case organization.OrgTypeProviderPersonal:
		return PersonaFreelance
	case organization.OrgTypeAgency:
		return PersonaAgency
	case organization.OrgTypeEnterprise:
		return PersonaEnterprise
	}
	return PersonaAgency
}

// resolvePersona picks the effective persona for the report. The
// caller-supplied override is honoured only when it is a valid
// alternative for the org type — today only provider_personal orgs
// accept the referrer override. Every unsupported override silently
// falls back to the default persona so a malicious or stale query
// param cannot surface someone else's checklist.
func resolvePersona(t organization.OrgType, override Persona) Persona {
	def := personaForOrg(t)
	if override == "" {
		return def
	}
	if t == organization.OrgTypeProviderPersonal {
		switch override {
		case PersonaFreelance, PersonaReferrer:
			return override
		}
	}
	return def
}
