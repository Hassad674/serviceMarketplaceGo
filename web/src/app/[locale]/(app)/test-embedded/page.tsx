"use client"

import { useCallback, useEffect, useMemo, useState } from "react"
import { loadConnectAndInitialize } from "@stripe/connect-js"
import type { StripeConnectInstance } from "@stripe/connect-js"
import {
  ConnectAccountOnboarding,
  ConnectComponentsProvider,
} from "@stripe/react-connect-js"
import { AlertCircle, ArrowRight, Loader2, Sparkles } from "lucide-react"

import { API_BASE_URL } from "@/shared/lib/api-client"

import { BusinessTypeCard } from "./components/business-type-card"
import { ContextSidebar } from "./components/context-sidebar"
import { CountrySelector } from "./components/country-selector"
import { StepProgress } from "./components/step-progress"
import { SuccessState } from "./components/success-state"
import { TrustSignals } from "./components/trust-signals"

/**
 * Isolated test page for Stripe Connect Embedded Components.
 *
 * 3-step state machine:
 *  1. select  → user picks country + business_type (individual/company)
 *  2. kyc     → Stripe Embedded onboarding with rose theming + context sidebar
 *  3. success → celebration + account recap + next steps
 *
 * This page is fully isolated from the existing /payment-info custom flow.
 */

type Step = "select" | "kyc" | "success"
type BusinessType = "individual" | "company"

type AccountSessionResponse = {
  client_secret: string
  account_id: string
  expires_at: number
}

type AccountStatusResponse = {
  account_id: string
  country: string
  business_type: string
  charges_enabled: boolean
  payouts_enabled: boolean
  details_submitted: boolean
  requirements_currently_due: string[]
  requirements_past_due: string[]
  requirements_count: number
}

const STRIPE_PUBLISHABLE_KEY =
  process.env.NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY ?? ""

// Full Stripe Connect Appearance API theming — pushes rose branding into
// every component slot Stripe exposes. Reference:
// https://docs.stripe.com/connect/embedded-appearance-options
const ROSE_APPEARANCE = {
  variables: {
    // Core palette
    colorPrimary: "#F43F5E",
    colorBackground: "#FFFFFF",
    colorText: "#0F172A",
    colorSecondaryText: "#475569",
    colorBorder: "#E2E8F0",
    colorDanger: "#EF4444",
    colorWarning: "#F59E0B",
    colorSuccess: "#22C55E",
    colorSecondaryLinkText: "#E11D48",

    // Typography
    fontFamily: "Inter, ui-sans-serif, system-ui, sans-serif",
    fontSizeBase: "15px",
    headingXlFontSize: "28px",
    headingXlFontWeight: "800",
    headingLgFontSize: "22px",
    headingLgFontWeight: "700",
    headingMdFontSize: "17px",
    headingMdFontWeight: "600",
    headingSmFontSize: "14px",
    headingSmFontWeight: "600",
    headingXsFontSize: "13px",
    headingXsFontWeight: "600",
    bodyMdFontSize: "15px",
    bodyMdFontWeight: "400",
    bodySmFontSize: "13px",
    bodySmFontWeight: "400",
    labelMdFontSize: "13px",
    labelMdFontWeight: "600",
    labelSmFontSize: "12px",
    labelSmFontWeight: "500",
    buttonPrimaryFontWeight: "600",
    buttonPrimaryFontSize: "14px",
    buttonSecondaryFontWeight: "600",
    buttonSecondaryFontSize: "14px",

    // Layout rhythm
    spacingUnit: "12px",
    borderRadius: "12px",
    buttonBorderRadius: "10px",
    formBorderRadius: "10px",
    badgeBorderRadius: "999px",
    overlayBorderRadius: "16px",
    formBorderWidth: "1.5px",
    buttonBorderWidth: "1px",

    // Surfaces
    offsetBackgroundColor: "#F8FAFC",
    formBackgroundColor: "#FFFFFF",
    formHighlightColorBorder: "#F43F5E",
    formAccentColor: "#F43F5E",

    // Primary button
    buttonPrimaryColorBackground: "#F43F5E",
    buttonPrimaryColorBorder: "#F43F5E",
    buttonPrimaryColorText: "#FFFFFF",

    // Secondary button
    buttonSecondaryColorBackground: "#FFFFFF",
    buttonSecondaryColorBorder: "#E2E8F0",
    buttonSecondaryColorText: "#0F172A",

    // Action links
    actionPrimaryColorText: "#E11D48",
    actionSecondaryColorText: "#475569",

    // Badges
    badgeNeutralColorBackground: "#F1F5F9",
    badgeNeutralColorText: "#475569",
    badgeNeutralColorBorder: "#E2E8F0",
    badgeSuccessColorBackground: "#DCFCE7",
    badgeSuccessColorText: "#166534",
    badgeSuccessColorBorder: "#BBF7D0",
    badgeWarningColorBackground: "#FEF3C7",
    badgeWarningColorText: "#92400E",
    badgeWarningColorBorder: "#FDE68A",
    badgeDangerColorBackground: "#FEE2E2",
    badgeDangerColorText: "#991B1B",
    badgeDangerColorBorder: "#FECACA",
  },
} as const

