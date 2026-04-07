import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import {
  getAdminNotifications,
  resetAdminNotification,
} from "@/shared/api/admin-notifications-api"

const NOTIF_QUERY_KEY = ["admin", "notifications"] as const

export function useAdminNotifications() {
  return useQuery({
    queryKey: NOTIF_QUERY_KEY,
    queryFn: getAdminNotifications,
    refetchInterval: 30 * 1000,
    staleTime: 10 * 1000,
  })
}

export function useResetNotification() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (category: string) => resetAdminNotification(category),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: [...NOTIF_QUERY_KEY] })
    },
  })
}

export function useInvalidateNotifications() {
  const queryClient = useQueryClient()
  return () => {
    queryClient.invalidateQueries({ queryKey: [...NOTIF_QUERY_KEY] })
  }
}
