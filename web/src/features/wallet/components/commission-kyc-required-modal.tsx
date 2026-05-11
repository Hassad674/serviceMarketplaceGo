"use client"

import { ExternalLink, X } from "lucide-react"
import { useEffect } from "react"

import { cn } from "@/shared/lib/utils"

// CommissionKYCRequiredModal is the small dialog the wallet opens when
// the apporteur clicks "Retirer" on a pending_kyc/failed commission
// row but their Stripe Connect account is not yet payable. The 422
// response from POST /wallet/commissions/{id}/retry carries the
// onboarding URL; when present we offer a deep-link to finish KYC,
// otherwise we fall back to the in-app /payment-info page.
//
// Soleil v2: corail accent on the primary CTA, ivoire surface, calm
// muted-foreground for the explainer copy. No hex hardcoded — tokens
// only.

export interface CommissionKYCRequiredModalProps {
  open: boolean
  onClose: () => void
  /**
   * Stripe Connect onboarding URL extracted from the 422 envelope.
   * When the resolver failed (or is not wired), this is undefined and
   * the modal falls back to the static /payment-info redirect.
   */
  onboardingURL?: string
  /**
   * Fallback redirect when no onboardingURL is available. Defaults to
   * "/payment-info" — the in-app KYC page that hosts Stripe Embedded
   * Components.
   */
  fallbackRedirect?: string
}

export function CommissionKYCRequiredModal({
  open,
  onClose,
  onboardingURL,
  fallbackRedirect = "/payment-info",
}: CommissionKYCRequiredModalProps) {
  // Esc to close — small a11y nicety that matches the rest of the
  // wallet's modals (kyc-incomplete-modal uses the same pattern).
  useEffect(() => {
    if (!open) return
    function handleKey(event: KeyboardEvent) {
      if (event.key === "Escape") onClose()
    }
    window.addEventListener("keydown", handleKey)
    return () => window.removeEventListener("keydown", handleKey)
  }, [open, onClose])

  if (!open) return null

  const ctaHref = onboardingURL ?? fallbackRedirect
  const ctaIsExternal = Boolean(onboardingURL)

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-foreground/40 px-4 py-6"
      role="dialog"
      aria-modal="true"
      aria-labelledby="commission-kyc-modal-title"
      onClick={onClose}
    >
      <div
        className={cn(
          "relative w-full max-w-md rounded-2xl border border-border bg-card p-6 shadow-xl",
        )}
        onClick={(event) => event.stopPropagation()}
      >
        <button
          type="button"
          aria-label="Fermer"
          onClick={onClose}
          className="absolute right-4 top-4 rounded-full p-1.5 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
        >
          <X className="h-4 w-4" aria-hidden="true" />
        </button>

        <h2
          id="commission-kyc-modal-title"
          className="font-serif text-[20px] font-medium tracking-[-0.015em] text-foreground"
        >
          Termine ton KYC pour recevoir ta commission
        </h2>
        <p className="mt-2 text-[13.5px] leading-relaxed text-muted-foreground">
          Ta commission est prête à être versée mais ton compte Stripe Connect
          n&apos;a pas encore activé les paiements. Termine ton onboarding pour
          recevoir le virement.
        </p>

        <div className="mt-5 flex flex-col gap-2">
          <a
            href={ctaHref}
            target={ctaIsExternal ? "_blank" : undefined}
            rel={ctaIsExternal ? "noopener noreferrer" : undefined}
            className={cn(
              "inline-flex items-center justify-center gap-2 rounded-full bg-primary px-4 py-2.5",
              "text-[13.5px] font-semibold text-primary-foreground transition-colors hover:bg-primary-deep",
            )}
          >
            {ctaIsExternal ? "Terminer mon KYC" : "Compléter mes infos paiement"}
            {ctaIsExternal ? (
              <ExternalLink className="h-3.5 w-3.5" aria-hidden="true" />
            ) : null}
          </a>
          <button
            type="button"
            onClick={onClose}
            className="inline-flex items-center justify-center rounded-full border border-border bg-card px-4 py-2.5 text-[13.5px] font-medium text-foreground transition-colors hover:bg-muted"
          >
            Plus tard
          </button>
        </div>
      </div>
    </div>
  )
}
