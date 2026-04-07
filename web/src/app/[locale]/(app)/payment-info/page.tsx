"use client"

import { useCallback, useEffect, useMemo, useRef, useState } from "react"
import { useParams, useSearchParams } from "next/navigation"
import { loadConnectAndInitialize } from "@stripe/connect-js"
import type { StripeConnectInstance } from "@stripe/connect-js"
import {
  ConnectAccountManagement,
  ConnectAccountOnboarding,
  ConnectComponentsProvider,
  ConnectNotificationBanner,
} from "@stripe/react-connect-js"
import { AlertCircle, ArrowLeft, Loader2, Sparkles } from "lucide-react"
import { useTranslations } from "next-intl"

import { API_BASE_URL } from "@/shared/lib/api-client"

import {
  AccountStatusCard,
  type AccountStatus,
} from "./components/account-status-card"
import { OnboardingWizard } from "./components/onboarding-wizard"
import { ROSE_APPEARANCE } from "./lib/rose-appearance"
import { mapAppLocaleToStripe } from "./lib/stripe-locale"

/**
 * Production-ready payment info page using Stripe Connect Embedded Components.
 *
 * Behavior:
 *  - On mount, fetches the user's account status.
 *  - If NO account yet → shows the onboarding wizard (country + business_type).
 *  - If account exists → shows status card + NotificationBanner + AccountManagement.
 *  - If account exists but details_submitted=false → shows AccountOnboarding to resume.
 *  - Polls account-status every 10s when user is actively editing to catch webhook updates.
 */

type BusinessType = "individual" | "company"

type AccountSessionResponse = {
  client_secret: string
  account_id: string
  expires_at: number
}

const STRIPE_PUBLISHABLE_KEY =
  process.env.NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY ?? ""

type Mode = "loading" | "wizard" | "onboarding" | "dashboard"

