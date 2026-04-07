import { adminApi } from "@/shared/lib/api-client"

type AdminNotificationsResponse = {
  data: Record<string, number>
}

export function getAdminNotifications(): Promise<AdminNotificationsResponse> {
  return adminApi<AdminNotificationsResponse>("/api/v1/admin/notifications")
}

export function resetAdminNotification(category: string): Promise<void> {
  return adminApi<void>(`/api/v1/admin/notifications/${category}/reset`, {
    method: "POST",
  })
}
