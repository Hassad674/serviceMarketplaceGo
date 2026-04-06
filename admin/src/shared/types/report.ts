export type AdminReport = {
  id: string
  reporter_id: string
  target_type: "message" | "user" | "job" | "job_application" | "review"
  target_id: string
  conversation_id?: string
  reason: string
  description: string
  status: "pending" | "reviewed" | "resolved" | "dismissed"
  admin_note: string
  resolved_at?: string
  resolved_by?: string
  created_at: string
  updated_at: string
}
