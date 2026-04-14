import { apiClient } from "@/shared/lib/api-client"

// WorkMode mirrors the backend "work_mode" enum. Duplicated here (vs
// re-exporting from shared) so the org-shared feature stays the
// canonical writer of the shared columns — consumers import types
// from this feature's public surface.
export type WorkMode = "remote" | "on_site" | "hybrid"

// OrganizationSharedProfile is the set of columns lifted from the
// profiles table onto the organizations row in the split-profile
// refactor: photo, location, languages. Both the freelance and the
// referrer persona read these through their own profile endpoint (via
// JOIN) but writes go through this feature.
export type OrganizationSharedProfile = {
  photo_url: string
  city: string
  country_code: string
  latitude: number | null
  longitude: number | null
  work_mode: WorkMode[]
  travel_radius_km: number | null
  languages_professional: string[]
  languages_conversational: string[]
}

// The shared-profile endpoints wrap their payload in `{ data: ... }`.
// Kept as a private envelope so callers get the naked object back and
// don't have to care about the transport shape.
type Envelope<T> = { data: T }

export async function getOrganizationShared(): Promise<OrganizationSharedProfile> {
  const wrapped = await apiClient<Envelope<OrganizationSharedProfile>>(
    "/api/v1/organization/shared",
  )
  return wrapped.data
}

export type UpdateOrganizationLocationInput = {
  city: string
  country_code: string
  latitude: number | null
  longitude: number | null
  work_mode: WorkMode[]
  travel_radius_km: number | null
}

export async function updateOrganizationLocation(
  input: UpdateOrganizationLocationInput,
): Promise<OrganizationSharedProfile> {
  const wrapped = await apiClient<Envelope<OrganizationSharedProfile>>(
    "/api/v1/organization/location",
    { method: "PUT", body: input },
  )
  return wrapped.data
}

export type UpdateOrganizationLanguagesInput = {
  professional: string[]
  conversational: string[]
}

export async function updateOrganizationLanguages(
  input: UpdateOrganizationLanguagesInput,
): Promise<OrganizationSharedProfile> {
  const wrapped = await apiClient<Envelope<OrganizationSharedProfile>>(
    "/api/v1/organization/languages",
    { method: "PUT", body: input },
  )
  return wrapped.data
}

export type UpdateOrganizationPhotoInput = {
  photo_url: string
}

export async function updateOrganizationPhoto(
  input: UpdateOrganizationPhotoInput,
): Promise<OrganizationSharedProfile> {
  const wrapped = await apiClient<Envelope<OrganizationSharedProfile>>(
    "/api/v1/organization/photo",
    { method: "PUT", body: input },
  )
  return wrapped.data
}
