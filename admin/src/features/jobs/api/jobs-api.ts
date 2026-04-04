import { adminApi } from "@/shared/lib/api-client"
import type {
  AdminJobListResponse,
  AdminJobDetailResponse,
  AdminJobApplicationListResponse,
  JobFilters,
  ApplicationFilters,
} from "../types"

export function listAdminJobs(filters: JobFilters): Promise<AdminJobListResponse> {
  const params = new URLSearchParams()
  if (filters.status) params.set("status", filters.status)
  if (filters.search) params.set("search", filters.search)
  if (filters.sort) params.set("sort", filters.sort)
  if (filters.page > 0) params.set("page", String(filters.page))
  params.set("limit", "20")
  const qs = params.toString()
  return adminApi<AdminJobListResponse>(`/api/v1/admin/jobs${qs ? `?${qs}` : ""}`)
}

export function getAdminJob(id: string): Promise<AdminJobDetailResponse> {
  return adminApi<AdminJobDetailResponse>(`/api/v1/admin/jobs/${id}`)
}

export function deleteAdminJob(id: string): Promise<void> {
  return adminApi(`/api/v1/admin/jobs/${id}`, { method: "DELETE" })
}

export function listAdminJobApplications(filters: ApplicationFilters): Promise<AdminJobApplicationListResponse> {
  const params = new URLSearchParams()
  if (filters.job_id) params.set("job_id", filters.job_id)
  if (filters.search) params.set("search", filters.search)
  if (filters.sort) params.set("sort", filters.sort)
  if (filters.page > 0) params.set("page", String(filters.page))
  params.set("limit", "20")
  const qs = params.toString()
  return adminApi<AdminJobApplicationListResponse>(`/api/v1/admin/job-applications${qs ? `?${qs}` : ""}`)
}

export function deleteAdminJobApplication(id: string): Promise<void> {
  return adminApi(`/api/v1/admin/job-applications/${id}`, { method: "DELETE" })
}
