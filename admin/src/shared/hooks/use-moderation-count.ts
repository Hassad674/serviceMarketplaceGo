import { useQuery } from "@tanstack/react-query"
import { getModerationCount } from "@/shared/api/moderation-count-api"

export function useModerationCount() {
  return useQuery({
    queryKey: ["admin", "moderation-count"],
    queryFn: getModerationCount,
    refetchInterval: 60 * 1000,
    staleTime: 30 * 1000,
  })
}
