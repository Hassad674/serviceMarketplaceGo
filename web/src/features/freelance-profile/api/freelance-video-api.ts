import { API_BASE_URL } from "@/shared/lib/api-client"

// Legacy video-upload endpoints are shared with the agency path and
// still write the URL onto the legacy profiles row. Until those
// endpoints are migrated to the freelance aggregate this thin wrapper
// is the freelance feature's boundary: it exposes a stable async
// function and the hook layer reconciles the cache.

type UploadVideoResponse = { url: string }

export async function uploadFreelanceVideo(
  file: File,
): Promise<UploadVideoResponse> {
  const formData = new FormData()
  formData.append("file", file)
  const res = await fetch(`${API_BASE_URL}/api/v1/upload/video`, {
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
  const res = await fetch(`${API_BASE_URL}/api/v1/upload/video`, {
    method: "DELETE",
    credentials: "include",
  })
  if (!res.ok) {
    throw new Error("delete_failed")
  }
}
