import { API_BASE_URL } from "@/shared/lib/api-client"

const API_URL = API_BASE_URL

type UploadResponse = {
  url: string
}

async function uploadFile(
  endpoint: string,
  file: File,
): Promise<UploadResponse> {
  const formData = new FormData()
  formData.append("file", file)

  const res = await fetch(`${API_URL}${endpoint}`, {
    method: "POST",
    credentials: "include",
    body: formData,
  })

  if (!res.ok) {
    const err = await res.json().catch(() => ({ message: "Upload failed" }))
    throw new Error(err.message || "Upload failed")
  }

  return res.json()
}

export async function uploadPhoto(
  file: File,
): Promise<UploadResponse> {
  return uploadFile("/api/v1/upload/photo", file)
}

export async function uploadVideo(
  file: File,
): Promise<UploadResponse> {
  return uploadFile("/api/v1/upload/video", file)
}

export async function uploadReferrerVideo(
  file: File,
): Promise<UploadResponse> {
  return uploadFile("/api/v1/upload/referrer-video", file)
}

export async function deleteVideo(): Promise<void> {
  const res = await fetch(`${API_URL}/api/v1/upload/video`, {
    method: "DELETE",
    credentials: "include",
  })
  if (!res.ok) throw new Error("Failed to delete video")
}

export async function deleteReferrerVideo(): Promise<void> {
  const res = await fetch(`${API_URL}/api/v1/upload/referrer-video`, {
    method: "DELETE",
    credentials: "include",
  })
  if (!res.ok) throw new Error("Failed to delete referrer video")
}
