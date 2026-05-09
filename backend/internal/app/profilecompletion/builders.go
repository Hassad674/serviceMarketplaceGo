package profilecompletion

import (
	"context"
	"strings"

	"marketplace-backend/internal/domain/organization"
	"marketplace-backend/internal/domain/user"
)

// buildFreelanceSections is the checklist for provider_personal orgs
// surfacing the freelance persona. Sections are ordered from identity
// → presentation → offer so the page reads in an intuitive priority.
//
// Mapping (11 sections):
//
//	photo, title, about, expertises, skills, pricing,
//	availability, location, languages, video, social_links.
//
// billing_profile and kyc were intentionally dropped: billing info is
// captured inline by Stripe at first payment and KYC has its own
// dedicated flow, so neither belongs in the profile-completion %.
//
// availability is counted as "filled" when the freelance row has any
// status — provider_personal orgs always default to available_now on
// creation, so this section is effectively a tautology that bumps the
// baseline to ~9%. We keep it explicit so a future redesign that adds
// "I haven't decided yet" as a fourth status surfaces immediately in
// the percent.
func (s *Service) buildFreelanceSections(
	ctx context.Context,
	u *user.User,
	org *organization.Organization,
) ([]Section, error) {
	bundle, err := s.loadSnapshot(ctx, u, org)
	if err != nil {
		return nil, err
	}
	fp := bundle.Freelance
	hasFP := fp != nil

	titleFilled := hasFP && strings.TrimSpace(fp.Title) != ""
	aboutFilled := hasFP && strings.TrimSpace(fp.About) != ""
	videoFilled := hasFP && strings.TrimSpace(fp.VideoURL) != ""
	expertisesFilled := hasFP && len(fp.ExpertiseDomains) > 0
	availabilityFilled := hasFP && fp.AvailabilityStatus != ""

	out := []Section{
		section(PersonaFreelance, SectionPhoto, hasPhoto(bundle.Shared)),
		section(PersonaFreelance, SectionTitle, titleFilled),
		section(PersonaFreelance, SectionAbout, aboutFilled),
		section(PersonaFreelance, SectionExpertises, expertisesFilled),
		section(PersonaFreelance, SectionSkills, bundle.SkillCount > 0),
		section(PersonaFreelance, SectionPricing, bundle.FreelancePricing),
		section(PersonaFreelance, SectionAvailability, availabilityFilled),
		section(PersonaFreelance, SectionLocation, hasLocation(bundle.Shared)),
		section(PersonaFreelance, SectionLanguages, hasLanguages(bundle.Shared)),
		section(PersonaFreelance, SectionVideo, videoFilled),
		section(PersonaFreelance, SectionSocialLinks, bundle.SocialFreelance > 0),
	}
	return out, nil
}

// buildReferrerSections is the checklist for the apporteur persona.
// Shorter than the freelance list — no skills, no portfolio, no
// languages section (the apporteur identity reuses the freelance
// languages declared at the org level).
//
// Mapping (8 sections):
//
//	photo, title, about, expertises, pricing, availability,
//	video, social_links.
func (s *Service) buildReferrerSections(
	ctx context.Context,
	u *user.User,
	org *organization.Organization,
) ([]Section, error) {
	bundle, err := s.loadSnapshot(ctx, u, org)
	if err != nil {
		return nil, err
	}
	rp := bundle.Referrer
	has := rp != nil

	titleFilled := has && strings.TrimSpace(rp.Title) != ""
	aboutFilled := has && strings.TrimSpace(rp.About) != ""
	videoFilled := has && strings.TrimSpace(rp.VideoURL) != ""
	expertisesFilled := has && len(rp.ExpertiseDomains) > 0
	availabilityFilled := has && rp.AvailabilityStatus != ""

	out := []Section{
		section(PersonaReferrer, SectionPhoto, hasPhoto(bundle.Shared)),
		section(PersonaReferrer, SectionTitle, titleFilled),
		section(PersonaReferrer, SectionAbout, aboutFilled),
		section(PersonaReferrer, SectionExpertises, expertisesFilled),
		section(PersonaReferrer, SectionPricing, bundle.ReferrerPricing),
		section(PersonaReferrer, SectionAvailability, availabilityFilled),
		section(PersonaReferrer, SectionVideo, videoFilled),
		section(PersonaReferrer, SectionSocialLinks, bundle.SocialReferrer > 0),
	}
	return out, nil
}

