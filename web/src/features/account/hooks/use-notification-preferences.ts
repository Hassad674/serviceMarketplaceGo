"use client"

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { apiClient } from "@/shared/lib/api-client"

export type NotificationPreference = {
  type: string
  in_app: boolean
  push: boolean
  email: boolean
}

export type NotificationPreferencesResponse = {
  data: NotificationPreference[]
  email_notifications_enabled: boolean
}

const PREFS_KEY = ["account", "notification-preferences"]

export function useNotificationPreferences() {
  return useQuery({
    queryKey: PREFS_KEY,
    queryFn: async () => {
      const res = await apiClient<NotificationPreferencesResponse>("/api/v1/notifications/preferences")
      return res
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

export function useBulkEmailPreferences() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (enabled: boolean) => {
      await apiClient("/api/v1/notifications/preferences/bulk-email", {
        method: "PATCH",
        body: { enabled },
      })
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: PREFS_KEY })
    },
  })
}
