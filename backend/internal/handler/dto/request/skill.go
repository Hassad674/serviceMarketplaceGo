package request

// PutProfileSkillsRequest is the payload for PUT /api/v1/profile/skills.
// SkillTexts is the ordered list of normalized skill_text values (the
// client sends the exact keys returned by the catalog endpoints,
// already normalized). Position in the array IS the display order:
// index 0 is the skill shown first on the profile card.
type PutProfileSkillsRequest struct {
	SkillTexts []string `json:"skill_texts"`
}

// CreateSkillRequest is the payload for POST /api/v1/skills. Used by
// the free-form "Create X" path in the autocomplete dropdown. The
// server auto-inherits ExpertiseKeys from the current user's declared
// expertise domains — the client does NOT pass them. A later iteration
// may let the client hint at specific keys; for V1 the list is empty.
type CreateSkillRequest struct {
	DisplayText string `json:"display_text"`
}
