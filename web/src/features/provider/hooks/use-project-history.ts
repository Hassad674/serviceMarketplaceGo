"use client"

import { useQuery } from "@tanstack/react-query"
import { fetchProjectHistory } from "../api/project-history-api"

export function useProjectHistory(orgId: string | undefined) {
  return useQuery({
    queryKey: ["profiles", "org", orgId, "project-history"],
    queryFn: () => fetchProjectHistory(orgId!),
    staleTime: 2 * 60 * 1000,
    enabled: !!orgId,
  })
}
