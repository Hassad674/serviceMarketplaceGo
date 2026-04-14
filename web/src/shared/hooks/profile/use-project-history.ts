"use client"

import { useQuery } from "@tanstack/react-query"
import { apiClient } from "@/shared/lib/api-client"
import type { Review } from "@/shared/types/review"

// ProjectHistoryEntry is the public, cursor-paginated view of a
// completed project attached to an organization. The row carries the
// associated review when the client submitted one — otherwise `review`
// is null and the UI surfaces an "awaiting review" placeholder.
export type ProjectHistoryEntry = {
  proposal_id: string
  title: string
  amount: number
  currency: string
  completed_at: string
  review: Review | null
}

export type ProjectHistoryResponse = {
  data: ProjectHistoryEntry[]
  next_cursor: string
  has_more: boolean
}

async function fetchProjectHistory(
  orgId: string,
): Promise<ProjectHistoryResponse> {
  return apiClient<ProjectHistoryResponse>(
    `/api/v1/profiles/${orgId}/project-history`,
  )
}

// useProjectHistory reads the first page of an organization's
// completed-project history. Shared across all profile personas
// because the backend keys this endpoint on organization_id, not on
// persona type — the freelance profile and the referrer profile of
// the same org share the same history endpoint.
export function useProjectHistory(orgId: string | undefined) {
  return useQuery({
    queryKey: ["profiles", "org", orgId, "project-history"],
    queryFn: () => fetchProjectHistory(orgId!),
    staleTime: 2 * 60 * 1000,
    enabled: Boolean(orgId),
  })
}
