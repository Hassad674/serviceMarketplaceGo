import { apiClient } from "@/shared/lib/api-client"

import type { Get, Void } from "@/shared/lib/api-paths"
export type SocialLink = {
  id: string
  platform: string
  url: string
  created_at: string
  updated_at: string
}

export async function getMySocialLinks(): Promise<SocialLink[]> {
  return apiClient<Get<"/api/v1/profile/social-links"> & SocialLink[]>("/api/v1/profile/social-links")
}

export async function getPublicSocialLinks(orgId: string): Promise<SocialLink[]> {
  return apiClient<Get<"/api/v1/profiles/{orgId}/social-links"> & SocialLink[]>(`/api/v1/profiles/${orgId}/social-links`)
}

export async function upsertSocialLink(platform: string, url: string): Promise<void> {
  return apiClient<Void<"/api/v1/profile/social-links">>("/api/v1/profile/social-links", {
    method: "PUT",
    body: { platform, url },
  })
}

export async function deleteSocialLink(platform: string): Promise<void> {
  return apiClient<Void<"/api/v1/profile/social-links/{platform}">>(`/api/v1/profile/social-links/${platform}`, {
    method: "DELETE",
  })
}
