import { adminApi } from "@/shared/lib/api-client"
import type {
  AdminDispute,
  AskAIResponse,
  DisputeListResponse,
  DisputeCountResponse,
  DisputeFilters,
} from "../types"

export function listDisputes(filters: DisputeFilters): Promise<DisputeListResponse> {
  const params = new URLSearchParams()
  if (filters.status) params.set("status", filters.status)
  if (filters.cursor) params.set("cursor", filters.cursor)
  params.set("limit", "20")
  const qs = params.toString()
  return adminApi<DisputeListResponse>(`/api/v1/admin/disputes${qs ? `?${qs}` : ""}`)
}

export function getDispute(id: string): Promise<AdminDispute> {
  return adminApi<AdminDispute>(`/api/v1/admin/disputes/${id}`)
}

export function resolveDispute(id: string, data: {
  amount_client: number
  amount_provider: number
  note: string
}): Promise<{ status: string }> {
  return adminApi<{ status: string }>(`/api/v1/admin/disputes/${id}/resolve`, {
    method: "POST",
    body: data,
  })
}

export function countDisputes(): Promise<DisputeCountResponse> {
  return adminApi<DisputeCountResponse>("/api/v1/admin/disputes/count")
}

// Dev/testing only — instantly escalates a dispute, bypassing the 7-day
// inactivity window. Returns 404 in production.
export function forceEscalateDispute(id: string): Promise<{ status: string }> {
  return adminApi<{ status: string }>(`/api/v1/admin/disputes/${id}/force-escalate`, {
    method: "POST",
  })
}

// askAIDispute sends a chat question about a dispute. The chat history
// is loaded server-side from the database, so the request body only
// carries the new question. The mutation invalidates the dispute query
// on success and the panel re-renders with the updated history.
export function askAIDispute(
  id: string,
  question: string,
): Promise<AskAIResponse> {
  return adminApi<AskAIResponse>(`/api/v1/admin/disputes/${id}/ai-chat`, {
    method: "POST",
    body: { question },
  })
}

// increaseAIBudget grants the dispute extra AI tokens via the
// "Augmenter le budget" button. Each call adds the default increment.
export function increaseAIBudget(
  id: string,
): Promise<{ status: string; bonus_increment: number }> {
  return adminApi<{ status: string; bonus_increment: number }>(
    `/api/v1/admin/disputes/${id}/ai-budget`,
    { method: "POST" },
  )
}
