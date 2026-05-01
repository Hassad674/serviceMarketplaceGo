"use client"

import { useCallback, useEffect, useMemo, useRef, useState } from "react"
import { useSearchParams } from "next/navigation"
import { useRouter } from "@i18n/navigation"
import {
  EmbeddedCheckoutProvider,
  EmbeddedCheckout,
} from "@stripe/react-stripe-js"
import { ArrowLeft, Loader2 } from "lucide-react"
import { ApiError } from "@/shared/lib/api-client"
import { cn } from "@/shared/lib/utils"
import { stripePromise } from "@/shared/lib/stripe-client"
import { BillingProfileForm } from "@/features/invoicing/components/billing-profile-form"
import {
  useBillingProfile,
  useSyncBillingProfile,
} from "@/features/invoicing/hooks/use-billing-profile"
import { subscribe } from "@/features/subscription/api/subscription-api"
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
  const router = useRouter()
  const plan = searchParams.get("plan") as Plan | null
  const billingCycle = searchParams.get("cycle") as BillingCycle | null
  const autoRenew = searchParams.get("auto_renew") === "true"
  const returnTo = searchParams.get("return_to") ?? ""

  const [step, setStep] = useState<"billing" | "payment">("billing")

  const validParams =
    (plan === "freelance" || plan === "agency") &&
    (billingCycle === "monthly" || billingCycle === "annual")

  // Cycle switch — rewrites the URL `cycle` param in place. PaymentStep's
  // useEffect lists `billingCycle` in its deps, so when the param flips the
  // component re-fires `subscribe()` and Stripe builds a fresh session for
  // the new cycle. `router.replace` is used so the user's back button
  // doesn't accumulate one history entry per toggle click.
  const handleCycleChange = useCallback(
    (next: BillingCycle) => {
      if (next === billingCycle) return
      const params = new URLSearchParams()
      searchParams.forEach((value, key) => params.set(key, value))
      params.set("cycle", next)
      router.replace(`/subscribe/embed?${params.toString()}`)
    },
    [billingCycle, router, searchParams],
  )

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

      {validParams && (
        <CycleToggle
          plan={plan as Plan}
          cycle={billingCycle as BillingCycle}
          onChange={handleCycleChange}
        />
      )}

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
 * Inline monthly/annual segmented control rendered between the page
 * header and the step content. Styling mirrors the `CycleToggle` inside
 * `upgrade-modal.tsx` (rose pill tabs) — duplicated rather than shared
 * to keep the subscription feature self-contained and avoid a
 * cross-feature import boundary.
 *
 * Prices are sourced from the same table as `upgrade-modal.tsx::pricing`
 * so the two surfaces stay visually consistent. If pricing ever changes,
 * both call sites must be updated.
 */
function CycleToggle({
  plan,
  cycle,
  onChange,
}: {
  plan: Plan
  cycle: BillingCycle
  onChange: (c: BillingCycle) => void
}) {
  const { monthlyAmount, annualPerMonth } = pricing(plan)
  return (
    <div
      role="tablist"
      aria-label="Periode de facturation"
      className="mb-6 flex items-center gap-1 rounded-full border border-slate-200 bg-slate-50 p-1 dark:border-slate-700 dark:bg-slate-800/50"
    >
      <CycleTab active={cycle === "monthly"} onClick={() => onChange("monthly")}>
        <span className="flex items-center justify-center gap-1.5">
          Mensuel
          <span className="text-[11px] font-normal text-slate-500 dark:text-slate-400">
            {monthlyAmount} €/mois
          </span>
        </span>
      </CycleTab>
      <CycleTab active={cycle === "annual"} onClick={() => onChange("annual")}>
        <span className="flex items-center justify-center gap-1.5">
          Annuel
          <span className="text-[11px] font-normal text-slate-500 dark:text-slate-400">
            {annualPerMonth} €/mois
          </span>
          <span className="inline-flex items-center rounded-full bg-rose-500 px-1.5 py-0.5 text-[9px] font-bold uppercase tracking-wider text-white">
            -21%
          </span>
        </span>
      </CycleTab>
    </div>
  )
}

