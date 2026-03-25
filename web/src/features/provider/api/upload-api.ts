const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080"

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
