import { apiClient } from "@/shared/lib/api-client"

// Domain types for the client-profile feature. They map the backend
// contract locked in the feature spec; we deliberately duplicate the
// minimal fields we need here rather than reusing the provider
// `Profile` type so the two features stay independent (removing one
// must not break the other).

export type ClientOrgType = "agency" | "enterprise"

// Shape of the public `/api/v1/clients/{orgId}` response envelope.
// Every counter defaults to `0` server-side so the UI never has to
// branch on `undefined`. Note: neither `project_history[]` nor a
// top-level `reviews[]` are surfaced here — the unified "project
// history" section reads the shared `/api/v1/profiles/{orgId}/
// project-history` endpoint (same source of truth as the provider
// profile), and each entry carries its associated provider→client
// review embedded inline.
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

// MyClientProfile is the exact shape the private client-profile page
// needs. Since `/api/v1/clients/{orgId}` (the public read endpoint)
// already returns every field at the right name and nesting —
// company_name from organizations.name, stats flat at the top level,
// avatar_url, client_description — we reuse it for the owner's own
// view instead of trying to fish values out of `/api/v1/profile`,
// whose `title` is the provider's job title (not the company name)
// and whose client stats are nested under a `client` section that is
// easy to mis-map. Single source of truth for the data, single cache
// key to invalidate after a write.
export type MyClientProfile = PublicClientProfile

// Fetches the authenticated owner's client profile via the public
// endpoint. Still 404s for `provider_personal` orgs — matching v1's
// rule. The private page renders its own NotFoundState for that case
// before this call ever runs.
export async function fetchMyClientProfile(
  orgId: string,
): Promise<MyClientProfile> {
  return fetchPublicClientProfile(orgId)
}