function CycleTab({
  active,
  onClick,
  children,
}: {
  active: boolean
  onClick: () => void
  children: React.ReactNode
}) {
  return (
    <button
      type="button"
      role="tab"
      aria-selected={active}
      onClick={onClick}
      className={cn(
        "flex-1 rounded-full px-3 py-1.5 text-xs font-semibold transition-all duration-200",
        active
          ? "bg-white text-slate-900 shadow-sm dark:bg-slate-700 dark:text-white"
          : "text-slate-500 hover:text-slate-700 dark:text-slate-400 dark:hover:text-slate-200",
      )}
    >
      {children}
    </button>
  )
}

// Mirrors `upgrade-modal.tsx::pricing()`. Kept local to avoid a
// cross-component coupling — the two functions are intentionally
// independent so the modal can evolve its UX without dragging the
// embed page along.
function pricing(plan: Plan) {
  if (plan === "agency") {
    return { monthlyAmount: 49, annualPerMonth: 39 }
  }
  return { monthlyAmount: 19, annualPerMonth: 15 }
}

/**
 * Step 1 — wraps BillingProfileForm with the auto-sync-on-mount and
 * an explicit "Continuer" button that transitions when the profile is
 * complete.
 */
function BillingStep({ onContinue }: { onContinue: () => void }) {
  const { data } = useBillingProfile()
  const sync = useSyncBillingProfile()
  const synced = useRef(false)

  // Auto-sync from Stripe Connect KYC once per page mount when the
  // profile is INCOMPLETE. The previous gate ("only if synced_from_kyc_at
  // is null") was too conservative: a partial sync that filled nothing
  // would still flip the timestamp and lock the user out of retries
  // forever. The merge logic on the backend is "fill empty fields only",
  // so re-syncing never overwrites user-edited values.
  //
  // synced.current = once-per-mount mutex; React strict mode double-runs
  // the effect but the ref keeps the network call to a single attempt.
  useEffect(() => {
    if (synced.current) return
    if (!data) return
    if (data.is_complete) {
      // Nothing to fill — user already has a usable profile, skip the
      // round-trip entirely.
      synced.current = true
      return
    }
    synced.current = true
    sync.mutate()
    // sync.mutate is stable across renders; including it would force
    // an unbounded re-trigger on each parent render.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [data])

  return (
    <div className="space-y-6">
      {sync.isPending && (
        <div className="flex items-center gap-2 rounded-xl border border-slate-200 bg-slate-50 px-4 py-3 text-xs text-slate-600 dark:border-slate-700 dark:bg-slate-800/40 dark:text-slate-300">
          <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
          Récupération de tes informations Stripe…
        </div>
      )}
      {sync.isError && (
        <div className="rounded-xl border border-amber-200 bg-amber-50 px-4 py-3 text-xs text-amber-800 dark:border-amber-500/30 dark:bg-amber-500/10 dark:text-amber-300">
          Pré-remplissage Stripe indisponible (KYC peut-être incomplet).
          Remplis les champs manuellement — les autres champs s&apos;adapteront
          au pays choisi.
        </div>
      )}
      <BillingProfileForm variant="compact" onSaved={onContinue} />
      <div className="rounded-xl border border-slate-200 bg-slate-50 p-4 text-xs text-slate-600 dark:border-slate-700 dark:bg-slate-800/40 dark:text-slate-400">
        Une fois ton profil enregistré et complet, tu passes automatiquement à
        l&apos;étape de paiement.
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
  // Direct fetch + useState pattern instead of useMutation here. The
  // previous useEffect+mutate combo lost data across React strict-mode
  // dev double-mounts: each mount got a fresh useMutation instance, the
  // first mount's POST result fell on the floor, and the second mount
  // sat at isPending=true even after Stripe returned a session.
  // useState survives the remount cycle because the value is held in
  // the React tree itself.
  const [clientSecret, setClientSecret] = useState<string | null>(null)
  const [error, setError] = useState<{ code?: string; message: string } | null>(null)
  const [retryToken, setRetryToken] = useState(0)

  // The reset-then-fetch pattern below is intentional: the comment on
  // useState above explains why we can't migrate to useMutation, and the
  // sync resets at the top of the effect drive the visible "Préparation
  // du paiement…" loader. Both setState calls at the top of the effect
  // are guarded by the dep-array semantics (no cascading renders).
  useEffect(() => {
    let cancelled = false
    // eslint-disable-next-line react-hooks/set-state-in-effect -- intentional reset before async fetch, see comment block above the effect
    setError(null)
    // Drop the previous client_secret while the new session is being
    // created — this swaps the Stripe iframe for the "Préparation du
    // paiement…" loader card so the user sees a clean hand-off rather
    // than a stale price flickering until the new POST resolves. The
    // EmbeddedCheckoutProvider also gets a fresh React key once the
    // new secret arrives (see `key={clientSecret}` below), guaranteeing
    // the iframe actually re-renders with the new price.
    setClientSecret(null)
    const input: SubscribeInput = {
      plan,
      billing_cycle: billingCycle,
      auto_renew: autoRenew,
    }
    subscribe(input)
      .then((res) => {
        if (cancelled) return
        if (!res?.client_secret) {
          setError({
            message:
              "Réponse inattendue du serveur (client_secret manquant).",
          })
          return
        }
        setClientSecret(res.client_secret)
      })
      .catch((err: unknown) => {
        if (cancelled) return
        if (err instanceof ApiError) {
          setError({ code: err.code, message: err.message })
          return
        }
        setError({
          message:
            err instanceof Error ? err.message : "Erreur inconnue.",
        })
      })
    return () => {
      cancelled = true
    }
  }, [plan, billingCycle, autoRenew, retryToken])

  const options = useMemo(() => {
    if (!clientSecret) return null
    return { clientSecret }
  }, [clientSecret])

  if (error) {
    return (
      <div className="space-y-3 rounded-xl border border-red-200 bg-red-50 p-6 text-sm text-red-700 dark:border-red-500/30 dark:bg-red-500/10 dark:text-red-300">
        <p className="font-medium">Le paiement n&apos;a pas pu démarrer.</p>
        <p className="text-xs">
          {error.message}
          {error.code && (
            <span className="ml-1 rounded bg-red-100 px-1.5 py-0.5 font-mono text-[10px] text-red-800 dark:bg-red-500/20 dark:text-red-200">
              {error.code}
            </span>
          )}
        </p>
        <button
          type="button"
          onClick={() => {
            setError(null)
            setClientSecret(null)
            setRetryToken((t) => t + 1)
          }}
          className="rounded-lg border border-red-300 bg-white px-3 py-1.5 text-xs font-medium text-red-700 hover:bg-red-50 dark:border-red-500/40 dark:bg-transparent dark:text-red-300"
        >
          Réessayer
        </button>
      </div>
    )
  }

  if (!options) {
    return (
      <div className="space-y-2">
        <div className="flex items-center gap-2 text-sm text-slate-500 dark:text-slate-400">
          <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
          Préparation du paiement…
        </div>
        <p className="text-xs text-slate-400 dark:text-slate-500">
          Création de la session Stripe en cours…
        </p>
      </div>
    )
  }

  return (
    <div
      data-testid="stripe-embedded-payment"
      data-return-to={returnTo}
      className="min-h-[400px]"
    >
      {/*
        key={clientSecret} forces React to unmount the previous
        EmbeddedCheckoutProvider subtree and mount a fresh one whenever
        the client secret changes (e.g. after the user toggles between
        mensuel/annuel and a new Stripe session is created). The Stripe
        provider caches the underlying Checkout instance keyed by the
        clientSecret it was first mounted with — updating
        options.clientSecret in place does NOT swap the iframe, so
        without the remount the user keeps seeing the old price.
      */}
      <EmbeddedCheckoutProvider
        key={clientSecret}
        stripe={stripePromise}
        options={options}
      >
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
