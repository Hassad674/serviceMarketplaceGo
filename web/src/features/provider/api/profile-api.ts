import { apiClient } from "@/shared/lib/api-client"

export type Profile = {
  user_id: string
  title: string
  photo_url: string
  presentation_video_url: string
  referrer_video_url: string
  about: string
  referrer_about: string
  created_at: string
  updated_at: string
}

export async function getMyProfile(): Promise<Profile> {
  return apiClient<Profile>("/api/v1/profile")
}

export async function updateProfile(
  data: Partial<Profile>,
): Promise<Profile> {
  return apiClient<Profile>("/api/v1/profile", {
    method: "PUT",
    body: data,
  })
}
