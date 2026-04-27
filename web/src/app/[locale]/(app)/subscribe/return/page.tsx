"use client"

import { useEffect, useState } from "react"
import Link from "next/link"
import { useSearchParams } from "next/navigation"
import { CheckCircle2, Loader2 } from "lucide-react"
import { useQueryClient } from "@tanstack/react-query"
import { cn } from "@/shared/lib/utils"
import { getMySubscription } from "@/features/subscription/api/subscription-api"
import { subscriptionQueryKey } from "@/features/subscription/hooks/keys"

/**
 * Landing page Stripe Embedded Checkout redirects to once the user
 * pays. Stripe substitutes `{CHECKOUT_SESSION_ID}` in the ReturnURL we
 * passed at session creation, so this page can read `?session_id=` to
 * correlate (we don't actually need it client-side because the webhook
 * is the source of truth, but we surface it for debugging).
 *
 * The webhook fires `customer.subscription.created` asynchronously, so
 * when this page first mounts the backend may still be writing the
 * row. We poll `/api/v1/subscriptions/me` every 1s for up to 15s and
 * seed the result into the TanStack cache so subsequent pages render
 * "Premium" instantly.
 *
 * Mobile flow: the Flutter WebView watches for navigations to
 * `/subscribe/return?return_to=mobile` and closes itself the moment it
 * sees that URL — even before the polling completes. The web UI here
 * still renders so the user has a brief "Activation en cours" before
 * the WebView dismisses.
 */
export default function SubscribeReturnPage() {
  const queryClient = useQueryClient()
  const searchParams = useSearchParams()
  const isMobile = searchParams.get("return_to") === "mobile"
  const [status, setStatus] = useState<"pending" | "ready" | "timeout">(
    "pending",
  )

  useEffect(() => {
    let cancelled = false
    const maxAttempts = 15
    const intervalMs = 1000

    async function poll(attempt: number) {
      if (cancelled) return
      try {
        const sub = await getMySubscription()
        if (cancelled) return
        if (sub) {
          queryClient.setQueryData(subscriptionQueryKey.me(), sub)
          setStatus("ready")
          return
        }
      } catch {
        // Network / 5xx — swallow and retry below. A real 404 is not
        // thrown by `getMySubscription` (it returns null).
      }
      if (attempt >= maxAttempts) {
        setStatus("timeout")
        return
      }
      setTimeout(() => poll(attempt + 1), intervalMs)
    }

    poll(1)
    return () => {
      cancelled = true
    }
  }, [queryClient])

  return (
    <div className="flex min-h-[60vh] items-center justify-center p-6">
      <div
        className={cn(
          "w-full max-w-md rounded-2xl border p-8 text-center shadow-sm",
          status === "pending"
            ? "border-slate-200 bg-white dark:border-slate-700 dark:bg-slate-800/40"
            : status === "ready"
              ? "border-emerald-200 bg-gradient-to-br from-emerald-50 to-white dark:border-emerald-500/30 dark:from-emerald-500/10 dark:to-slate-900/40"
              : "border-amber-200 bg-amber-50/60 dark:border-amber-500/30 dark:bg-amber-500/10",
        )}
      >
        {status === "pending" ? <PendingContent /> : null}
        {status === "ready" ? <ReadyContent isMobile={isMobile} /> : null}
        {status === "timeout" ? <TimeoutContent isMobile={isMobile} /> : null}
      </div>
    </div>
  )
}

function PendingContent() {
  return (
    <>
      <div className="mx-auto flex h-14 w-14 items-center justify-center rounded-full bg-slate-200 dark:bg-slate-700">
        <Loader2
          className="h-7 w-7 animate-spin text-slate-500 dark:text-slate-300"
          aria-hidden="true"
        />
      </div>
      <h1 className="mt-5 text-xl font-semibold text-slate-900 dark:text-white">
        Activation en cours…
      </h1>
      <p className="mt-2 text-sm text-slate-600 dark:text-slate-300">
        Paiement confirmé. Nous finalisons ton abonnement Premium — ça prend
        quelques secondes.
      </p>
    </>
  )
}

function ReadyContent({ isMobile }: { isMobile: boolean }) {
  return (
    <>
      <div className="mx-auto flex h-14 w-14 items-center justify-center rounded-full bg-emerald-500">
        <CheckCircle2 className="h-7 w-7 text-white" aria-hidden="true" />
      </div>
      <h1 className="mt-5 text-xl font-semibold text-slate-900 dark:text-white">
        Premium activé
      </h1>
      <p className="mt-2 text-sm text-slate-600 dark:text-slate-300">
        {isMobile
          ? "Tu peux fermer cette fenêtre et revenir à l'application."
          : "Tes prochaines missions ne seront plus soumises aux frais de plateforme."}
      </p>
      {!isMobile && (
        <Link
          href="/dashboard"
          className={cn(
            "mt-6 inline-flex w-full items-center justify-center rounded-full",
            "bg-gradient-to-r from-rose-500 to-rose-600 px-4 py-3 text-sm font-semibold text-white",
            "transition-all duration-200 hover:shadow-glow active:scale-[0.98]",
            "focus:outline-none focus:ring-2 focus:ring-rose-500/40",
          )}
        >
          Retour au tableau de bord
        </Link>
      )}
    </>
  )
}

function TimeoutContent({ isMobile }: { isMobile: boolean }) {
  return (
    <>
      <div className="mx-auto flex h-14 w-14 items-center justify-center rounded-full bg-amber-100 dark:bg-amber-500/20">
        <CheckCircle2
          className="h-7 w-7 text-amber-600 dark:text-amber-400"
          aria-hidden="true"
        />
      </div>
      <h1 className="mt-5 text-xl font-semibold text-slate-900 dark:text-white">
        Paiement confirmé
      </h1>
      <p className="mt-2 text-sm text-slate-600 dark:text-slate-300">
        Ton paiement est bien passé. L'activation Premium prend parfois un peu
        plus de temps — rafraîchis l'application dans une minute.
      </p>
      {!isMobile && (
        <Link
          href="/dashboard"
          className={cn(
            "mt-6 inline-flex w-full items-center justify-center rounded-full",
            "border border-slate-300 bg-white px-4 py-3 text-sm font-semibold text-slate-700",
            "transition-colors duration-200 hover:bg-slate-50",
            "dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:hover:bg-slate-700",
            "focus:outline-none focus:ring-2 focus:ring-rose-500/40",
          )}
        >
          Retour au tableau de bord
        </Link>
      )}
    </>
  )
}
