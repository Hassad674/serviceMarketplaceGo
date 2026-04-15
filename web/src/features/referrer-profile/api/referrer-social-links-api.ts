import { apiClient } from "@/shared/lib/api-client"

// API boundary for the referrer persona's social link set.
// Independent from the freelance persona so both can evolve on their
// own cadence without affecting the other.

export type ReferrerSocialLink = {
  id: string
  persona: "freelance" | "referrer" | "agency"
  platform: string
  url: string
  created_at: string
  updated_at: string
}

export async function getMyReferrerSocialLinks(): Promise<
  ReferrerSocialLink[]
> {
  return apiClient<ReferrerSocialLink[]>(
    "/api/v1/referrer-profile/social-links",
  )
}

export async function getPublicReferrerSocialLinks(
  orgId: string,
): Promise<ReferrerSocialLink[]> {
  return apiClient<ReferrerSocialLink[]>(
    `/api/v1/referrer-profiles/${orgId}/social-links`,
  )
}

export async function upsertReferrerSocialLink(
  platform: string,
  url: string,
): Promise<void> {
  return apiClient<void>("/api/v1/referrer-profile/social-links", {
    method: "PUT",
    body: { platform, url },
  })
}

export async function deleteReferrerSocialLink(
  platform: string,
): Promise<void> {
  return apiClient<void>(
    `/api/v1/referrer-profile/social-links/${platform}`,
    { method: "DELETE" },
  )
}
