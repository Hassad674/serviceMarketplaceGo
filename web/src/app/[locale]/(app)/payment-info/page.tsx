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
import { AlertCircle, ArrowLeft, Loader2, ShieldX, Sparkles } from "lucide-react"
import { useTranslations } from "next-intl"

import { API_BASE_URL } from "@/shared/lib/api-client"
import { usePermissionStatus } from "@/shared/hooks/use-permissions"

import {
  AccountStatusCard,
  type AccountStatus,
} from "./components/account-status-card"
import { OnboardingWizard } from "./components/onboarding-wizard"
import { ROSE_APPEARANCE } from "./lib/rose-appearance"
import { mapAppLocaleToStripe } from "./lib/stripe-locale"

import { Button } from "@/shared/components/ui/button"
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
  const tPerm = useTranslations("permissions")
  const kycPermission = usePermissionStatus("kyc.manage")
  const canManageKyc = kycPermission.granted
  const isPermissionLoading = kycPermission.status === "loading"
  const isPermissionDenied = kycPermission.status === "denied"

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
  }, [apiBase, authHeaders, mobileToken])

  /* ---------- Initial load (skip when user lacks kyc.manage) ---------- */
  useEffect(() => {
    if (!canManageKyc) return
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
  }, [fetchStatus, canManageKyc])

  /* ---------- Polling (dashboard/onboarding) ---------- */
  useEffect(() => {
    if (!canManageKyc) return
    if (mode !== "dashboard" && mode !== "onboarding") return
    const interval = setInterval(async () => {
      const s = await fetchStatus()
      if (s) setStatus(s)
    }, 10_000)
    return () => clearInterval(interval)
  }, [mode, fetchStatus, canManageKyc])

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
  }, [apiBase, authHeaders, mobileToken])

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
  // The setState happens through initializeConnect() but is gated by
  // `!connectInstance` so the effect runs at most once per status change
  // — no cascading renders. The Stripe Connect SDK must be initialised
  // synchronously after the account status arrives, which is an
  // external-system bootstrap; useState lazy init can't help because
  // we don't have the data on first render.
  useEffect(() => {
    if ((mode === "onboarding" || mode === "dashboard") && !connectInstance && status) {
      // eslint-disable-next-line react-hooks/set-state-in-effect -- gated by !connectInstance above, runs at most once per status change
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

      <div className="mx-auto max-w-5xl px-0 pt-3 sm:px-6 sm:pt-8">
        {/* Loading skeleton — render while the permission check is still
            in flight so we never flash the "Accès restreint" card. */}
        {isPermissionLoading ? (
          <div
            data-testid="permission-loading"
            className="flex items-center justify-center py-20"
          >
            <Loader2
              className="h-6 w-6 animate-spin text-rose-500"
              aria-hidden
            />
            <span className="ml-3 text-sm text-slate-600">
              {t("loadingAccount")}
            </span>
          </div>
        ) : null}

        {/* Permission denied card — only after the permission check has
            settled with a definitive `denied`. */}
        {isPermissionDenied ? (
          <div className="mx-3 mt-4 flex flex-col items-center gap-4 rounded-2xl border border-slate-100 bg-white p-8 text-center shadow-sm sm:mx-0 sm:mt-8 sm:p-12">
            <div className="flex h-14 w-14 items-center justify-center rounded-2xl bg-rose-50">
              <ShieldX className="h-7 w-7 text-rose-500" aria-hidden />
            </div>
            <div>
              <h2 className="text-lg font-bold text-slate-900">
                {tPerm("restrictedTitle")}
              </h2>
              <p className="mt-2 max-w-md text-sm text-slate-500">
                {tPerm("noKycManage")}
              </p>
              <p className="mt-1 text-xs text-slate-400">
                {tPerm("restrictedDescription")}
              </p>
            </div>
          </div>
        ) : null}

        {/* Error banner */}
        {canManageKyc && errorMessage ? (
          <div
            role="alert"
            className="mb-6 flex items-start gap-3 rounded-xl border border-red-200 bg-red-50 p-4 animate-slide-up"
          >
            <AlertCircle className="mt-0.5 h-5 w-5 shrink-0 text-red-500" aria-hidden />
            <div className="flex-1">
              <p className="text-sm font-semibold text-red-900">{t("errorOccurred")}</p>
              <p className="mt-1 text-sm text-red-700">{errorMessage}</p>
            </div>
            <Button variant="ghost" size="auto"
              onClick={() => setErrorMessage("")}
              className="text-xs font-medium text-red-600 hover:text-red-700"
            >
              {t("dismiss")}
            </Button>
          </div>
        ) : null}

        {canManageKyc && mode === "loading" ? (
          <div className="flex items-center justify-center py-20">
            <Loader2 className="h-6 w-6 animate-spin text-rose-500" aria-hidden />
            <span className="ml-3 text-sm text-slate-600">{t("loadingAccount")}</span>
          </div>
        ) : null}

        {canManageKyc && mode === "wizard" ? (
          <OnboardingWizard loading={creating} onSubmit={handleWizardSubmit} />
        ) : null}

        {canManageKyc && (mode === "onboarding" || mode === "dashboard") && connectInstance ? (
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
                <section className="overflow-hidden sm:rounded-2xl sm:border sm:border-slate-100 sm:bg-white sm:shadow-sm">
                  <div className="flex items-center justify-between border-b border-slate-100 bg-slate-50 px-2 py-2.5 sm:px-6 sm:py-3">
                    <h3 className="text-[13px] font-semibold text-slate-900">
                      {t("completeVerification")}
                    </h3>
                    <Button variant="ghost" size="auto"
                      onClick={handleResetToWizard}
                      className="flex items-center gap-1.5 text-[12px] font-medium text-slate-500 transition-colors hover:text-rose-600"
                    >
                      <ArrowLeft className="h-3.5 w-3.5" aria-hidden />
                      {t("changeCountry")}
                    </Button>
                  </div>
                  <div className="sm:p-6">
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
                <section className="overflow-hidden sm:rounded-2xl sm:border sm:border-slate-100 sm:bg-white sm:shadow-sm">
                  <div className="border-b border-slate-100 bg-slate-50 px-2 py-2.5 sm:px-6 sm:py-3">
                    <h3 className="text-[13px] font-semibold text-slate-900">
                      {t("manageInfo")}
                    </h3>
                    <p className="mt-0.5 text-[12px] text-slate-500">
                      {t("manageInfoHint")}
                    </p>
                  </div>
                  <div className="sm:p-6">
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
