import { apiClient } from "@/shared/lib/api-client"

// ProfileSkill matches the backend ProfileResponse.skills entries: the
// normalized `skill_text` (canonical lowercase lookup key) paired with
// the user-facing `display_text`. The backend guarantees this array is
// non-null — empty profiles return `[]`, never omitted.
export type ProfileSkill = {
  skill_text: string
  display_text: string
}

// Tier 1 taxonomy — kept as string literal unions so the compiler
// rejects any typo on the backend contract without a runtime check.
export type WorkMode = "remote" | "on_site" | "hybrid"
export type AvailabilityStatus =
  | "available_now"
  | "available_soon"
  | "not_available"
export type PricingKind = "direct" | "referral"
export type PricingType =
  | "daily"
  | "hourly"
  | "project_from"
  | "project_range"
  | "commission_pct"
  | "commission_flat"

// A single pricing row returned by the backend. `min_amount` and
// `max_amount` are stored in the smallest unit of the row's currency:
//  - centimes for EUR/USD/GBP/CAD/AUD (currency is an ISO 4217 code)
//  - basis points (1/100 of a percent) when currency === "pct", used
//    exclusively by commission_pct.
// `max_amount` is null when the row does not have an upper bound.
// `negotiable` is the explicit yes/no flag surfaced as a "négociable"
// badge on the profile card — distinct from the free-text `note`.
export type Pricing = {
  kind: PricingKind
  type: PricingType
  min_amount: number
  max_amount: number | null
  currency: string
  note: string
  negotiable: boolean
}

// Profile is the organization's shared marketplace identity: the same
// photo, video, about text, and title that every team member edits
// collaboratively. Since the team refactor the anchor is the org id,
// not an individual user id.
export type Profile = {
  organization_id: string
  title: string
  photo_url: string
  presentation_video_url: string
  referrer_video_url: string
  about: string
  referrer_about: string
  // Ordered list of expertise domain keys (see expertise.ts catalog).
  // Order is significant — it is the display order on profile pages.
  // Absent for orgs that do not have expertise (legacy clients should
  // treat `undefined` as an empty list).
  expertise_domains?: string[]
  // Skills attached to the organization, in insertion order. Backend
  // always returns an array (possibly empty) — never null. Older clients
  // that predate the skills endpoint should treat `undefined` as empty.
  skills?: ProfileSkill[]
  // --- Tier 1 (Location) ---
  city?: string
  country_code?: string
  latitude?: number | null
  longitude?: number | null
  work_mode?: WorkMode[]
  travel_radius_km?: number | null
  // --- Tier 1 (Languages) ---
  languages_professional?: string[]
  languages_conversational?: string[]
  // --- Tier 1 (Availability) ---
  availability_status?: AvailabilityStatus
  referrer_availability_status?: AvailabilityStatus | null
  // --- Tier 1 (Pricing) ---
  pricing?: Pricing[]
  created_at: string
  updated_at: string
}

export async function getMyProfile(): Promise<Profile> {
  return apiClient<Profile>("/api/v1/profile")
}

export async function updateProfile(
  data: Partial<Profile>,
): Promise<Profile> {
  return apiClient<Profile>("/api/v1/profile", {
    method: "PUT",
    body: data,
  })
}

// --- Tier 1 mutations ----------------------------------------------------

export type UpdateLocationInput = {
  city: string
  country_code: string
  work_mode: WorkMode[]
  travel_radius_km: number | null
}

export async function updateLocation(
  input: UpdateLocationInput,
): Promise<void> {
  await apiClient<void>("/api/v1/profile/location", {
    method: "PUT",
    body: input,
  })
}

export type UpdateLanguagesInput = {
  professional: string[]
  conversational: string[]
}

export async function updateLanguages(
  input: UpdateLanguagesInput,
): Promise<void> {
  await apiClient<void>("/api/v1/profile/languages", {
    method: "PUT",
    body: input,
  })
}

export type UpdateAvailabilityInput = {
  availability_status: AvailabilityStatus
  referrer_availability_status?: AvailabilityStatus | null
}

export async function updateAvailability(
  input: UpdateAvailabilityInput,
): Promise<void> {
  await apiClient<void>("/api/v1/profile/availability", {
    method: "PUT",
    body: input,
  })
}

export async function getPricing(): Promise<Pricing[]> {
  return apiClient<Pricing[]>("/api/v1/profile/pricing")
}

export async function upsertPricing(pricing: Pricing): Promise<void> {
  await apiClient<void>("/api/v1/profile/pricing", {
    method: "PUT",
    body: pricing,
  })
}

export async function deletePricing(kind: PricingKind): Promise<void> {
  await apiClient<void>(`/api/v1/profile/pricing/${kind}`, {
    method: "DELETE",
  })
}
