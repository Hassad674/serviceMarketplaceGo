import { apiClient } from "@/shared/lib/api-client"

export type PublicProfileSummary = {
  user_id: string
  display_name: string
  first_name: string
  last_name: string
  role: string
  title: string
  photo_url: string
  referrer_enabled: boolean
}

export type SearchType = "freelancer" | "agency" | "referrer"

export async function searchProfiles(
  type: SearchType,
): Promise<PublicProfileSummary[]> {
  return apiClient<PublicProfileSummary[]>(
    `/api/v1/profiles/search?type=${type}`,
  )
}

export async function getPublicProfile(
  userId: string,
): Promise<PublicProfileSummary> {
  return apiClient<PublicProfileSummary>(`/api/v1/profiles/${userId}`)
}
