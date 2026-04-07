import { adminApi } from "@/shared/lib/api-client"

type ModerationCountResponse = {
  count: number
}

export function getModerationCount(): Promise<ModerationCountResponse> {
  return adminApi<ModerationCountResponse>("/api/v1/admin/moderation/count")
}
