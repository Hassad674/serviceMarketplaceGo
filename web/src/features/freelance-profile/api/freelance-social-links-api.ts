import { apiClient } from "@/shared/lib/api-client"

// API boundary for the freelance persona's social link set.
// Independent from the legacy /api/v1/profile/social-links agency
// surface so the two personas can evolve separately.

export type FreelanceSocialLink = {
  id: string
  persona: "freelance" | "referrer" | "agency"
  platform: string
  url: string
  created_at: string
  updated_at: string
}

export async function getMyFreelanceSocialLinks(): Promise<
  FreelanceSocialLink[]
> {
  return apiClient<FreelanceSocialLink[]>(
    "/api/v1/freelance-profile/social-links",
  )
}

export async function getPublicFreelanceSocialLinks(
  orgId: string,
): Promise<FreelanceSocialLink[]> {
  return apiClient<FreelanceSocialLink[]>(
    `/api/v1/freelance-profiles/${orgId}/social-links`,
  )
}

export async function upsertFreelanceSocialLink(
  platform: string,
  url: string,
): Promise<void> {
  return apiClient<void>("/api/v1/freelance-profile/social-links", {
    method: "PUT",
    body: { platform, url },
  })
}

export async function deleteFreelanceSocialLink(
  platform: string,
): Promise<void> {
  return apiClient<void>(
    `/api/v1/freelance-profile/social-links/${platform}`,
    { method: "DELETE" },
  )
}
