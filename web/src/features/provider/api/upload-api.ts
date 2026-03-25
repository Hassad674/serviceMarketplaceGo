const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8083"

type UploadResponse = {
  url: string
}

async function uploadFile(
  endpoint: string,
  token: string,
  file: File,
): Promise<UploadResponse> {
  const formData = new FormData()
  formData.append("file", file)

  const res = await fetch(`${API_URL}${endpoint}`, {
    method: "POST",
    headers: { Authorization: `Bearer ${token}` },
    body: formData,
  })

  if (!res.ok) {
    const err = await res.json().catch(() => ({ message: "Upload failed" }))
    throw new Error(err.message || "Upload failed")
  }

  return res.json()
}

export async function uploadPhoto(
  token: string,
  file: File,
): Promise<UploadResponse> {
  return uploadFile("/api/v1/upload/photo", token, file)
}

export async function uploadVideo(
  token: string,
  file: File,
): Promise<UploadResponse> {
  return uploadFile("/api/v1/upload/video", token, file)
}

export async function uploadReferrerVideo(
  token: string,
  file: File,
): Promise<UploadResponse> {
  return uploadFile("/api/v1/upload/referrer-video", token, file)
}

export async function deleteVideo(token: string): Promise<void> {
  const res = await fetch(`${API_URL}/api/v1/upload/video`, {
    method: "DELETE",
    headers: { Authorization: `Bearer ${token}` },
  })
  if (!res.ok) throw new Error("Failed to delete video")
}

export async function deleteReferrerVideo(token: string): Promise<void> {
  const res = await fetch(`${API_URL}/api/v1/upload/referrer-video`, {
    method: "DELETE",
    headers: { Authorization: `Bearer ${token}` },
  })
  if (!res.ok) throw new Error("Failed to delete referrer video")
}
