import { apiClient } from "@/shared/lib/api-client"
import type { Get, Post, Void } from "@/shared/lib/api-paths"
import type {
  JobListResponse,
  JobApplicationResponse,
  ApplicationListResponse,
  MyApplicationListResponse,
  OpenJobListFilters,
} from "../types"

export function listOpenJobs(filters?: OpenJobListFilters, cursor?: string): Promise<JobListResponse> {
  const params = new URLSearchParams()
  if (cursor) params.set("cursor", cursor)
  if (filters?.search) params.set("search", filters.search)
  if (filters?.applicant_type) params.set("applicant_type", filters.applicant_type)
  if (filters?.budget_type) params.set("budget_type", filters.budget_type)
  if (filters?.min_budget !== undefined) params.set("min_budget", String(filters.min_budget))
  if (filters?.max_budget !== undefined) params.set("max_budget", String(filters.max_budget))
  if (filters?.skills?.length) params.set("skills", filters.skills.join(","))
  const query = params.toString()
  return apiClient<Get<"/api/v1/jobs/open"> & JobListResponse>(`/api/v1/jobs/open${query ? `?${query}` : ""}`)
}

export function applyToJob(jobId: string, body: { message: string; video_url?: string }): Promise<JobApplicationResponse> {
  return apiClient<Post<"/api/v1/jobs/{id}/apply"> & JobApplicationResponse>(`/api/v1/jobs/${jobId}/apply`, {
    method: "POST",
    body,
  })
}

export function withdrawApplication(applicationId: string): Promise<void> {
  return apiClient<Void<"/api/v1/jobs/applications/{applicationId}">>(`/api/v1/jobs/applications/${applicationId}`, {
    method: "DELETE",
  })
}

export function listJobApplications(jobId: string, cursor?: string): Promise<ApplicationListResponse> {
  const params = cursor ? `?cursor=${encodeURIComponent(cursor)}` : ""
  return apiClient<Get<"/api/v1/jobs/{id}/applications"> & ApplicationListResponse>(`/api/v1/jobs/${jobId}/applications${params}`)
}

export function listMyApplications(cursor?: string): Promise<MyApplicationListResponse> {
  const params = cursor ? `?cursor=${encodeURIComponent(cursor)}` : ""
  return apiClient<Get<"/api/v1/jobs/applications/mine"> & MyApplicationListResponse>(`/api/v1/jobs/applications/mine${params}`)
}

export function contactApplicant(jobId: string, applicantId: string): Promise<{ conversation_id: string }> {
  return apiClient<Post<"/api/v1/jobs/{id}/applications/{applicantId}/contact"> & { conversation_id: string }>(`/api/v1/jobs/${jobId}/applications/${applicantId}/contact`, {
    method: "POST",
  })
}

export function hasApplied(jobId: string): Promise<{ has_applied: boolean }> {
  return apiClient<Get<"/api/v1/jobs/{id}/has-applied"> & { has_applied: boolean }>(`/api/v1/jobs/${jobId}/has-applied`)
}
