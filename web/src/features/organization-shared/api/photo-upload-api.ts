import { API_BASE_URL } from "@/shared/lib/api-client"

// UploadPhotoResponse is the tiny shape the multipart `/upload/photo`
// endpoint returns — just the canonical URL assigned by the backend
// storage adapter. The backend handler also persists the URL on the
// legacy profiles row; the org-shared mutation endpoint is invoked
// separately when the split path owner needs the URL stamped on the
// org row itself.
type UploadPhotoResponse = {
  url: string
}

// uploadOrganizationPhoto pushes a multipart file to the backend
// photo upload endpoint and resolves to the canonical URL. Raw fetch
// is used (not apiClient) because the request body is FormData, not
// JSON — apiClient would force a content-type header that corrupts
// the multipart boundary.
export async function uploadOrganizationPhoto(
  file: File,
): Promise<UploadPhotoResponse> {
  const formData = new FormData()
  formData.append("file", file)

  const res = await fetch(`${API_BASE_URL}/api/v1/upload/photo`, {
    method: "POST",
    credentials: "include",
    body: formData,
  })
  if (!res.ok) {
    const err = await res
      .json()
      .catch(() => ({ message: "upload_failed" }))
    throw new Error(err.message || "upload_failed")
  }
  return res.json()
}
