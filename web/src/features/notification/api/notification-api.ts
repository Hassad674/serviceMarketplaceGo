import { apiClient } from "@/shared/lib/api-client"
import type {
  NotificationListResponse,
  NotificationPreference,
  UnreadCountResponse,
} from "../types"

export function listNotifications(cursor?: string): Promise<NotificationListResponse> {
  const params = new URLSearchParams()
  if (cursor) params.set("cursor", cursor)
  params.set("limit", "20")
  return apiClient<NotificationListResponse>(`/api/v1/notifications?${params.toString()}`)
}

export function getUnreadNotificationCount(): Promise<UnreadCountResponse> {
  return apiClient<UnreadCountResponse>("/api/v1/notifications/unread-count")
}

export function markNotificationAsRead(id: string): Promise<void> {
  return apiClient<void>(`/api/v1/notifications/${id}/read`, { method: "POST" })
}

export function markAllNotificationsAsRead(): Promise<void> {
  return apiClient<void>("/api/v1/notifications/read-all", { method: "POST" })
}

export function deleteNotification(id: string): Promise<void> {
  return apiClient<void>(`/api/v1/notifications/${id}`, { method: "DELETE" })
}

export function getNotificationPreferences(): Promise<{ data: NotificationPreference[] }> {
  return apiClient<{ data: NotificationPreference[] }>("/api/v1/notifications/preferences")
}

export function updateNotificationPreferences(preferences: NotificationPreference[]): Promise<void> {
  return apiClient<void>("/api/v1/notifications/preferences", {
    method: "PUT",
    body: { preferences },
  })
}

export function registerDeviceToken(token: string, platform: string): Promise<void> {
  return apiClient<void>("/api/v1/notifications/device-token", {
    method: "POST",
    body: { token, platform },
  })
}
