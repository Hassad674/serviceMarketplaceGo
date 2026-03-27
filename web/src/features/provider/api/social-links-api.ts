import { apiClient } from "@/shared/lib/api-client"

export type SocialLink = {
  id: string
  platform: string
  url: string
  created_at: string
  updated_at: string
}

export async function getMySocialLinks(): Promise<SocialLink[]> {
  return apiClient<SocialLink[]>("/api/v1/profile/social-links")
}

export async function getPublicSocialLinks(userId: string): Promise<SocialLink[]> {
  return apiClient<SocialLink[]>(`/api/v1/profiles/${userId}/social-links`)
}

export async function upsertSocialLink(platform: string, url: string): Promise<void> {
  return apiClient<void>("/api/v1/profile/social-links", {
    method: "PUT",
    body: { platform, url },
  })
}

export async function deleteSocialLink(platform: string): Promise<void> {
  return apiClient<void>(`/api/v1/profile/social-links/${platform}`, {
    method: "DELETE",
  })
}
