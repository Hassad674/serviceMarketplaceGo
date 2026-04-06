import { adminApi } from "@/shared/lib/api-client"
import type { AdminReport } from "@/shared/types/report"

type ConversationReportsResponse = {
  data: AdminReport[]
}

type UserReportsResponse = {
  reports_against: AdminReport[]
  reports_filed: AdminReport[]
}

type ResolvePayload = {
  status: "resolved" | "dismissed"
  admin_note: string
}

export function listConversationReports(conversationId: string): Promise<ConversationReportsResponse> {
  return adminApi<ConversationReportsResponse>(`/api/v1/admin/conversations/${conversationId}/reports`)
}

export function listUserReports(userId: string): Promise<UserReportsResponse> {
  return adminApi<UserReportsResponse>(`/api/v1/admin/users/${userId}/reports`)
}

type JobReportsResponse = {
  data: AdminReport[]
}

export function listJobReports(jobId: string): Promise<JobReportsResponse> {
  return adminApi<JobReportsResponse>(`/api/v1/admin/jobs/${jobId}/reports`)
}

type ReviewReportsResponse = {
  data: AdminReport[]
}

export function listReviewReports(reviewId: string): Promise<ReviewReportsResponse> {
  return adminApi<ReviewReportsResponse>(`/api/v1/admin/reviews/${reviewId}/reports`)
}

export function resolveReport(reportId: string, payload: ResolvePayload): Promise<void> {
  return adminApi<void>(`/api/v1/admin/reports/${reportId}/resolve`, {
    method: "POST",
    body: payload,
  })
}
