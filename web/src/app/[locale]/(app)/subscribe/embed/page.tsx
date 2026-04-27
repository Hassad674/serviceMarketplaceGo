"use client"

import { useEffect, useMemo, useRef, useState } from "react"
import { useSearchParams } from "next/navigation"
import {
  EmbeddedCheckoutProvider,
  EmbeddedCheckout,
} from "@stripe/react-stripe-js"
import { ArrowLeft, Loader2 } from "lucide-react"
import { cn } from "@/shared/lib/utils"
import { stripePromise } from "@/shared/lib/stripe-client"
import { BillingProfileForm } from "@/features/invoicing/components/billing-profile-form"
import {
  useBillingProfile,
  useSyncBillingProfile,
} from "@/features/invoicing/hooks/use-billing-profile"
import { useSubscribe } from "@/features/subscription/hooks/use-subscribe"
import type {
  BillingCycle,
  Plan,
  SubscribeInput,
} from "@/features/subscription/types"

/**
 * Single-modal Premium subscribe flow rendered as a full page so the
 * mobile WebView and the desktop modal both reuse the same DOM.
 *
 * Step 1 — billing profile (our country-aware form)
 *   - At mount, kicks off `sync-from-stripe` exactly once when no
 *     prior sync ever happened (`synced_from_kyc_at == null`). The
 *     mutation only fills empty fields, so a user who has already
 *     edited their profile keeps their values.
 *   - The form's onSaved callback fires when the latest UPDATE
 *     succeeded AND the resulting profile passes CheckCompleteness.
 *     We then transition to step 2.
 *
 * Step 2 — Stripe Embedded Checkout
 *   - useSubscribe mutates POST /api/v1/subscriptions, which returns
 *     a `client_secret` for the embedded session. The Customer was
 *     pre-enriched with the billing profile by the backend so the
 *     Stripe form has nothing to re-collect (we set NO billing
 *     address collection and NO tax_id collection on the session).
 *   - The user pays inside <EmbeddedCheckout/>; Stripe redirects to
 *     /subscribe/return?session_id={ID} on completion.
 *
 * Mobile UX: the Flutter WebView watches for the navigation to
 * /subscribe/return?return_to=mobile and dismisses itself, no JS
 * bridge required.
 */
export default function SubscribeEmbedPage() {
  const searchParams = useSearchParams()
  const plan = searchParams.get("plan") as Plan | null
  const billingCycle = searchParams.get("cycle") as BillingCycle | null
  const autoRenew = searchParams.get("auto_renew") === "true"
  const returnTo = searchParams.get("return_to") ?? ""

  const [step, setStep] = useState<"billing" | "payment">("billing")

  const validParams =
    (plan === "freelance" || plan === "agency") &&
    (billingCycle === "monthly" || billingCycle === "annual")

  return (
    <div className="mx-auto flex min-h-[80vh] max-w-2xl flex-col p-6">
      <header className="mb-6">
        <h1 className="text-2xl font-semibold tracking-tight text-slate-900 dark:text-white">
          {step === "billing" ? "Informations de facturation" : "Paiement"}
        </h1>
        <p className="mt-1 text-sm text-slate-600 dark:text-slate-300">
          {step === "billing"
            ? "Vérifie tes informations légales avant le paiement. Elles serviront sur tes factures."
            : "Règle ton abonnement Premium en toute sécurité avec Stripe."}
        </p>
        {step === "payment" && (
          <button
            type="button"
            onClick={() => setStep("billing")}
            className={cn(
              "mt-3 inline-flex items-center gap-1.5 text-sm font-medium",
              "text-slate-600 transition-colors hover:text-rose-600",
              "dark:text-slate-300 dark:hover:text-rose-400",
            )}
          >
            <ArrowLeft className="h-4 w-4" aria-hidden="true" />
            Modifier mes informations
          </button>
        )}
      </header>

      {!validParams ? (
        <InvalidParamsCard />
      ) : step === "billing" ? (
        <BillingStep onContinue={() => setStep("payment")} />
      ) : (
        <PaymentStep
          plan={plan as Plan}
          billingCycle={billingCycle as BillingCycle}
          autoRenew={autoRenew}
          returnTo={returnTo}
        />
      )}
    </div>
  )
}

