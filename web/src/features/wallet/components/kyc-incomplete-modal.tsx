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
 *
 * Soleil v2: amber-soft warning panel + outline "Plus tard" + filled
 * corail "Aller à Infos paiement" pill (rounded-full, calm tinted shadow).
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
        <div
          className={cn(
            "flex items-start gap-3 rounded-xl p-3 text-[13.5px]",
            "bg-amber-soft text-foreground",
          )}
        >
          <AlertTriangle
            className="mt-0.5 h-4 w-4 shrink-0 text-warning"
            aria-hidden="true"
          />
          <p className="leading-relaxed">
            {message ??
              "Avant de pouvoir retirer tes gains, finalise ton onboarding Stripe sur la page Infos paiement. Les virements ne sont activés qu'après vérification de ton identité par Stripe."}
          </p>
        </div>

        <div className="flex flex-col gap-2 pt-1 sm:flex-row sm:justify-end">
          <Button
            variant="ghost"
            size="auto"
            type="button"
            onClick={onClose}
            className={cn(
              "inline-flex items-center justify-center rounded-full px-5 py-2.5",
              "border border-border-strong bg-card",
              "text-sm font-semibold text-foreground",
              "transition-colors duration-150 hover:bg-primary-soft/40",
            )}
          >
            Plus tard
          </Button>
          <Button
            variant="ghost"
            size="auto"
            type="button"
            onClick={handleCta}
            className={cn(
              "inline-flex items-center justify-center gap-2 rounded-full px-5 py-2.5",
              "bg-primary text-sm font-bold text-primary-foreground",
              "transition-all duration-200 ease-out",
              "hover:bg-primary-deep hover:shadow-[0_4px_14px_rgba(232,93,74,0.28)]",
              "active:scale-[0.98]",
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
