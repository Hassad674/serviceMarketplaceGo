import { apiClient } from "@/shared/lib/api-client"

export type AvailabilityStatus =
  | "available_now"
  | "available_soon"
  | "not_available"

export type WorkMode = "remote" | "on_site" | "hybrid"

// ReferrerPricingType enumerates the pricing strategies a referrer
// can declare. Commission-only, matching the domain model — the
// freelance persona uses a different type union (daily/hourly/etc).
export type ReferrerPricingType = "commission_pct" | "commission_flat"

// ReferrerPricing is the persona-specific pricing row attached to a
// referrer profile. commission_pct uses "pct" as currency and stores
// basis points in min/max. commission_flat uses an ISO 4217 code and
// stores cents.
export type ReferrerPricing = {
  type: ReferrerPricingType
  min_amount: number
  max_amount: number | null
  currency: string
  note: string
  negotiable: boolean
}

// ReferrerProfile mirrors FreelanceProfile structurally minus the
// skills decoration: skills describe what a person does themselves,
// which is orthogonal to the referrer "I bring deals" positioning.
export type ReferrerProfile = {
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

  pricing: ReferrerPricing | null

  created_at: string
  updated_at: string
}

export async function getMyReferrerProfile(): Promise<ReferrerProfile> {
  return apiClient<ReferrerProfile>("/api/v1/referrer-profile")
}

export async function getPublicReferrerProfile(
  orgId: string,
): Promise<ReferrerProfile> {
  return apiClient<ReferrerProfile>(`/api/v1/referrer-profiles/${orgId}`)
}

export type UpdateReferrerProfileInput = {
  title: string
  about: string
  video_url: string
}

export async function updateReferrerProfile(
  input: UpdateReferrerProfileInput,
): Promise<ReferrerProfile> {
  return apiClient<ReferrerProfile>("/api/v1/referrer-profile", {
    method: "PUT",
    body: input,
  })
}

export async function updateReferrerAvailability(
  status: AvailabilityStatus,
): Promise<ReferrerProfile> {
  return apiClient<ReferrerProfile>("/api/v1/referrer-profile/availability", {
    method: "PUT",
    body: { availability_status: status },
  })
}

export async function updateReferrerExpertise(
  domains: string[],
): Promise<ReferrerProfile> {
  return apiClient<ReferrerProfile>("/api/v1/referrer-profile/expertise", {
    method: "PUT",
    body: { domains },
  })
}

// --- Pricing --------------------------------------------------------------

type PricingEnvelope<T> = { data: T }

export async function getReferrerPricing(): Promise<ReferrerPricing | null> {
  const wrapped = await apiClient<PricingEnvelope<ReferrerPricing | null>>(
    "/api/v1/referrer-profile/pricing",
  )
  return wrapped.data
}

export type UpsertReferrerPricingInput = ReferrerPricing

export async function upsertReferrerPricing(
  input: UpsertReferrerPricingInput,
): Promise<ReferrerPricing> {
  const wrapped = await apiClient<PricingEnvelope<ReferrerPricing>>(
    "/api/v1/referrer-profile/pricing",
    { method: "PUT", body: input },
  )
  return wrapped.data
}

export async function deleteReferrerPricing(): Promise<void> {
  await apiClient<void>("/api/v1/referrer-profile/pricing", {
    method: "DELETE",
  })
}
