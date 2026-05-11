"use client"

import { Loader2 } from "lucide-react"
import { useTranslations } from "next-intl"
import { BillingProfileForm } from "@/features/invoicing/components/billing-profile-form"
import { useBillingProfile } from "@/shared/hooks/billing-profile/use-billing-profile"
import { BillingProfileSummary } from "./billing-profile-summary"

/**
 * Inline embed for the billing-profile, mirroring the prestataire
 * settings page form but designed to live INSIDE another flow (today
 * the client payment page; tomorrow any flow that needs the receipt
 * identity on file before proceeding).
 *
 * Two modes:
 *   - "summary"  — compact read-only card with a single "Modifier"
 *                  CTA. Used when the profile is already complete
 *                  (the data was filled on the prestataire's
 *                  /settings/billing-profile page or persisted from
 *                  a prior payment).
 *   - "form"     — full BillingProfileForm (compact variant) so the
 *                  user can fix or complete the data without leaving
 *                  the page.
 *
 * The parent owns the toggle state — it decides when to flip to
 * "form" (typically when the user clicks "Modifier" or when the
 * profile is incomplete on first paint) and when to flip to
 * "summary" (on successful save).
 *
 * Why the parent owns the toggle: the payment page needs to gate the
 * Stripe Payment Element on `mode === "summary" && isBillingComplete`.
 * Keeping the toggle local to the embed would force a callback chain
 * for that gating logic; lifting it gives the parent direct read access.
 *
 * Data: reads the snapshot through `useBillingProfile()` (cached by
 * TanStack Query, 30s staleTime). The summary defers to the latest
 * snapshot from the cache so a save in another tab is immediately
 * reflected here.
 *
 * Soleil v2: inherited from the form sections and the summary card —
 * no tokens hardcoded here.
 */

export type BillingProfileEmbedProps = {
  /**
   * Controlled rendering mode. The parent toggles between modes based
   * on whether the user is editing.
   */
  mode: "summary" | "form"
  /**
   * Fired when the user clicks "Modifier" inside the summary view. The
   * parent should set its mode state to "form".
   */
  onEdit: () => void
  /**
   * Fired exactly once after a successful BillingProfileForm save when
   * the resulting profile passes server-side completeness. The parent
   * should set its mode state back to "summary".
   */
  onSaved: () => void
}

export function BillingProfileEmbed({
  mode,
  onEdit,
  onSaved,
}: BillingProfileEmbedProps) {
  const t = useTranslations("proposal.billingEmbed")
  const { data, isLoading } = useBillingProfile()

  if (isLoading) {
    return (
      <div
        role="status"
        aria-label={t("loading")}
        className="flex items-center justify-center rounded-2xl border border-border bg-surface p-4"
      >
        <Loader2
          className="h-4 w-4 animate-spin text-primary"
          aria-hidden="true"
        />
      </div>
    )
  }

  if (mode === "summary" && data?.profile) {
    return <BillingProfileSummary profile={data.profile} onEdit={onEdit} />
  }

  return (
    <div className="space-y-2">
      {!data?.is_complete && (
        <div className="rounded-2xl border border-warning/40 bg-amber-soft p-4 text-sm text-foreground">
          <p className="font-medium">{t("completePromptTitle")}</p>
          <p className="mt-1 text-[13px] text-muted-foreground">
            {t("completePromptBody")}
          </p>
        </div>
      )}
      <BillingProfileForm variant="compact" onSaved={onSaved} />
    </div>
  )
}
