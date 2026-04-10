"use client"

import { useQuery } from "@tanstack/react-query"
import { fetchProjectHistory } from "../api/project-history-api"

export function useProjectHistory(userId: string | undefined) {
  return useQuery({
    queryKey: ["profiles", userId, "project-history"],
    queryFn: () => fetchProjectHistory(userId!),
    staleTime: 2 * 60 * 1000,
    enabled: !!userId,
  })
}
