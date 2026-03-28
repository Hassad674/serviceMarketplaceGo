"use client"

import { useMutation, useQueryClient } from "@tanstack/react-query"
import {
  markNotificationAsRead,
  markAllNotificationsAsRead,
  deleteNotification,
} from "../api/notification-api"
import { NOTIFICATIONS_QUERY_KEY } from "./use-notifications"
import { UNREAD_NOTIF_COUNT_KEY } from "./use-unread-notification-count"

export function useMarkAsRead() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => markNotificationAsRead(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: NOTIFICATIONS_QUERY_KEY })
      queryClient.invalidateQueries({ queryKey: UNREAD_NOTIF_COUNT_KEY })
    },
  })
}

export function useMarkAllAsRead() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: () => markAllNotificationsAsRead(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: NOTIFICATIONS_QUERY_KEY })
      queryClient.setQueryData(UNREAD_NOTIF_COUNT_KEY, { data: { count: 0 } })
    },
  })
}

export function useDeleteNotification() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteNotification(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: NOTIFICATIONS_QUERY_KEY })
      queryClient.invalidateQueries({ queryKey: UNREAD_NOTIF_COUNT_KEY })
    },
  })
}
