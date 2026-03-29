"use client"

import { useMutation, useQueryClient } from "@tanstack/react-query"
import {
  markNotificationAsRead,
  markAllNotificationsAsRead,
  deleteNotification,
} from "../api/notification-api"
import { notificationsQueryKey } from "./use-notifications"
import { unreadNotifCountKey } from "./use-unread-notification-count"
import { useCurrentUserId } from "@/shared/hooks/use-current-user-id"

export function useMarkAsRead() {
  const queryClient = useQueryClient()
  const uid = useCurrentUserId()

  return useMutation({
    mutationFn: (id: string) => markNotificationAsRead(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: notificationsQueryKey(uid) })
      queryClient.invalidateQueries({ queryKey: unreadNotifCountKey(uid) })
    },
  })
}

export function useMarkAllAsRead() {
  const queryClient = useQueryClient()
  const uid = useCurrentUserId()

  return useMutation({
    mutationFn: () => markAllNotificationsAsRead(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: notificationsQueryKey(uid) })
      queryClient.setQueryData(unreadNotifCountKey(uid), { data: { count: 0 } })
    },
  })
}

export function useDeleteNotification() {
  const queryClient = useQueryClient()
  const uid = useCurrentUserId()

  return useMutation({
    mutationFn: (id: string) => deleteNotification(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: notificationsQueryKey(uid) })
      queryClient.invalidateQueries({ queryKey: unreadNotifCountKey(uid) })
    },
  })
}
