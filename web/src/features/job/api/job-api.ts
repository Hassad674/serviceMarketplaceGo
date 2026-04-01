import { apiClient } from "@/shared/lib/api-client"
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
  return apiClient<JobResponse>("/api/v1/jobs", {
    method: "POST",
    body: data,
  })
}

export function getJob(id: string): Promise<JobResponse> {
  return apiClient<JobResponse>(`/api/v1/jobs/${id}`)
}

export function listMyJobs(cursor?: string): Promise<JobWithCountsListResponse> {
  const params = cursor ? `?cursor=${encodeURIComponent(cursor)}` : ""
  return apiClient<JobWithCountsListResponse>(`/api/v1/jobs/mine${params}`)
}

export function closeJob(id: string): Promise<void> {
  return apiClient<void>(`/api/v1/jobs/${id}/close`, { method: "POST" })
}

export function reopenJob(id: string): Promise<void> {
  return apiClient<void>(`/api/v1/jobs/${id}/reopen`, { method: "POST" })
}

export function deleteJob(id: string): Promise<void> {
  return apiClient<void>(`/api/v1/jobs/${id}`, { method: "DELETE" })
}

export function markApplicationsViewed(id: string): Promise<void> {
  return apiClient<void>(`/api/v1/jobs/${id}/mark-viewed`, { method: "POST" })
}
