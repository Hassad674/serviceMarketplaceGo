import { apiClient } from "@/shared/lib/api-client"

import type { Post } from "@/shared/lib/api-paths"
export type UploadedFile = {
  filename: string
  url: string
  size: number
  mime_type: string
}

type PresignedURLResponse = {
  upload_url: string
  file_key: string
  public_url: string
}

/**
 * Upload a single file using a presigned URL.
 * Returns the public URL and metadata to send to the backend.
 */
export async function uploadFile(file: File): Promise<UploadedFile> {
  const presigned = await apiClient<Post<"/api/v1/messaging/upload-url"> & PresignedURLResponse>(
    "/api/v1/messaging/upload-url",
    {
      method: "POST",
      body: { filename: file.name, content_type: file.type },
    },
  )

  const uploadRes = await fetch(presigned.upload_url, {
    method: "PUT",
    body: file,
    headers: { "Content-Type": file.type },
  })

  if (!uploadRes.ok) {
    throw new Error(`Upload failed: ${uploadRes.status}`)
  }

  return {
    filename: file.name,
    url: presigned.public_url,
    size: file.size,
    mime_type: file.type,
  }
}

/**
 * Upload multiple files in parallel.
 */
export async function uploadFiles(files: File[]): Promise<UploadedFile[]> {
  return Promise.all(files.map(uploadFile))
}
