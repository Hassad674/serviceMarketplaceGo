import { apiClient } from "@/shared/lib/api-client"
import type { Get } from "@/shared/lib/api-paths"
import type { Review } from "@/shared/types/review"

export type ProjectHistoryEntry = {
  proposal_id: string
  title: string // empty when the client opted out of sharing the title
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
  orgId: string,
  cursor?: string,
): Promise<ProjectHistoryResponse> {
  const qs = cursor ? `?cursor=${encodeURIComponent(cursor)}` : ""
  return apiClient<Get<"/api/v1/profiles/{orgId}/project-history"> & ProjectHistoryResponse>(
    `/api/v1/profiles/${orgId}/project-history${qs}`,
  )
}
