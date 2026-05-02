export type AccountSection = "notifications" | "email" | "password" | "data-and-deletion"
export const DEFAULT_SECTION: AccountSection = "notifications"
export const VALID_SECTIONS: AccountSection[] = [
  "notifications",
  "email",
  "password",
  "data-and-deletion",
]
