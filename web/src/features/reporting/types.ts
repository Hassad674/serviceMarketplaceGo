export type TargetType = "message" | "user" | "job" | "application"

export type ReportReason =
  | "harassment"
  | "fraud"
  | "fraud_or_scam"
  | "spam"
  | "inappropriate_content"
  | "fake_profile"
  | "unprofessional_behavior"
  | "misleading_description"
  | "other"

export type Report = {
  id: string
  target_type: TargetType
  target_id: string
  reason: ReportReason
  description: string
  status: string
  created_at: string
}

export const MESSAGE_REASONS: ReportReason[] = [
  "harassment",
  "fraud",
  "spam",
  "inappropriate_content",
  "other",
]

export const USER_REASONS: ReportReason[] = [
  "harassment",
  "fraud",
  "spam",
  "fake_profile",
  "unprofessional_behavior",
  "other",
]

export const JOB_REASONS: ReportReason[] = [
  "fraud_or_scam",
  "misleading_description",
  "inappropriate_content",
  "spam",
  "other",
]

export const APPLICATION_REASONS: ReportReason[] = [
  "fraud_or_scam",
  "spam",
  "inappropriate_content",
  "other",
]