/**
 * Step 1 — wraps BillingProfileForm with the auto-sync-on-mount and
 * an explicit "Continuer" button that transitions when the profile is
 * complete.
 */
function BillingStep({ onContinue }: { onContinue: () => void }) {
  const { data, isLoading } = useBillingProfile()
  const sync = useSyncBillingProfile()
  const synced = useRef(false)

  // Auto-sync once when we know the profile has never been pre-filled
  // from Stripe KYC. Subsequent visits skip this — the user might have
  // intentionally cleared a field they don't want re-populated.
  useEffect(() => {
    if (synced.current) return
    if (!data) return
    if (data.profile.synced_from_kyc_at !== null) {
      synced.current = true
      return
    }
    synced.current = true
    sync.mutate()
    // sync.mutate is stable across renders; including it would force
    // an unbounded re-trigger on each parent render.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [data])

  if (isLoading) {
    return (
      <div className="flex items-center gap-2 text-sm text-slate-500 dark:text-slate-400">
        <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
        Chargement de ton profil…
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <BillingProfileForm variant="compact" onSaved={onContinue} />
      <div className="rounded-xl border border-slate-200 bg-slate-50 p-4 text-xs text-slate-600 dark:border-slate-700 dark:bg-slate-800/40 dark:text-slate-400">
        Une fois ton profil enregistré et complet, tu passes automatiquement à
        l'étape de paiement.
      </div>
    </div>
  )
}

/**
 * Step 2 — fires the subscribe mutation exactly once, then mounts the
 * Stripe Embedded Checkout with the resulting client_secret. The form
 * itself is provided by Stripe (PCI-compliant iframe).
 */
function PaymentStep({
  plan,
  billingCycle,
  autoRenew,
  returnTo,
}: {
  plan: Plan
  billingCycle: BillingCycle
  autoRenew: boolean
  returnTo: string
}) {
  const subscribe = useSubscribe()
  const fired = useRef(false)

  useEffect(() => {
    if (fired.current) return
    fired.current = true
    const input: SubscribeInput = {
      plan,
      billing_cycle: billingCycle,
      auto_renew: autoRenew,
    }
    subscribe.mutate(input)
    // subscribe.mutate is stable; deps would re-trigger needlessly.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [plan, billingCycle, autoRenew])

  const options = useMemo(() => {
    if (!subscribe.data?.client_secret) return null
    return { clientSecret: subscribe.data.client_secret }
  }, [subscribe.data?.client_secret])

  if (subscribe.isError) {
    return (
      <div className="rounded-xl border border-red-200 bg-red-50 p-6 text-sm text-red-700 dark:border-red-500/30 dark:bg-red-500/10 dark:text-red-300">
        Une erreur est survenue lors de la création du paiement. Réessaie ou
        contacte le support.
      </div>
    )
  }

  if (!options) {
    return (
      <div className="flex items-center gap-2 text-sm text-slate-500 dark:text-slate-400">
        <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
        Préparation du paiement…
      </div>
    )
  }

  return (
    <div data-testid="stripe-embedded-payment" data-return-to={returnTo}>
      <EmbeddedCheckoutProvider stripe={stripePromise} options={options}>
        <EmbeddedCheckout />
      </EmbeddedCheckoutProvider>
    </div>
  )
}

function InvalidParamsCard() {
  return (
    <div className="rounded-xl border border-amber-200 bg-amber-50 p-6 text-sm text-amber-800 dark:border-amber-500/30 dark:bg-amber-500/10 dark:text-amber-300">
      Paramètres de souscription invalides. Reviens à la page Premium et
      sélectionne un plan + un cycle de facturation.
    </div>
  )
}
