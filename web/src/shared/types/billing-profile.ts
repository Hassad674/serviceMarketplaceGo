/**
 * Shared billing-profile types used across the `invoicing`, `wallet`,
 * and `subscription` features. The single source of truth for the
 * billing-profile data shape, lifted out of the invoicing feature so
 * other features can render the completion gate without importing
 * from `@/features/invoicing/...`.
 */

/**
 * One field the backend considers missing for the billing profile to
 * be considered "complete". The reason is a machine-readable token
 * (e.g. "required", "invalid_format") that the UI maps to localized
 * copy — never displayed verbatim.
 */
export type MissingField = {
  field: string
  reason: string
}
