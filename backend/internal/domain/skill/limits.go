package skill

// OrgType is a plain string alias matching the wire values used
// throughout the marketplace backend ("agency", "provider_personal",
// "enterprise"). Using a string alias rather than importing the
// organization or expertise packages preserves the hexagonal invariant
// that skill is a fully independent domain module: the only shared
// dependency for the skills feature is the organizations TABLE,
// referenced by the migration — never the Go package.
type OrgType = string

const (
	OrgTypeAgency           OrgType = "agency"
	OrgTypeProviderPersonal OrgType = "provider_personal"
	OrgTypeEnterprise       OrgType = "enterprise"
)

// MaxSkillsForOrgType returns the maximum number of skills an
// organization of the given type may declare on its public profile.
//
// Limits:
//
//   - agency            : 40 (teams legitimately cover broader stacks)
//   - provider_personal : 25 (solo practitioners should curate)
//   - enterprise        :  0 (feature disabled — clients, not providers)
//
// Returns 0 for enterprise and for any unknown / future org type as a
// safe default: the service layer compares against this value, so 0
// means "reject every non-empty payload". Adding a new provider-like
// org type in the future is therefore an opt-in code change rather
// than an accidental opening of the feature.
func MaxSkillsForOrgType(orgType OrgType) int {
	switch orgType {
	case OrgTypeAgency:
		return 40
	case OrgTypeProviderPersonal:
		return 25
	default:
		// Enterprise and any unrecognized future org type fall through
		// to zero so the service layer rejects every non-empty payload.
		return 0
	}
}

// IsSkillsFeatureEnabled reports whether the skills feature is
// available for the given organization type. Returns false for
// enterprise (by design) and for unknown types (safe default).
func IsSkillsFeatureEnabled(orgType OrgType) bool {
	return MaxSkillsForOrgType(orgType) > 0
}
