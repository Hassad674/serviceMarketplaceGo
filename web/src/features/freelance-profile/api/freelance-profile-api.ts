import { apiClient } from "@/shared/lib/api-client"

// AvailabilityStatus mirrors the enum accepted by the freelance
// profile backend. Matches shared/components/ui/availability-pill's
// union so consumers never have to map between them.
export type AvailabilityStatus =
  | "available_now"
  | "available_soon"
  | "not_available"

export type WorkMode = "remote" | "on_site" | "hybrid"

// FreelancePricingType enumerates the disjoint set of pricing strategies
// supported by a freelance persona: no commission kinds — those live on
// the referrer persona.
export type FreelancePricingType =
  | "daily"
  | "hourly"
  | "project_from"
  | "project_range"

// FreelancePricing is the persona-specific pricing row attached to a
// freelance profile. MinAmount / MaxAmount are stored in the smallest
// unit of the row's currency (centimes for EUR/USD/GBP, etc.). MaxAmount
// is null when the row does not have an upper bound (daily, hourly,
// project_from) and required when the type is a range.
export type FreelancePricing = {
  type: FreelancePricingType
  min_amount: number
  max_amount: number | null
  currency: string
  note: string
  negotiable: boolean
}

// ProfileSkill matches backend response.ProfileSkillSummary — the
// normalized skill_text (canonical lowercase lookup key) plus the
// user-facing display_text preserved from the original submission.
export type ProfileSkill = {
  skill_text: string
  display_text: string
}

// FreelanceProfile is the full response shape for GET
// /api/v1/freelance-profile and GET /api/v1/freelance-profiles/{orgID}.
// Fields in the "shared" section are JOINed from the organizations
// row — writes for those fields go through the organization-shared
// feature, reads always come back denormalized here.
export type FreelanceProfile = {
  id: string
  organization_id: string
  title: string
  about: string
  video_url: string
  availability_status: AvailabilityStatus
  expertise_domains: string[]

  // Shared (JOINed from organizations)
  photo_url: string
  city: string
  country_code: string
  latitude: number | null
  longitude: number | null
  work_mode: WorkMode[]
  travel_radius_km: number | null
  languages_professional: string[]
  languages_conversational: string[]

  // Decorations
  skills: ProfileSkill[]
  pricing: FreelancePricing | null

  created_at: string
  updated_at: string
}

export async function getMyFreelanceProfile(): Promise<FreelanceProfile> {
  return apiClient<FreelanceProfile>("/api/v1/freelance-profile")
}

export async function getPublicFreelanceProfile(
  orgId: string,
): Promise<FreelanceProfile> {
  return apiClient<FreelanceProfile>(`/api/v1/freelance-profiles/${orgId}`)
}

export type UpdateFreelanceProfileInput = {
  title: string
  about: string
  video_url: string
}

export async function updateFreelanceProfile(
  input: UpdateFreelanceProfileInput,
): Promise<FreelanceProfile> {
  return apiClient<FreelanceProfile>("/api/v1/freelance-profile", {
    method: "PUT",
    body: input,
  })
}

export async function updateFreelanceAvailability(
  status: AvailabilityStatus,
): Promise<FreelanceProfile> {
  return apiClient<FreelanceProfile>("/api/v1/freelance-profile/availability", {
    method: "PUT",
    body: { availability_status: status },
  })
}

export async function updateFreelanceExpertise(
  domains: string[],
): Promise<FreelanceProfile> {
  return apiClient<FreelanceProfile>("/api/v1/freelance-profile/expertise", {
    method: "PUT",
    body: { domains },
  })
}

// --- Pricing --------------------------------------------------------------

type PricingEnvelope<T> = { data: T }

export async function getFreelancePricing(): Promise<FreelancePricing | null> {
  const wrapped = await apiClient<PricingEnvelope<FreelancePricing | null>>(
    "/api/v1/freelance-profile/pricing",
  )
  return wrapped.data
}

export type UpsertFreelancePricingInput = FreelancePricing

export async function upsertFreelancePricing(
  input: UpsertFreelancePricingInput,
): Promise<FreelancePricing> {
  const wrapped = await apiClient<PricingEnvelope<FreelancePricing>>(
    "/api/v1/freelance-profile/pricing",
    { method: "PUT", body: input },
  )
  return wrapped.data
}

export async function deleteFreelancePricing(): Promise<void> {
  await apiClient<void>("/api/v1/freelance-profile/pricing", {
    method: "DELETE",
  })
}
