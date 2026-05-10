// Single source of truth for the cookie / tracker inventory.
//
// Drives BOTH the vanilla-cookieconsent CMP runtime AND the public
// `/cookies` page so the two never drift. Adding a new cookie requires
// editing one place; the CMP toggles + the cookies disclosure table
// pick the change up automatically.
//
// Design notes:
//   - Categories follow the GDPR taxonomy (necessary / analytics).
//     A `functional` slot is reserved for future opt-in functional
//     cookies; today we have none, so the category is intentionally
//     omitted from the runtime config (no empty "Functional" toggle
//     in the user's preferences modal).
//   - The `services` array under each cookie is a flat enumeration
//     used by the cookies page table — we don't mirror the
//     vanilla-cookieconsent "service" granularity in the CMP
//     because the project only has one analytics provider per
//     toggle (PostHog + GA4 share the same opt-in).

/**
 * Logical category for a cookie / tracker. `functional` is reserved
 * for non-strict functional cookies; if added in the future, expose
 * the toggle in the CMP runtime too.
 */
export type ConsentCategory = "necessary" | "analytics" | "functional"

/**
 * One entry in the public `/cookies` table. The shape mirrors the
 * i18n message keys under `legal.cookies.rows.*` so the page can
 * iterate this list and reuse the existing translations.
 */
export type CookieEntry = {
  /** Stable i18n row key — `legal.cookies.rows.<key>.*`. */
  key: string
  category: ConsentCategory
}

/**
 * Cookie inventory. Order matters — it determines the row order on
 * the public `/cookies` page.
 */
export const COOKIE_INVENTORY: readonly CookieEntry[] = [
  { key: "session", category: "necessary" },
  { key: "consent", category: "necessary" },
  { key: "stripe", category: "necessary" },
  { key: "posthog", category: "analytics" },
  { key: "ga4", category: "analytics" },
  { key: "locale", category: "necessary" },
] as const

/**
 * Categories surfaced by the CMP runtime. `necessary` is always on
 * (readOnly); `analytics` is opt-in. `functional` is intentionally
 * absent — see file header.
 */
export const CMP_CATEGORIES: readonly ConsentCategory[] = ["necessary", "analytics"] as const

/** Test helper — quickly assert a category is recognised by the CMP. */
export function isCmpCategory(value: string): value is ConsentCategory {
  return (CMP_CATEGORIES as readonly string[]).includes(value)
}
