import { apiClient } from "@/shared/lib/api-client"

export async function createReport(data: {
  target_type: string
  target_id: string
  conversation_id: string
  reason: string
  description: string
}) {
  return apiClient("/api/v1/reports", { method: "POST", body: data })
}
