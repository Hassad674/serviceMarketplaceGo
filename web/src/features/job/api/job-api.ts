import { apiClient } from "@/shared/lib/api-client"
import type { JobResponse, JobListResponse } from "../types"

export type CreateJobData = {
  title: string
  description: string
  skills: string[]
  applicant_type: string
  budget_type: string
  min_budget: number
  max_budget: number
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

export function listMyJobs(cursor?: string): Promise<JobListResponse> {
  const params = cursor ? `?cursor=${encodeURIComponent(cursor)}` : ""
  return apiClient<JobListResponse>(`/api/v1/jobs/mine${params}`)
}

export function closeJob(id: string): Promise<void> {
  return apiClient<void>(`/api/v1/jobs/${id}/close`, { method: "POST" })
}
