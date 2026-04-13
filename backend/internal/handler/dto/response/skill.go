package response

import (
	domainskill "marketplace-backend/internal/domain/skill"
)

// SkillResponse is the public DTO for one catalog entry. Exposes every
// field the browse panels and autocomplete dropdowns need without
// leaking timestamps (the frontend does not use created_at/updated_at
// for skill UI).
type SkillResponse struct {
	SkillText     string   `json:"skill_text"`
	DisplayText   string   `json:"display_text"`
	ExpertiseKeys []string `json:"expertise_keys"`
	IsCurated     bool     `json:"is_curated"`
	UsageCount    int      `json:"usage_count"`
}

// NewSkillResponse maps a domain CatalogEntry to its DTO.
func NewSkillResponse(e *domainskill.CatalogEntry) SkillResponse {
	return SkillResponse{
		SkillText:     e.SkillText,
		DisplayText:   e.DisplayText,
		ExpertiseKeys: safeSkillStrings(e.ExpertiseKeys),
		IsCurated:     e.IsCurated,
		UsageCount:    e.UsageCount,
	}
}

// NewSkillsListResponse maps a slice of domain entries. Returns a
// non-nil slice so the JSON envelope always contains [] instead of
// null when the list is empty.
func NewSkillsListResponse(entries []*domainskill.CatalogEntry) []SkillResponse {
	out := make([]SkillResponse, 0, len(entries))
	for _, e := range entries {
		out = append(out, NewSkillResponse(e))
	}
	return out
}

// ProfileSkillResponse is the DTO for one skill attached to a profile.
// DisplayText mirrors the catalog's canonical casing so the client can
// render the skill chip without a second query.
type ProfileSkillResponse struct {
	SkillText   string `json:"skill_text"`
	DisplayText string `json:"display_text"`
	Position    int    `json:"position"`
}

// safeSkillStrings ensures a non-nil slice is returned — JSON `null`
// is awkward for frontend clients expecting an empty array.
func safeSkillStrings(in []string) []string {
	if in == nil {
		return []string{}
	}
	return in
}
