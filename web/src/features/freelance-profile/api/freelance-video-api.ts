import { API_BASE_URL } from "@/shared/lib/api-client"

// Per-persona video-upload boundary for the freelance aggregate.
// Hits the dedicated endpoints under /api/v1/freelance-profile/video
// which write directly to freelance_profiles.video_url — the legacy
// /api/v1/upload/video path still serves agency orgs but cannot
// persist for provider_personal users since migration 104.

type UploadVideoResponse = { video_url: string }

export async function uploadFreelanceVideo(
  file: File,
): Promise<UploadVideoResponse> {
  const formData = new FormData()
  formData.append("file", file)
  const res = await fetch(`${API_BASE_URL}/api/v1/freelance-profile/video`, {
    method: "POST",
    credentials: "include",
    body: formData,
  })
  if (!res.ok) {
    const err = await res.json().catch(() => ({ message: "upload_failed" }))
    throw new Error(err.message || "upload_failed")
  }
  return res.json()
}

export async function deleteFreelanceVideo(): Promise<void> {
  const res = await fetch(`${API_BASE_URL}/api/v1/freelance-profile/video`, {
    method: "DELETE",
    credentials: "include",
  })
  if (!res.ok) {
    throw new Error("delete_failed")
  }
}
