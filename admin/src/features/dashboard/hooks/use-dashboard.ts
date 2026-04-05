import { useQuery } from "@tanstack/react-query"
import { getDashboardStats } from "../api/dashboard-api"

export function useDashboardStats() {
  return useQuery({
    queryKey: ["admin", "dashboard", "stats"],
    queryFn: getDashboardStats,
    staleTime: 60 * 1000,
  })
}
