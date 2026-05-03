import { apiClient } from "@/shared/lib/api-client"

import type { Post } from "@/shared/lib/api-paths"
/**
 * Shared reporting API. The reporting feature's UX (P9) is consumed
 * cross-feature (messaging, job), so the API call lives in `shared/`.
 */
export async function createReport(data: {
  target_type: string
  target_id: string
  conversation_id: string
  reason: string
  description: string
}) {
  return apiClient<Post<"/api/v1/reports">>("/api/v1/reports", { method: "POST", body: data })
}
