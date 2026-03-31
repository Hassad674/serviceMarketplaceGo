export type TargetType = "message" | "user"

export type ReportReason =
  | "harassment"
  | "fraud"
  | "spam"
  | "inappropriate_content"
  | "fake_profile"
  | "unprofessional_behavior"
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
