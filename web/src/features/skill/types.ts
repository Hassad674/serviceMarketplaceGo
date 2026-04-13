// Local DTOs mirroring the backend skill feature responses. These are
// hand-written rather than generated so the skill module stays
// independent of the shared OpenAPI types file and can be removed
// without touching generated artifacts.
//
// Keep field names in snake_case to match the backend JSON directly —
// the API client does not transform keys.

export type SkillResponse = {
  skill_text: string
  display_text: string
  expertise_keys: string[]
  is_curated: boolean
  usage_count: number
}

export type ProfileSkillResponse = {
  skill_text: string
  display_text: string
  position: number
}

export type CatalogResponse = {
  skills: SkillResponse[]
  total: number
}
