"use client"

import { useTranslations } from "next-intl"
import { Modal } from "@/shared/components/ui/modal"
import { BillingProfileForm } from "@/features/invoicing/components/billing-profile-form"

/**
 * BillingProfileInlineModal embeds the full billing-profile form
 * (compact variant) inside a modal so the user fixes their billing
 * identity WITHOUT leaving the current page.
 *
 * Used by the proposal payment flow: when the client organization has
 * an incomplete billing profile, the backend rejects the InitiatePayment
 * call with 412 `billing_profile_incomplete`. Instead of redirecting
 * the user to /settings/billing-profile (the existing
 * BillingProfileCompletionModal pattern, used by the wallet payout +
 * subscription flows), we open this inline modal so the user saves
 * their info and the parent component immediately retries the payment
 * call.
 *
 * The form itself is the canonical
 * `@/features/invoicing/components/billing-profile-form` — same code
 * path as the standalone /settings/billing-profile page. The compact
 * variant hides the editorial header and pumps the rendered density
 * for modal use.
 *
 * Soleil v2 styling is inherited from the shared Modal primitive
 * (ivoire surface, rounded-2xl) and the form section components
 * (corail accent, mono labels). No Soleil tokens are hardcoded here.
 *
 * Props:
 *   - `open` / `onClose`: controlled state owned by the parent.
 *   - `onSaved`: invoked exactly once after a successful save where
 *     the resulting profile passes server-side completeness. Parents
 *     use it to chain the retry of the gated action (e.g. retry the
 *     proposal payment intent creation).
 */
export type BillingProfileInlineModalProps = {
  open: boolean
  onClose: () => void
  onSaved: () => void
}

export function BillingProfileInlineModal({
  open,
  onClose,
  onSaved,
}: BillingProfileInlineModalProps) {
  const t = useTranslations("billingProfileInlineModal")
  return (
    <Modal
      open={open}
      onClose={onClose}
      title={t("title")}
      maxWidthClassName="max-w-2xl"
    >
      <div className="space-y-4">
        <p className="text-sm text-muted-foreground">{t("subtitle")}</p>
        <BillingProfileForm variant="compact" onSaved={onSaved} />
      </div>
    </Modal>
  )
}
