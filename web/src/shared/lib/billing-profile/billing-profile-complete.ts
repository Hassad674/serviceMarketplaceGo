/**
 * Pure completeness check for a BillingProfile.
 *
 * Mirrors the contract enforced server-side: the canonical
 * `is_complete` flag on the snapshot is computed by the backend and
 * remains the single source of truth for any gate (payment, payout,
 * subscription). This helper is a defensive/UX mirror used by:
 *   - Tests that need a deterministic completeness check without
 *     standing up a backend.
 *   - The mobile widget which works the same way as web.
 *
 * Rules (kept aligned with the backend's
 * internal/domain/billing/profile.go `IsComplete()` rule set):
 *   - Required for any profile: legal_name, country, address_line1,
 *     postal_code, city.
 *   - Required additionally when profile_type === 'business': tax_id
 *     (SIRET in France, equivalent abroad).
 *   - Optional everywhere: vat_number, trading_name, legal_form,
 *     address_line2, invoicing_email.
 *
 * The function intentionally returns a boolean only — callers that
 * need the list of missing fields should read the snapshot's
 * `missing_fields` array (server-authoritative).
 */

import type { BillingProfile } from "@/shared/types/billing-profile"

type ProfileLike = Partial<
  Pick<
    BillingProfile,
    | "legal_name"
    | "country"
    | "address_line1"
    | "postal_code"
    | "city"
    | "profile_type"
    | "tax_id"
  >
>

function isFilled(value: string | undefined | null): boolean {
  if (value === undefined || value === null) return false
  return value.trim() !== ""
}

export function checkBillingProfileComplete(
  profile: ProfileLike | null | undefined,
): boolean {
  if (!profile) return false
  if (!isFilled(profile.legal_name)) return false
  if (!isFilled(profile.country)) return false
  if (!isFilled(profile.address_line1)) return false
  if (!isFilled(profile.postal_code)) return false
  if (!isFilled(profile.city)) return false
  if (profile.profile_type === "business" && !isFilled(profile.tax_id)) {
    return false
  }
  return true
}
