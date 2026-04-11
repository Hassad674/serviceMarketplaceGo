import { apiClient, API_BASE_URL } from "@/shared/lib/api-client"

export type PortfolioMedia = {
  id: string
  media_url: string
  media_type: "image" | "video"
  thumbnail_url: string
  position: number
  created_at: string
}

// A portfolio item belongs to an organization, shared across every
// operator on the team.
export type PortfolioItem = {
  id: string
  organization_id: string
  title: string
  description: string
  link_url: string
  cover_url: string
  position: number
  media: PortfolioMedia[]
  created_at: string
  updated_at: string
}

type PortfolioListResponse = {
  data: PortfolioItem[]
  next_cursor: string
  has_more: boolean
}

type MediaPayload = {
  media_url: string
  media_type: string
  thumbnail_url?: string
  position: number
}

type CreatePortfolioPayload = {
  title: string
  description?: string
  link_url?: string
  position: number
  media?: MediaPayload[]
}

type UpdatePortfolioPayload = {
  title?: string
  description?: string
  link_url?: string
  media?: MediaPayload[]
}

export async function fetchPortfolioByOrganization(
  orgId: string,
): Promise<PortfolioListResponse> {
  return apiClient<PortfolioListResponse>(
    `/api/v1/portfolio/org/${orgId}?limit=30`,
  )
}

export async function fetchPortfolioItem(
  id: string,
): Promise<{ data: PortfolioItem }> {
  return apiClient<{ data: PortfolioItem }>(`/api/v1/portfolio/${id}`)
}

export async function createPortfolioItem(
  payload: CreatePortfolioPayload,
): Promise<{ data: PortfolioItem }> {
  return apiClient<{ data: PortfolioItem }>("/api/v1/portfolio", {
    method: "POST",
    body: payload,
  })
}

export async function updatePortfolioItem(
  id: string,
  payload: UpdatePortfolioPayload,
): Promise<{ data: PortfolioItem }> {
  return apiClient<{ data: PortfolioItem }>(`/api/v1/portfolio/${id}`, {
    method: "PUT",
    body: payload,
  })
}

export async function deletePortfolioItem(id: string): Promise<void> {
  return apiClient<void>(`/api/v1/portfolio/${id}`, { method: "DELETE" })
}

export async function reorderPortfolio(itemIds: string[]): Promise<void> {
  return apiClient<void>("/api/v1/portfolio/reorder", {
    method: "PUT",
    body: { item_ids: itemIds },
  })
}

export async function uploadPortfolioImage(
  file: File,
): Promise<{ url: string }> {
  const formData = new FormData()
  formData.append("file", file)

  const res = await fetch(`${API_BASE_URL}/api/v1/upload/portfolio-image`, {
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

export async function uploadPortfolioVideo(
  file: File,
): Promise<{ url: string }> {
  const formData = new FormData()
  formData.append("file", file)

  const res = await fetch(`${API_BASE_URL}/api/v1/upload/portfolio-video`, {
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
