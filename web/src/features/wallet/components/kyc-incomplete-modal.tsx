"use client"

import { useRouter } from "next/navigation"
import { AlertTriangle, ArrowRight } from "lucide-react"
import { Modal } from "@/shared/components/ui/modal"
import { cn } from "@/shared/lib/utils"

import { Button } from "@/shared/components/ui/button"
type KYCIncompleteModalProps = {
  open: boolean
  onClose: () => void
  /**
   * Optional override for the destination route. Defaults to the
   * payment-info screen where the Stripe onboarding lives. Exposed
   * so the same modal can be reused on locale-aware routes later.
   */
  destination?: string
  /**
   * Optional override for the message — defaults to the French copy
   * the backend sends back. Lets the wallet page use the server's
   * exact wording when it is available.
   */
  message?: string
}

/**
 * Wallet payout gate modal — opened when the backend refuses the
 * RequestPayout call with a 403 + `kyc_incomplete` envelope. Mirrors
 * the BillingProfileCompletionModal pattern (small, single CTA) so the
 * UX stays consistent across the two prerequisites.
 *
 * KYC is checked BEFORE the billing profile by the backend, so this
 * modal will be the first thing a brand-new provider sees when they
 * click "Retirer" — directing them to the Stripe onboarding rather
 * than to a billing form they cannot save before KYC is done.
 */
export function KYCIncompleteModal({
  open,
  onClose,
  destination = "/payment-info",
  message,
}: KYCIncompleteModalProps) {
  const router = useRouter()

  function handleCta() {
    router.push(destination)
    onClose()
  }

  return (
    <Modal
      open={open}
      onClose={onClose}
      title="Termine ton onboarding Stripe pour pouvoir retirer"
      maxWidthClassName="max-w-md"
    >
      <div className="space-y-4">
        <div className="flex items-start gap-3 rounded-lg border border-amber-200 bg-amber-50 p-3 text-sm text-amber-900 dark:border-amber-500/30 dark:bg-amber-500/10 dark:text-amber-200">
          <AlertTriangle
            className="mt-0.5 h-4 w-4 shrink-0"
            aria-hidden="true"
          />
          <p>
            {message ??
              "Avant de pouvoir retirer tes gains, finalise ton onboarding Stripe sur la page Infos paiement. Les virements ne sont activés qu'après vérification de ton identité par Stripe."}
          </p>
        </div>

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
            Aller à Infos paiement
            <ArrowRight className="h-4 w-4" aria-hidden="true" />
          </Button>
        </div>
      </div>
    </Modal>
  )
}