export default function PaymentInfoV2Page() {
  const params = useParams()
  const searchParams = useSearchParams()
  const appLocale = (params?.locale as string) || "en"
  const stripeLocale = useMemo(() => mapAppLocaleToStripe(appLocale), [appLocale])
  const t = useTranslations("paymentInfo")

  // Mobile WebView passes the JWT via ?token= so we can authenticate
  // API calls with Authorization: Bearer header (no cookie needed).
  const mobileToken = searchParams.get("token")
  // Mobile uses relative URLs (through Next.js proxy), desktop uses
  // the absolute API_BASE_URL (direct to Railway/backend).
  const apiBase = mobileToken ? "" : API_BASE_URL

  const [mode, setMode] = useState<Mode>("loading")
  const [status, setStatus] = useState<AccountStatus | null>(null)
  const [connectInstance, setConnectInstance] = useState<StripeConnectInstance | null>(null)
  const [creating, setCreating] = useState(false)
  const [errorMessage, setErrorMessage] = useState<string>("")

  // Ref holds the latest pending values so fetchClientSecret (called by
  // Stripe Connect asynchronously and possibly multiple times) always reads
  // the current country/business_type without stale-closure issues.
  const pendingRef = useRef<{ country: string | null }>({
    country: null,
  })

  // Build fetch options — if a mobile token is present, use Authorization
  // header instead of cookies. Works for both web (cookie) and mobile (JWT).
  const authHeaders = useMemo((): Record<string, string> => {
    if (!mobileToken) return {}
    return {
      Authorization: `Bearer ${mobileToken}`,
      "X-Auth-Mode": "token",
    }
  }, [mobileToken])

  /* ---------- Status fetch ---------- */
  const fetchStatus = useCallback(async (): Promise<AccountStatus | null> => {
    try {
      const res = await fetch(
        `${apiBase}/api/v1/payment-info/account-status`,
        {
          credentials: mobileToken ? "omit" : "include",
          headers: authHeaders,
        },
      )
      if (res.status === 404) return null
      if (!res.ok) return null
      return (await res.json()) as AccountStatus
    } catch {
      return null
    }
  }, [])

  /* ---------- Initial load ---------- */
  useEffect(() => {
    let cancelled = false
    ;(async () => {
      const s = await fetchStatus()
      if (cancelled) return
      if (!s) {
        setMode("wizard")
        return
      }
      setStatus(s)
      // First-time users (never submitted initial details) see the full
      // AccountOnboarding wizard. Everyone else lands on the dashboard where
      // the NotificationBanner surfaces any pending requirements and the
      // AccountManagement component lets them resolve them inline.
      if (!s.details_submitted) {
        setMode("onboarding")
      } else {
        setMode("dashboard")
      }
    })()
    return () => {
      cancelled = true
    }
  }, [fetchStatus])

  /* ---------- Polling (dashboard/onboarding) ---------- */
  useEffect(() => {
    if (mode !== "dashboard" && mode !== "onboarding") return
    const interval = setInterval(async () => {
      const s = await fetchStatus()
      if (s) setStatus(s)
    }, 10_000)
    return () => clearInterval(interval)
  }, [mode, fetchStatus])

  /* ---------- Stripe Connect init (once we have a session possible) ---------- */
  const fetchClientSecret = useCallback(async (): Promise<string> => {
    const { country } = pendingRef.current
    const res = await fetch(
      `${apiBase}/api/v1/payment-info/account-session`,
      {
        method: "POST",
        credentials: mobileToken ? "omit" : "include",
        headers: { "Content-Type": "application/json", ...authHeaders },
        body: JSON.stringify({ country }),
      },
    )
    if (!res.ok) {
      const errBody = (await res.json().catch(() => null)) as
        | { error?: { message?: string } }
        | null
      throw new Error(
        errBody?.error?.message ?? `HTTP ${res.status}: account session failed`,
      )
    }
    const payload = (await res.json()) as AccountSessionResponse
    return payload.client_secret
  }, [])

  const initializeConnect = useCallback(() => {
    if (!STRIPE_PUBLISHABLE_KEY) {
      setErrorMessage("NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY non défini")
      return null
    }
    const instance = loadConnectAndInitialize({
      publishableKey: STRIPE_PUBLISHABLE_KEY,
      fetchClientSecret,
      appearance: ROSE_APPEARANCE,
      locale: stripeLocale as never,
    })
    setConnectInstance(instance)
    return instance
  }, [fetchClientSecret, stripeLocale])

  /* ---------- Handlers ---------- */
  const handleWizardSubmit = async (country: string) => {
    setCreating(true)
    setErrorMessage("")
    // Write to ref synchronously so fetchClientSecret can read it immediately.
    pendingRef.current = { country }

    try {
      initializeConnect()
      setMode("onboarding")
    } catch (err) {
      setErrorMessage(
        err instanceof Error ? err.message : "Échec initialisation Stripe Connect",
      )
    } finally {
      setCreating(false)
    }
  }

  const handleResetToWizard = async () => {
    try {
      await fetch(`${apiBase}/api/v1/payment-info/account-session`, {
        method: "DELETE",
        credentials: mobileToken ? "omit" : "include",
        headers: authHeaders,
      })
    } catch {
      // Silent fail — best effort
    }
    setMode("wizard")
    setStatus(null)
    setConnectInstance(null)
    setErrorMessage("")
    pendingRef.current = { country: null }
  }

  const handleOnboardingExit = async () => {
    const s = await fetchStatus()
    if (s) {
      setStatus(s)
      if (s.details_submitted) {
        setMode("dashboard")
      }
    }
  }

  /* ---------- Ensure connectInstance for existing account mode ---------- */
  useEffect(() => {
    if ((mode === "onboarding" || mode === "dashboard") && !connectInstance && status) {
      initializeConnect()
    }
  }, [mode, connectInstance, status, initializeConnect])

  /* ---------- Render ---------- */
  return (
    <main className="min-h-screen bg-gradient-to-br from-slate-50 via-white to-rose-50/30 pb-16">
      {/* Header */}
      <header className="border-b border-slate-100 bg-white/70 backdrop-blur-xl">
        <div className="mx-auto max-w-5xl px-3 py-4 sm:px-6 sm:py-5">
          <div className="flex items-center gap-2.5">
            <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-gradient-to-br from-rose-500 to-rose-600 text-white shadow-sm">
              <Sparkles className="h-4 w-4" aria-hidden />
            </div>
            <div>
              <div className="text-[11px] font-semibold uppercase tracking-wider text-rose-600">
                {t("header")}
              </div>
              <h1 className="text-[15px] font-bold text-slate-900">
                {t("subheader")}
              </h1>
            </div>
          </div>
        </div>
      </header>

      <div className="mx-auto max-w-5xl px-2 pt-4 sm:px-6 sm:pt-8">
        {/* Error banner */}
        {errorMessage ? (
          <div
            role="alert"
            className="mb-6 flex items-start gap-3 rounded-xl border border-red-200 bg-red-50 p-4 animate-slide-up"
          >
            <AlertCircle className="mt-0.5 h-5 w-5 shrink-0 text-red-500" aria-hidden />
            <div className="flex-1">
              <p className="text-sm font-semibold text-red-900">{t("errorOccurred")}</p>
              <p className="mt-1 text-sm text-red-700">{errorMessage}</p>
            </div>
            <button
              onClick={() => setErrorMessage("")}
              className="text-xs font-medium text-red-600 hover:text-red-700"
            >
              {t("dismiss")}
            </button>
          </div>
        ) : null}

        {mode === "loading" ? (
          <div className="flex items-center justify-center py-20">
            <Loader2 className="h-6 w-6 animate-spin text-rose-500" aria-hidden />
            <span className="ml-3 text-sm text-slate-600">{t("loadingAccount")}</span>
          </div>
        ) : null}

        {mode === "wizard" ? (
          <OnboardingWizard loading={creating} onSubmit={handleWizardSubmit} />
        ) : null}

        {(mode === "onboarding" || mode === "dashboard") && connectInstance ? (
          <ConnectComponentsProvider connectInstance={connectInstance}>
            <div className="flex flex-col gap-4 animate-fade-in sm:gap-6">
              {/* Always-visible notification banner — also flags eventually_due
                 requirements (not just currently_due) so the user is aware
                 of what Stripe will need before it becomes urgent. */}
              <ConnectNotificationBanner
                collectionOptions={{
                  fields: "eventually_due",
                  futureRequirements: "include",
                }}
              />

              {/* Status card — always visible when account exists */}
              {status ? <AccountStatusCard status={status} /> : null}

              {/* Onboarding resume OR management editor */}
              {mode === "onboarding" ? (
                <section className="overflow-hidden rounded-2xl border border-slate-100 bg-white shadow-sm">
                  <div className="flex items-center justify-between border-b border-slate-100 bg-slate-50 px-3 py-2.5 sm:px-6 sm:py-3">
                    <h3 className="text-[13px] font-semibold text-slate-900">
                      {t("completeVerification")}
                    </h3>
                    <button
                      onClick={handleResetToWizard}
                      className="flex items-center gap-1.5 text-[12px] font-medium text-slate-500 transition-colors hover:text-rose-600"
                    >
                      <ArrowLeft className="h-3.5 w-3.5" aria-hidden />
                      {t("changeCountry")}
                    </button>
                  </div>
                  <div className="px-1 py-3 sm:p-6">
                    <ConnectAccountOnboarding
                      onExit={handleOnboardingExit}
                      collectionOptions={{
                        fields: "eventually_due",
                        futureRequirements: "include",
                      }}
                    />
                  </div>
                </section>
              ) : (
                <section className="overflow-hidden rounded-2xl border border-slate-100 bg-white shadow-sm">
                  <div className="border-b border-slate-100 bg-slate-50 px-4 py-3 sm:px-6">
                    <h3 className="text-[13px] font-semibold text-slate-900">
                      {t("manageInfo")}
                    </h3>
                    <p className="mt-0.5 text-[12px] text-slate-500">
                      {t("manageInfoHint")}
                    </p>
                  </div>
                  <div className="px-1 py-3 sm:p-6">
                    <ConnectAccountManagement
                      collectionOptions={{
                        fields: "eventually_due",
                        futureRequirements: "include",
                      }}
                    />
                  </div>
                </section>
              )}
            </div>
          </ConnectComponentsProvider>
        ) : null}
      </div>
    </main>
  )
}
