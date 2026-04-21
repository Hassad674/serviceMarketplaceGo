import { apiClient } from "@/shared/lib/api-client"
import type { Review } from "@/shared/types/review"

// Domain types for the client-profile feature. They map the backend
// contract locked in the feature spec; we deliberately duplicate the
// minimal fields we need here rather than reusing the provider
// `Profile` type so the two features stay independent (removing one
// must not break the other).

export type ClientProfileProjectHistoryProvider = {
  organization_id: string
  display_name: string
  avatar_url: string | null
}

export type ClientProjectHistoryEntry = {
  proposal_id: string
  title: string
  // Amount is in the smallest currency unit (cents) — same convention
  // as the rest of the billing surface. The component is responsible
  // for formatting via `formatCurrency`.
  amount: number
  completed_at: string
  provider: ClientProfileProjectHistoryProvider
}

export type ClientOrgType = "agency" | "enterprise"

// Shape of the public `/api/v1/clients/{orgId}` response envelope.
// Every counter defaults to `0` server-side so the UI never has to
// branch on `undefined`.
export type PublicClientProfile = {
  organization_id: string
  type: ClientOrgType
  company_name: string
  avatar_url: string | null
  client_description: string
  // Money in smallest currency unit (cents).
  total_spent: number
  review_count: number
  average_rating: number
  projects_completed_as_client: number
  project_history: ClientProjectHistoryEntry[]
  reviews: Review[]
}

export type PublicClientProfileResponse = {
  data: PublicClientProfile
}

// Private update payload for `/api/v1/profile/client`. Both fields
// are optional so callers can update only the description without
// touching the company name (and vice versa). The backend enforces
// `org_client_profile.edit` permission and returns 403 for
// `provider_personal` orgs.
export type UpdateClientProfileInput = {
  company_name?: string
  client_description?: string
}

export async function updateClientProfile(
  input: UpdateClientProfileInput,
): Promise<void> {
  await apiClient<void>("/api/v1/profile/client", {
    method: "PUT",
    body: input,
  })
}

export async function fetchPublicClientProfile(
  orgId: string,
): Promise<PublicClientProfile> {
  const envelope = await apiClient<PublicClientProfileResponse>(
    `/api/v1/clients/${orgId}`,
  )
  return envelope.data
}

// MyClientProfile projects the subset of `/api/v1/profile` that the
// private client-profile page needs. The backend extends the existing
// ProfileResponse with five client-side fields (client_description,
// total_spent, client_review_count, client_avg_rating,
// projects_completed_as_client). Every counter defaults to `0` so
// the form / header never branch on undefined.
export type MyClientProfile = {
  organization_id: string
  company_name: string
  avatar_url: string | null
  client_description: string
  total_spent: number
  client_review_count: number
  client_avg_rating: number
  projects_completed_as_client: number
}

// Raw backend payload. Kept loose on purpose — the /profile endpoint
// carries a lot of provider-side fields we don't consume here, and
// re-declaring them would create brittle cross-feature coupling.
type RawProfileResponse = {
  organization_id: string
  title?: string
  photo_url?: string | null
  client_description?: string
  total_spent?: number
  client_review_count?: number
  client_avg_rating?: number
  projects_completed_as_client?: number
}

export async function fetchMyClientProfile(): Promise<MyClientProfile> {
  const raw = await apiClient<RawProfileResponse>("/api/v1/profile")
  return {
    organization_id: raw.organization_id,
    company_name: raw.title ?? "",
    avatar_url: raw.photo_url ?? null,
    client_description: raw.client_description ?? "",
    total_spent: raw.total_spent ?? 0,
    client_review_count: raw.client_review_count ?? 0,
    client_avg_rating: raw.client_avg_rating ?? 0,
    projects_completed_as_client: raw.projects_completed_as_client ?? 0,
  }
}
