import { API_BASE_URL } from "@/shared/lib/api-client"

// Legacy referrer-video endpoints — same rationale as the freelance
// video wrapper. Kept inside the feature so the hook layer can treat
// them like any other persona-owned mutation.

type UploadVideoResponse = { url: string }

export async function uploadReferrerVideo(
  file: File,
): Promise<UploadVideoResponse> {
  const formData = new FormData()
  formData.append("file", file)
  const res = await fetch(
    `${API_BASE_URL}/api/v1/upload/referrer-video`,
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
    `${API_BASE_URL}/api/v1/upload/referrer-video`,
    {
      method: "DELETE",
      credentials: "include",
    },
  )
  if (!res.ok) {
    throw new Error("delete_failed")
  }
}