export default function TestEmbeddedPage() {
  const [step, setStep] = useState<Step>("select")
  const [country, setCountry] = useState<string | null>(null)
  const [businessType, setBusinessType] = useState<BusinessType | null>(null)
  const [connectInstance, setConnectInstance] = useState<StripeConnectInstance | null>(null)
  const [accountId, setAccountId] = useState<string>("")
  const [accountStatus, setAccountStatus] = useState<AccountStatusResponse | null>(null)
  const [creating, setCreating] = useState(false)
  const [errorMessage, setErrorMessage] = useState<string>("")

  const fetchClientSecret = useMemo(() => {
    return async (): Promise<string> => {
      const res = await fetch(
        `${API_BASE_URL}/api/v1/payment-info/account-session`,
        {
          method: "POST",
          credentials: "include",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ country, business_type: businessType }),
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
      setAccountId(payload.account_id)
      return payload.client_secret
    }
  }, [country, businessType])

  const fetchStatus = useCallback(async (): Promise<AccountStatusResponse | null> => {
    const res = await fetch(
      `${API_BASE_URL}/api/v1/payment-info/account-status`,
      { credentials: "include" },
    )
    if (!res.ok) return null
    return (await res.json()) as AccountStatusResponse
  }, [])

  const handleContinue = async () => {
    if (!country || !businessType) return
    setCreating(true)
    setErrorMessage("")

    if (!STRIPE_PUBLISHABLE_KEY) {
      setErrorMessage("NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY non défini")
      setCreating(false)
      return
    }

    try {
      const instance = loadConnectAndInitialize({
        publishableKey: STRIPE_PUBLISHABLE_KEY,
        fetchClientSecret,
        appearance: ROSE_APPEARANCE,
        locale: "fr-FR",
      })
      setConnectInstance(instance)
      setStep("kyc")
    } catch (err) {
      setErrorMessage(
        err instanceof Error ? err.message : "Échec initialisation Stripe Connect",
      )
    } finally {
      setCreating(false)
    }
  }

  const handleOnboardingExit = async () => {
    const status = await fetchStatus()
    setAccountStatus(status)
    setStep("success")
  }

  const handleRestart = async () => {
    // Wipe the test account mapping server-side so a fresh account is
    // created on the next session (with the newly selected country/type).
    try {
      await fetch(`${API_BASE_URL}/api/v1/payment-info/account-session`, {
        method: "DELETE",
        credentials: "include",
      })
    } catch {
      // Silent fail — not critical for test page.
    }
    setStep("select")
    setCountry(null)
    setBusinessType(null)
    setConnectInstance(null)
    setAccountId("")
    setAccountStatus(null)
    setErrorMessage("")
  }

  const currentStepNumber = step === "select" ? 1 : step === "kyc" ? 2 : 3

  return (
    <main className="min-h-screen bg-gradient-to-br from-slate-50 via-white to-rose-50/30 pb-16">
      {/* Header */}
      <header className="border-b border-slate-100 bg-white/70 backdrop-blur-xl">
        <div className="mx-auto max-w-6xl px-6 py-5">
          <div className="flex items-center gap-2.5">
            <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-gradient-to-br from-rose-500 to-rose-600 text-white shadow-sm">
              <Sparkles className="h-4 w-4" aria-hidden />
            </div>
            <div>
              <div className="text-[11px] font-semibold uppercase tracking-wider text-rose-600">
                Configuration des paiements
              </div>
              <h1 className="text-[15px] font-bold text-slate-900">
                Activation de votre compte
              </h1>
            </div>
          </div>
        </div>
      </header>

      <div className="mx-auto max-w-6xl px-6 pt-8">
        {/* Progress bar */}
        <div className="mb-8 rounded-2xl border border-slate-100 bg-white/80 p-5 shadow-sm backdrop-blur-sm">
          <StepProgress currentStep={currentStepNumber} />
        </div>

        {/* Error banner */}
        {errorMessage ? (
          <div
            role="alert"
            className="mb-6 flex items-start gap-3 rounded-xl border border-red-200 bg-red-50 p-4 animate-slide-up"
          >
            <AlertCircle
              className="mt-0.5 h-5 w-5 shrink-0 text-red-500"
              aria-hidden
            />
            <div className="flex-1">
              <p className="text-sm font-semibold text-red-900">Une erreur est survenue</p>
              <p className="mt-1 text-sm text-red-700">{errorMessage}</p>
            </div>
            <button
              onClick={() => setErrorMessage("")}
              className="text-xs font-medium text-red-600 hover:text-red-700"
            >
              Fermer
            </button>
          </div>
        ) : null}

        {/* Content */}
        {step === "select" ? (
          <section className="mx-auto max-w-2xl animate-slide-up">
            <div className="mb-8 text-center">
              <h2 className="text-3xl font-extrabold tracking-tight text-slate-900 sm:text-4xl">
                Commençons par les bases
              </h2>
              <p className="mx-auto mt-3 max-w-md text-[15px] leading-relaxed text-slate-600">
                Quelques informations pour adapter le formulaire à votre situation fiscale.
              </p>
            </div>

            <div className="rounded-3xl border border-slate-100 bg-white p-6 shadow-sm sm:p-8">
              <div className="flex flex-col gap-6">
                <div>
                  <label className="mb-2 block text-[13px] font-semibold text-slate-900">
                    Pays de résidence fiscale
                  </label>
                  <CountrySelector
                    value={country}
                    onChange={setCountry}
                    disabled={creating}
                  />
                  <p className="mt-2 text-[12px] text-slate-500">
                    Le pays où vous déclarez vos revenus, indépendamment de votre nationalité.
                  </p>
                </div>

                <div>
                  <label className="mb-3 block text-[13px] font-semibold text-slate-900">
                    Type de compte
                  </label>
                  <BusinessTypeCard
                    value={businessType}
                    onChange={setBusinessType}
                    disabled={creating}
                  />
                </div>

                <button
                  onClick={handleContinue}
                  disabled={!country || !businessType || creating}
                  className={`flex h-12 items-center justify-center gap-2 rounded-xl text-[15px] font-semibold transition-all ${
                    !country || !businessType || creating
                      ? "cursor-not-allowed bg-slate-100 text-slate-400"
                      : "bg-gradient-to-r from-rose-500 to-rose-600 text-white shadow-md hover:shadow-lg hover:shadow-rose-500/25 active:scale-[0.98]"
                  }`}
                >
                  {creating ? (
                    <>
                      <Loader2 className="h-4 w-4 animate-spin" aria-hidden />
                      Préparation...
                    </>
                  ) : (
                    <>
                      Continuer
                      <ArrowRight className="h-4 w-4" aria-hidden />
                    </>
                  )}
                </button>
              </div>
            </div>

            <div className="mt-6">
              <TrustSignals />
            </div>
          </section>
        ) : null}

        {step === "kyc" && connectInstance ? (
          <section className="flex gap-6 animate-fade-in">
            <div className="flex-1 min-w-0">
              <div className="mb-4">
                <h2 className="text-2xl font-extrabold tracking-tight text-slate-900">
                  Vérification de votre identité
                </h2>
                <p className="mt-1.5 text-[14px] text-slate-600">
                  Suivez les étapes ci-dessous. Vous pouvez interrompre et reprendre à tout moment.
                </p>
              </div>

              <div className="overflow-hidden rounded-3xl border border-slate-100 bg-white shadow-sm">
                <div className="p-6 sm:p-8">
                  <ConnectComponentsProvider connectInstance={connectInstance}>
                    <ConnectAccountOnboarding onExit={handleOnboardingExit} />
                  </ConnectComponentsProvider>
                </div>
              </div>
            </div>

            <ContextSidebar />
          </section>
        ) : null}

        {step === "success" ? (
          <section className="mx-auto max-w-3xl animate-slide-up">
            <SuccessState
              accountId={accountStatus?.account_id ?? accountId}
              country={accountStatus?.country ?? country}
              chargesEnabled={accountStatus?.charges_enabled ?? false}
              payoutsEnabled={accountStatus?.payouts_enabled ?? false}
              requirementsCount={accountStatus?.requirements_count ?? 0}
              onRestart={handleRestart}
            />
          </section>
        ) : null}
      </div>
    </main>
  )
}
