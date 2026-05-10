// Dashboard internal types — kept in this file (not in shared/) since
// they describe the dashboard composition surface and are not consumed
// elsewhere. If a downstream feature ever needs the same shapes,
// promote them to shared/types/ following the rule of three.

export type DashboardRole = "agency" | "enterprise" | "provider"

export type DashboardLayout =
  | "agency"
  | "enterprise"
  | "provider"
  | "referrer"

export type ActionSeverity = "info" | "warning" | "critical"

export interface DashboardAction {
  id: string
  severity: ActionSeverity
  /** Pre-translated label (caller passes the i18n result, not a key). */
  label: string
  /** Pre-translated CTA copy. */
  ctaLabel: string
  href: string
}
