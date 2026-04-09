import { adminApi } from "@/shared/lib/api-client"
import type { AdminDispute, DisputeListResponse, DisputeCountResponse, DisputeFilters } from "../types"

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
