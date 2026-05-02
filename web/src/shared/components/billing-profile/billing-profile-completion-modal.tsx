"use client"

import { useRouter } from "next/navigation"
import { AlertTriangle, ArrowRight } from "lucide-react"
import { Modal } from "@/shared/components/ui/modal"
import { cn } from "@/shared/lib/utils"
import type { MissingField } from "@/shared/types/billing-profile"
import { describeMissing } from "@/shared/lib/billing-profile/missing-fields-copy"

import { Button } from "@/shared/components/ui/button"
type BillingProfileCompletionModalProps = {
  open: boolean
  onClose: () => void
  /**
   * Missing-fields list returned by the backend. May come either
   * from the `useBillingProfile()` snapshot (proactive gate) or
   * from a 403 envelope (defensive gate). Either source is valid —
   * the modal only renders the labels.
   */
  missingFields: MissingField[]
  /**
   * Optional override for the destination route. Defaults to the
   * settings page where the form lives. Exposed so the same modal
   * can be reused on locale-aware routes later without forking it.
   */
  destination?: string
  /**
   * Optional path to return to after the user saves the profile.
   * When set, the modal appends `?return_to=<path>` to the
   * destination URL. The billing-profile page reads it and
   * router.push()es back on a successful save so the user lands
   * directly on the screen that triggered the gate (wallet, etc.)
   * instead of being stranded on the settings page.
   */
  returnTo?: string
}

/**
 * Lightweight gate modal opened when the wallet payout or the
 * subscription subscribe flow refuses to proceed because the
 * billing profile is incomplete. The modal explains what is
 * missing and routes to `/settings/billing-profile`. We never
 * embed the form here — completion modals must stay short.
 */
export function BillingProfileCompletionModal({
  open,
  onClose,
  missingFields,
  destination = "/settings/billing-profile",
  returnTo,
}: BillingProfileCompletionModalProps) {
  const router = useRouter()

  function handleCta() {
    const target = returnTo
      ? `${destination}?return_to=${encodeURIComponent(returnTo)}`
      : destination
    router.push(target)
    onClose()
  }

  return (
    <Modal
      open={open}
      onClose={onClose}
      title="Complète ton profil de facturation pour continuer"
      maxWidthClassName="max-w-md"
    >
      <div className="space-y-4">
        <div className="flex items-start gap-3 rounded-lg border border-amber-200 bg-amber-50 p-3 text-sm text-amber-900 dark:border-amber-500/30 dark:bg-amber-500/10 dark:text-amber-200">
          <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" aria-hidden="true" />
          <p>
            Avant de pouvoir effectuer cette opération, complète les
            informations suivantes. Elles apparaîtront sur tes factures.
          </p>
        </div>

        {missingFields.length > 0 ? (
          <ul className="space-y-1.5 text-sm text-slate-700 dark:text-slate-300">
            {missingFields.map((field) => (
              <li
                key={`${field.field}:${field.reason}`}
                className="flex items-start gap-2"
              >
                <span
                  aria-hidden="true"
                  className="mt-1 h-1.5 w-1.5 shrink-0 rounded-full bg-rose-500"
                />
                <span>{describeMissing(field)}</span>
              </li>
            ))}
          </ul>
        ) : (
          <p className="text-sm text-slate-500 dark:text-slate-400">
            Quelques informations restent à compléter sur ton profil de
            facturation.
          </p>
        )}

        <div className="flex flex-col gap-2 pt-1 sm:flex-row sm:justify-end">
          <Button variant="ghost" size="auto"
            type="button"
            onClick={onClose}
            className={cn(
              "inline-flex items-center justify-center rounded-lg border border-slate-200 bg-white px-4 py-2",
              "text-sm font-medium text-slate-700 transition-colors hover:bg-slate-50",
              "dark:border-slate-700 dark:bg-slate-800 dark:text-slate-200 dark:hover:bg-slate-700",
            )}
          >
            Plus tard
          </Button>
          <Button variant="ghost" size="auto"
            type="button"
            onClick={handleCta}
            className={cn(
              "inline-flex items-center justify-center gap-2 rounded-lg px-4 py-2",
              "text-sm font-semibold text-white",
              "gradient-primary hover:shadow-glow active:scale-[0.98]",
              "transition-all duration-200",
            )}
          >
            Compléter mon profil
            <ArrowRight className="h-4 w-4" aria-hidden="true" />
          </Button>
        </div>
      </div>
    </Modal>
  )
}
