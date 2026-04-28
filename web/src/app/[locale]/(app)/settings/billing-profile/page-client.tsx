"use client"

import { useRouter, useSearchParams } from "next/navigation"
import { BillingProfileForm } from "@/features/invoicing/components/billing-profile-form"

// Whitelist of paths the form is allowed to redirect to after save.
// Anchored to "/" + a path segment so an attacker can't smuggle an
// open-redirect target through the query string. Locale-prefixed
// paths (e.g. "/fr/wallet") are routed-through unchanged because
// Next.js handles the locale on its own.
const ALLOWED_RETURN_PATHS = new Set([
  "/wallet",
  "/payment-info",
  "/invoices",
  "/settings",
])

function safeReturnTo(raw: string | null): string | null {
  if (!raw) return null
  if (!raw.startsWith("/")) return null
  // Strip an optional locale segment (/fr/..., /en/...) before
  // checking against the allow-list so deep links keep working.
  const stripped = raw.replace(/^\/[a-z]{2}(?=\/|$)/, "")
  if (!ALLOWED_RETURN_PATHS.has(stripped)) return null
  return raw
}

export function BillingProfilePageClient() {
  const router = useRouter()
  const searchParams = useSearchParams()
  const returnTo = safeReturnTo(searchParams.get("return_to"))

  return (
    <div className="mx-auto max-w-3xl px-4 py-8">
      <header className="mb-6">
        <h1 className="text-2xl font-bold text-slate-900 dark:text-white">
          Profil de facturation
        </h1>
        <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">
          Ces informations apparaissent sur les factures que la plateforme
          émet à ton organisation. Elles doivent être complètes pour pouvoir
          retirer ton solde et souscrire à un abonnement Premium.
        </p>
      </header>
      <BillingProfileForm
        onSaved={
          returnTo
            ? () => {
                router.push(returnTo)
              }
            : undefined
        }
      />
    </div>
  )
}