// buildAgencySections is the checklist for agency orgs. Same fields
// as freelance plus a portfolio section because agencies are expected
// to ship a curated case-study list.
//
// Mapping (10 sections):
//
//	photo, title, about, skills, pricing, availability,
//	location, languages, social_links, portfolio.
//
// expertises and video are NOT mandatory for agencies — agencies are
// curated through skills + portfolio so the expertise taxonomy is
// optional, and the video upload is a recent provider-only feature.
//
// billing_profile and kyc were dropped from the checklist: billing
// info is captured inline at first payment and KYC has its own
// dedicated flow, so neither belongs in the profile-completion %.
func (s *Service) buildAgencySections(
	ctx context.Context,
	u *user.User,
	org *organization.Organization,
) ([]Section, error) {
	bundle, err := s.loadSnapshot(ctx, u, org)
	if err != nil {
		return nil, err
	}
	legacy := bundle.Legacy
	hasLegacy := legacy != nil

	titleFilled := hasLegacy && strings.TrimSpace(legacy.Title) != ""
	aboutFilled := hasLegacy && strings.TrimSpace(legacy.About) != ""
	availabilityFilled := hasLegacy && legacy.AvailabilityStatus != ""

	out := []Section{
		section(PersonaAgency, SectionPhoto, hasPhoto(bundle.Shared) ||
			(hasLegacy && strings.TrimSpace(legacy.PhotoURL) != "")),
		section(PersonaAgency, SectionTitle, titleFilled),
		section(PersonaAgency, SectionAbout, aboutFilled),
		section(PersonaAgency, SectionSkills, bundle.SkillCount > 0),
		section(PersonaAgency, SectionPricing, bundle.LegacyPricingN > 0),
		section(PersonaAgency, SectionAvailability, availabilityFilled),
		section(PersonaAgency, SectionLocation, hasLocation(bundle.Shared) ||
			(hasLegacy && strings.TrimSpace(legacy.City) != "" &&
				strings.TrimSpace(legacy.CountryCode) != "")),
		section(PersonaAgency, SectionLanguages, hasLanguages(bundle.Shared) ||
			(hasLegacy && len(legacy.LanguagesProfessional) > 0)),
		section(PersonaAgency, SectionSocialLinks, bundle.SocialAgency > 0),
		section(PersonaAgency, SectionPortfolio, bundle.PortfolioCount > 0),
	}
	return out, nil
}

// buildEnterpriseSections is the checklist for enterprise (client)
// orgs. Enterprises do not sell on the marketplace — their checklist
// focuses on the client-facing identity.
//
// Mapping (2 sections):
//
//	photo, client_about.
//
// billing_profile and kyc were dropped from the checklist: enterprises
// register their billing details inline at first payment and KYC has
// its own dedicated flow, so neither belongs in the profile-completion
// %.
func (s *Service) buildEnterpriseSections(
	ctx context.Context,
	u *user.User,
	org *organization.Organization,
) ([]Section, error) {
	bundle, err := s.loadSnapshot(ctx, u, org)
	if err != nil {
		return nil, err
	}
	legacy := bundle.Legacy
	hasLegacy := legacy != nil

	clientAboutFilled := hasLegacy && strings.TrimSpace(legacy.ClientDescription) != ""

	out := []Section{
		section(PersonaEnterprise, SectionPhoto, hasPhoto(bundle.Shared) ||
			(hasLegacy && strings.TrimSpace(legacy.PhotoURL) != "")),
		section(PersonaEnterprise, SectionClientAbout, clientAboutFilled),
	}
	return out, nil
}
