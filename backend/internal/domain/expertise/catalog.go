// Package expertise owns the closed catalog of domain specializations
// a provider organization can declare on its public profile.
//
// The catalog is intentionally frozen: adding or removing a key is a
// code change that touches the domain layer, the frontend translation
// files, and (usually) a release note. There is no "custom" free-form
// domain — the whole point is to give discovery a small, curated
// vocabulary so filters and search stay meaningful.
//
// Keys live in Go and in the database as plain TEXT. Human-readable
// labels live in the frontend i18n files (web and mobile) — the
// backend never formats labels. This keeps the domain layer pure
// (no localization, no presentation concerns) and lets product
// translate the catalog without a backend deploy.
//
// Design note — zero cross-domain imports: MaxForOrgType and
// IsFeatureEnabled take an OrgType string rather than the typed
// organization.OrgType alias. Keeping this package free of other
// domain imports preserves the hexagonal invariant that domain
// modules are independent and removable (the only shared dependency
// for the expertise feature is the organizations TABLE, referenced
// by the migration — never the Go package).
package expertise

// OrgType mirrors the small subset of organization.OrgType values this
// package needs to reason about. Using a plain string alias here keeps
// the expertise domain module independent of the organization domain
// module at the Go package level, while still matching the exact wire
// values used by both the database and the API.
type OrgType = string

const (
	OrgTypeAgency           OrgType = "agency"
	OrgTypeProviderPersonal OrgType = "provider_personal"
	OrgTypeEnterprise       OrgType = "enterprise"
)

// Key is a strongly-typed alias for a catalog entry. Using a named
// type (instead of a bare string) lets the type system catch places
// that try to pass arbitrary strings where a catalog key is expected.
type Key string

const (
	KeyDevelopment        Key = "development"
	KeyDataAIML           Key = "data_ai_ml"
	KeyDesignUIUX         Key = "design_ui_ux"
	KeyDesign3DAnimation  Key = "design_3d_animation"
	KeyVideoMotion        Key = "video_motion"
	KeyPhotoAudiovisual   Key = "photo_audiovisual"
	KeyMarketingGrowth    Key = "marketing_growth"
	KeyWritingTranslation Key = "writing_translation"
	KeyBusinessDevSales   Key = "business_dev_sales"
	KeyConsultingStrategy Key = "consulting_strategy"
	KeyProductUXResearch  Key = "product_ux_research"
	KeyOpsAdminSupport    Key = "ops_admin_support"
	KeyLegal              Key = "legal"
	KeyFinanceAccounting  Key = "finance_accounting"
	KeyHRRecruitment      Key = "hr_recruitment"
)

// All is the ordered, canonical list of catalog keys as plain strings.
// The order reflects the discovery UI taxonomy (technical first, then
// creative, then business) and is also the order surfaced by any
// catalog-exposing endpoint if one is added later. Callers that need
// the typed version can map over this slice.
//
// It is a var (not a const) only because Go lacks constant slices;
// treat it as immutable — never mutate it from client code, never
// append to it at runtime.
var All = []string{
	string(KeyDevelopment),
	string(KeyDataAIML),
	string(KeyDesignUIUX),
	string(KeyDesign3DAnimation),
	string(KeyVideoMotion),
	string(KeyPhotoAudiovisual),
	string(KeyMarketingGrowth),
	string(KeyWritingTranslation),
	string(KeyBusinessDevSales),
	string(KeyConsultingStrategy),
	string(KeyProductUXResearch),
	string(KeyOpsAdminSupport),
	string(KeyLegal),
	string(KeyFinanceAccounting),
	string(KeyHRRecruitment),
}

// validKeys is the O(1) lookup set backing IsValidKey. Built once at
// package init from All so the two stay in sync automatically — there
// is no second source of truth to forget to update.
var validKeys = func() map[string]struct{} {
	set := make(map[string]struct{}, len(All))
	for _, k := range All {
		set[k] = struct{}{}
	}
	return set
}()

// IsValidKey reports whether key is part of the frozen catalog. Empty
// strings, unknown keys, and whitespace-padded strings all return
// false — the caller is expected to have trimmed inputs already
// (handler-layer concern).
func IsValidKey(key string) bool {
	_, ok := validKeys[key]
	return ok
}

// MaxForOrgType returns the number of expertise domains an organization
// of the given type may declare. A return value of 0 signals the feature
// is forbidden for that org type (enterprise — it is a client-side org,
// not a provider). Unknown org types also return 0 as a safe default:
// the service layer compares against this value, so 0 means "reject
// every non-empty payload".
//
// Limits:
//   - agency            : 8 (teams have more breadth of service)
//   - provider_personal : 5 (solo freelancers should pick their focus)
//   - enterprise        : 0 (feature disabled — clients, not providers)
func MaxForOrgType(orgType OrgType) int {
	switch orgType {
	case OrgTypeAgency:
		return 8
	case OrgTypeProviderPersonal:
		return 5
	default:
		// Enterprise and any unrecognized future org type fall through
		// to zero so the service layer rejects every non-empty payload.
		return 0
	}
}

// IsFeatureEnabled reports whether the expertise feature is available
// for the given organization type. Returns false for enterprise (by
// design) and for unknown types (safe default).
func IsFeatureEnabled(orgType OrgType) bool {
	return MaxForOrgType(orgType) > 0
}
