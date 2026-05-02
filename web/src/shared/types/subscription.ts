// Shared subscription types (P9 — `UpgradeCta` / `UpgradeModal` UX is
// consumed cross-feature by proposal, so the types they read live in
// `shared/`). Internal-only subscription types stay scoped to the
// subscription feature.

export type Plan = "freelance" | "agency"

export type BillingCycle = "monthly" | "annual"
