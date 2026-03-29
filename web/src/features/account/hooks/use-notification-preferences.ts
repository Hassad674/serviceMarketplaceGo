"use client"

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { apiClient } from "@/shared/lib/api-client"

export type NotificationPreference = {
  type: string
  in_app: boolean
  push: boolean
  email: boolean
}

const PREFS_KEY = ["account", "notification-preferences"]

export function useNotificationPreferences() {
  return useQuery({
    queryKey: PREFS_KEY,
    queryFn: async () => {
      const res = await apiClient<{ data: NotificationPreference[] }>("/api/v1/notifications/preferences")
      return res.data
    },
  })
}

export function useUpdateNotificationPreferences() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (preferences: NotificationPreference[]) => {
      await apiClient("/api/v1/notifications/preferences", {
        method: "PUT",
        body: { preferences },
      })
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: PREFS_KEY })
    },
  })
}
