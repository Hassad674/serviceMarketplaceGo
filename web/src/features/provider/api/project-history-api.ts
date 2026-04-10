import { apiClient } from "@/shared/lib/api-client"
import type { Review } from "@/shared/types/review"

export type ProjectHistoryEntry = {
  proposal_id: string
  amount: number // in cents
  currency: string // "EUR"
  completed_at: string
  review: Review | null
}

export type ProjectHistoryResponse = {
  data: ProjectHistoryEntry[]
  next_cursor: string
  has_more: boolean
}

export async function fetchProjectHistory(
  userId: string,
  cursor?: string,
): Promise<ProjectHistoryResponse> {
  const qs = cursor ? `?cursor=${encodeURIComponent(cursor)}` : ""
  return apiClient<ProjectHistoryResponse>(
    `/api/v1/profiles/${userId}/project-history${qs}`,
  )
}
