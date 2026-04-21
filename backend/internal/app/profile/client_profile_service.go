package profileapp

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/organization"
	"marketplace-backend/internal/domain/profile"
	"marketplace-backend/internal/port/repository"
)

// ClientProfileService owns the write flow for the organization's
// client-facing profile facet (company_name + client_description).
// It lives in the profile application package because the client
// profile and the provider-facing profile share the same underlying
// aggregate (the Profile row) — co-locating keeps the aggregate's
// use cases together.
//
// Dependencies: the profile repository (persistence of the client
// description) and the organization repository (type gating + name
// rename). Both are interfaces from port/, so the service is fully
// testable with mocks.
type ClientProfileService struct {
	profiles      repository.ProfileRepository
	organizations repository.OrganizationRepository
}

// NewClientProfileService wires a new client-profile service. It
// takes the two repositories directly (no options struct) so the
// dependency graph at wiring time stays flat — matching the
// ExpertiseService pattern.
func NewClientProfileService(
	profileRepo repository.ProfileRepository,
	orgRepo repository.OrganizationRepository,
) *ClientProfileService {
	return &ClientProfileService{
		profiles:      profileRepo,
		organizations: orgRepo,
	}
}

// UpdateClientProfileInput is the write payload. Both fields are
// optional: a caller may touch only the company name or only the
// description and leave the other block untouched. CompanyName is a
// pointer so the zero-value case ("do not touch") is distinguishable
// from the explicit empty-string case — empty is rejected as a
// rename because the org's display name is mandatory.
type UpdateClientProfileInput struct {
	// CompanyName, when non-nil, renames the organization. The value
	// is trimmed; an empty trimmed string yields
	// organization.ErrNameRequired.
	CompanyName *string
	// ClientDescription, when non-nil, replaces the client-facing
	// description. An empty string is a valid explicit value (clears
	// the description). Length is capped by
	// profile.MaxClientDescriptionLength.
	ClientDescription *string
}

// UpdateClientProfile persists the client-facing facet of the org's
// profile. The flow is deliberately linear so an unsupervised agent
// reading the code can predict the validation order:
//
//  1. Resolve the org to discover its type. Missing orgs short-circuit
//     with a wrapped ErrOrgNotFound.
//  2. Reject provider_personal orgs — v1 exposes the client profile
//     only to agency and enterprise. Extending to other types later
//     means flipping one condition in isClientProfileEnabled.
//  3. Validate the inputs:
//     - CompanyName (if set) must trim to a non-empty string.
//     - ClientDescription (if set) must respect the max length.
//  4. Write each side. Rename first (it touches a different table);
//     client_description second. If the description write fails after
//     a successful rename, the caller still sees a 500 and the DB
//     ends up in a half-applied state — the two writes are across
//     different aggregates and cannot share a transaction without
//     leaking infrastructure concerns into the service. V1 accepts
//     that trade-off; a future phase can introduce an outbox if
//     partial-failure visibility matters.
//
// Returns the refreshed Profile entity with the new state so the
// handler can hand it to the response DTO without a second fetch.
func (s *ClientProfileService) UpdateClientProfile(
	ctx context.Context,
	orgID uuid.UUID,
	input UpdateClientProfileInput,
) (*profile.Profile, error) {
	org, err := s.organizations.FindByID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("update client profile: resolve org: %w", err)
	}
	if !isClientProfileEnabled(org.Type) {
		return nil, profile.ErrForbiddenOrgType
	}

	if err := validateClientProfileInput(input); err != nil {
		return nil, err
	}

	if input.CompanyName != nil {
		name := strings.TrimSpace(*input.CompanyName)
		if err := org.Rename(name); err != nil {
			return nil, fmt.Errorf("update client profile: rename: %w", err)
		}
		if err := s.organizations.Update(ctx, org); err != nil {
			return nil, fmt.Errorf("update client profile: persist rename: %w", err)
		}
	}

	if input.ClientDescription != nil {
		// Moderation hook (see flagged follow-up in the feature report):
		// no text-moderation service is currently wired for profile free-
		// form text. When one lands, fire it here before the repository
		// write so rejected copy never touches the DB.
		if err := s.profiles.UpdateClientDescription(ctx, orgID, *input.ClientDescription); err != nil {
			return nil, fmt.Errorf("update client profile: persist description: %w", err)
		}
	}

	refreshed, err := s.profiles.GetByOrganizationID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("update client profile: reload: %w", err)
	}
	return refreshed, nil
}

// isClientProfileEnabled is the single source of truth for which org
// types can edit a client profile. V1 exposes it to agency and
// enterprise only; provider_personal receives ErrForbiddenOrgType.
// Extending the feature to provider_personal later is a one-line
// change here plus the corresponding public-read relaxation in the
// clientprofile package.
func isClientProfileEnabled(t organization.OrgType) bool {
	switch t {
	case organization.OrgTypeAgency, organization.OrgTypeEnterprise:
		return true
	}
	return false
}

// validateClientProfileInput enforces the pure-input rules (length,
// trimmed non-empty) that do not require a database round-trip.
// Returned sentinel errors map to 400 at the handler layer.
func validateClientProfileInput(input UpdateClientProfileInput) error {
	if input.ClientDescription != nil {
		if len(*input.ClientDescription) > profile.MaxClientDescriptionLength {
			return profile.ErrClientDescriptionTooLong
		}
	}
	if input.CompanyName != nil {
		if strings.TrimSpace(*input.CompanyName) == "" {
			return organization.ErrNameRequired
		}
	}
	return nil
}
