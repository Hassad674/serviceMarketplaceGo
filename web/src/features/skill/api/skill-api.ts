import { apiClient } from "@/shared/lib/api-client"
import type { Get, Post, Put } from "@/shared/lib/api-paths"
import type {
  CatalogResponse,
  ProfileSkillResponse,
  SkillResponse,
} from "../types"

// GET /api/v1/profile/skills — returns the current operator's
// ordered skill list. `position` is significant: the frontend must
// preserve ordering when rendering chips.
export async function fetchProfileSkills(): Promise<ProfileSkillResponse[]> {
  return apiClient<Get<"/api/v1/profile/skills"> & ProfileSkillResponse[]>("/api/v1/profile/skills")
}

// PUT /api/v1/profile/skills — replaces the full list. The order of
// `skillTexts` becomes the canonical order. Returns `{ status: "ok" }`
// on success, which we intentionally drop because callers only need
// to know the request succeeded.
export async function updateProfileSkills(
  skillTexts: string[],
): Promise<void> {
  await apiClient<Put<"/api/v1/profile/skills"> & { status: string }>("/api/v1/profile/skills", {
    method: "PUT",
    body: { skill_texts: skillTexts },
  })
}

// GET /api/v1/skills/catalog — browse by expertise domain. Public
// endpoint (no auth required), but we still route it through
// apiClient for consistent error handling.
export async function fetchCatalog(
  expertiseKey: string,
  limit = 50,
): Promise<CatalogResponse> {
  const params = new URLSearchParams({
    expertise: expertiseKey,
    limit: String(limit),
  })
  return apiClient<Get<"/api/v1/skills/catalog"> & CatalogResponse>(`/api/v1/skills/catalog?${params.toString()}`)
}

// GET /api/v1/skills/autocomplete — prefix search across the whole
// catalog. Returns an unwrapped array of skills (no `total` field).
export async function searchSkillsAutocomplete(
  query: string,
  limit = 20,
): Promise<SkillResponse[]> {
  const params = new URLSearchParams({ q: query, limit: String(limit) })
  return apiClient<Get<"/api/v1/skills/autocomplete"> & SkillResponse[]>(
    `/api/v1/skills/autocomplete?${params.toString()}`,
  )
}

// POST /api/v1/skills — creates a user-defined ("uncurated") skill
// from a free-form display text. The backend normalises the text
// into a canonical `skill_text` key, so callers should use the
// returned object rather than the input they sent.
export async function createUserSkill(
  displayText: string,
): Promise<SkillResponse> {
  return apiClient<Post<"/api/v1/skills"> & SkillResponse>("/api/v1/skills", {
    method: "POST",
    body: { display_text: displayText },
  })
}
