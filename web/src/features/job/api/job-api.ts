import { apiClient } from "@/shared/lib/api-client"
import type { Get, Post, Put, Void } from "@/shared/lib/api-paths"
import type { JobResponse, JobWithCountsListResponse } from "../types"

export type CreateJobData = {
  title: string
  description: string
  skills: string[]
  applicant_type: string
  budget_type: string
  min_budget: number
  max_budget: number
  payment_frequency?: string
  duration_weeks?: number
  is_indefinite: boolean
  description_type: string
  video_url?: string
}

export function createJob(data: CreateJobData): Promise<JobResponse> {
  return apiClient<Post<"/api/v1/jobs"> & JobResponse>("/api/v1/jobs", {
    method: "POST",
    body: data,
  })
}

export function updateJob(id: string, data: CreateJobData): Promise<JobResponse> {
  return apiClient<Put<"/api/v1/jobs/{id}"> & JobResponse>(`/api/v1/jobs/${id}`, {
    method: "PUT",
    body: data,
  })
}

export function getJob(id: string): Promise<JobResponse> {
  return apiClient<Get<"/api/v1/jobs/{id}"> & JobResponse>(`/api/v1/jobs/${id}`)
}

export function listMyJobs(cursor?: string): Promise<JobWithCountsListResponse> {
  const params = cursor ? `?cursor=${encodeURIComponent(cursor)}` : ""
  return apiClient<Get<"/api/v1/jobs/mine"> & JobWithCountsListResponse>(`/api/v1/jobs/mine${params}`)
}

export function closeJob(id: string): Promise<void> {
  return apiClient<Void<"/api/v1/jobs/{id}/close">>(`/api/v1/jobs/${id}/close`, { method: "POST" })
}

export function reopenJob(id: string): Promise<void> {
  return apiClient<Void<"/api/v1/jobs/{id}/reopen">>(`/api/v1/jobs/${id}/reopen`, { method: "POST" })
}

export function deleteJob(id: string): Promise<void> {
  return apiClient<Void<"/api/v1/jobs/{id}">>(`/api/v1/jobs/${id}`, { method: "DELETE" })
}

export function markApplicationsViewed(id: string): Promise<void> {
  return apiClient<Void<"/api/v1/jobs/{id}/mark-viewed">>(`/api/v1/jobs/${id}/mark-viewed`, { method: "POST" })
}

export function getCredits(): Promise<{ credits: number }> {
  return apiClient<Get<"/api/v1/jobs/credits"> & { credits: number }>("/api/v1/jobs/credits")
}
