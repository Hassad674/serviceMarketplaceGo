import { API_BASE_URL } from "@/shared/lib/api-client"

// Per-persona video-upload boundary for the referrer aggregate.
// Hits the dedicated endpoints under /api/v1/referrer-profile/video
// which write directly to referrer_profiles.video_url. The legacy
// /api/v1/upload/referrer-video path is unreachable for
// provider_personal orgs since migration 104 removed their legacy
// profiles row.

type UploadVideoResponse = { video_url: string }

export async function uploadReferrerVideo(
  file: File,
): Promise<UploadVideoResponse> {
  const formData = new FormData()
  formData.append("file", file)
  const res = await fetch(
    `${API_BASE_URL}/api/v1/referrer-profile/video`,
    {
      method: "POST",
      credentials: "include",
      body: formData,
    },
  )
  if (!res.ok) {
    const err = await res.json().catch(() => ({ message: "upload_failed" }))
    throw new Error(err.message || "upload_failed")
  }
  return res.json()
}

export async function deleteReferrerVideo(): Promise<void> {
  const res = await fetch(
    `${API_BASE_URL}/api/v1/referrer-profile/video`,
    {
      method: "DELETE",
      credentials: "include",
    },
  )
  if (!res.ok) {
    throw new Error("delete_failed")
  }
}
