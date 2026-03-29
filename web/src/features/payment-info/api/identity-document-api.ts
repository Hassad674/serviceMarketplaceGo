import { apiClient, API_BASE_URL } from "@/shared/lib/api-client"

export type IdentityDocumentResponse = {
  id: string
  user_id: string
  category: string
  document_type: string
  side: string
  file_url: string
  status: "pending" | "verified" | "rejected"
  rejection_reason?: string
  created_at: string
  updated_at: string
}

export async function listIdentityDocuments(): Promise<IdentityDocumentResponse[]> {
  return apiClient<IdentityDocumentResponse[]>("/api/v1/identity-documents")
}

export async function uploadIdentityDocument(
  file: File,
  category: string,
  documentType: string,
  side: string,
): Promise<IdentityDocumentResponse> {
  const formData = new FormData()
  formData.append("file", file)
  formData.append("category", category)
  formData.append("document_type", documentType)
  formData.append("side", side)

  const res = await fetch(`${API_BASE_URL}/api/v1/identity-documents/upload`, {
    method: "POST",
    credentials: "include",
    body: formData,
  })
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: { message: "Upload failed" } }))
    throw new Error(err.error?.message || "Upload failed")
  }
  const data = await res.json()
  return data.data ?? data
}

export async function deleteIdentityDocument(id: string): Promise<void> {
  return apiClient<void>(`/api/v1/identity-documents/${id}`, { method: "DELETE" })
}
