"use client"

import { useState, useCallback } from "react"
import { CheckCircle, Loader2, CreditCard, Shield } from "lucide-react"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import { ConnectOnboarding } from "./connect-onboarding"
import { useUser } from "@/shared/hooks/use-user"
import {
  usePaymentInfo,
  useCreateAccountSession,
  useInvalidatePaymentInfo,
} from "../hooks/use-payment-info"

export function PaymentInfoPage() {
  const t = useTranslations("paymentInfo")
  const { data: user } = useUser()
  const { data: existing, isLoading } = usePaymentInfo()
  const createSession = useCreateAccountSession()
  const invalidate = useInvalidatePaymentInfo()

  const [sessionData, setSessionData] = useState<{
    clientSecret: string
    stripeAccountId: string
  } | null>(null)

  const handleSetupPayments = useCallback(() => {
    if (!user?.email) return
    createSession.mutate(user.email, {
      onSuccess: (result) => {
        setSessionData({
          clientSecret: result.client_secret,
          stripeAccountId: result.stripe_account_id,
        })
      },
    })
  }, [user, createSession])

  const handleRefreshSession = useCallback(() => {
    if (!user?.email) return
    createSession.mutate(user.email, {
      onSuccess: (result) => {
        setSessionData({
          clientSecret: result.client_secret,
          stripeAccountId: result.stripe_account_id,
        })
      },
    })
  }, [user, createSession])

  const handleOnboardingComplete = useCallback(() => {
    setSessionData(null)
    invalidate()
  }, [invalidate])

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-24">
        <Loader2 className="h-6 w-6 animate-spin text-rose-500" />
      </div>
    )
  }

  const isVerified =
    existing?.stripe_verified === true ||
    (existing?.stripe_account_id && existing?.stripe_account_id !== "")

  const hasAccount = existing?.stripe_account_id && existing.stripe_account_id !== ""

  return (
    <div className="space-y-6">
      {/* Page header */}
      <div>
        <h1 className="text-2xl font-bold tracking-tight text-gray-900 dark:text-white">
          {t("title")}
        </h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
          {t("subtitle")}
        </p>
      </div>

      {/* Verified account status */}
      {isVerified && existing?.stripe_verified && (
        <AccountVerifiedCard stripeAccountId={existing.stripe_account_id} />
      )}

      {/* Account exists but not fully verified — show onboarding */}
      {hasAccount && !existing?.stripe_verified && !sessionData && (
        <AccountPendingCard onContinue={handleRefreshSession} isLoading={createSession.isPending} />
      )}

      {/* No account — show CTA */}
      {!hasAccount && !sessionData && (
        <SetupPaymentsCard onSetup={handleSetupPayments} isLoading={createSession.isPending} />
      )}

      {/* Error state */}
      {createSession.isError && (
        <div className="flex items-center gap-3 rounded-xl border border-red-200 bg-red-50 p-4 dark:border-red-500/30 dark:bg-red-500/10">
          <p className="text-sm font-medium text-red-700 dark:text-red-300">
            {createSession.error instanceof Error
              ? createSession.error.message
              : t("saveError")}
          </p>
        </div>
      )}

      {/* Embedded onboarding component */}
      {sessionData && (
        <ConnectOnboarding
          clientSecret={sessionData.clientSecret}
          stripeAccountId={sessionData.stripeAccountId}
          onComplete={handleOnboardingComplete}
        />
      )}
    </div>
  )
}

interface AccountVerifiedCardProps {
  stripeAccountId: string
}

function AccountVerifiedCard({ stripeAccountId }: AccountVerifiedCardProps) {
  const t = useTranslations("paymentInfo")

  return (
    <div className="rounded-2xl border border-emerald-200 bg-emerald-50 p-6 dark:border-emerald-500/30 dark:bg-emerald-500/10">
      <div className="flex items-start gap-4">
        <div className="flex h-12 w-12 items-center justify-center rounded-full bg-emerald-100 dark:bg-emerald-500/20">
          <CheckCircle className="h-6 w-6 text-emerald-600 dark:text-emerald-400" />
        </div>
        <div className="flex-1">
          <h3 className="text-lg font-semibold text-emerald-800 dark:text-emerald-200">
            {t("accountVerified")}
          </h3>
          <p className="mt-1 text-sm text-emerald-700 dark:text-emerald-300">
            {t("accountVerifiedDescription")}
          </p>
          <p className="mt-2 font-mono text-xs text-emerald-600 dark:text-emerald-400">
            {stripeAccountId}
          </p>
        </div>
      </div>
    </div>
  )
}

interface AccountPendingCardProps {
  onContinue: () => void
  isLoading: boolean
}

function AccountPendingCard({ onContinue, isLoading }: AccountPendingCardProps) {
  const t = useTranslations("paymentInfo")

  return (
    <div className="rounded-2xl border border-amber-200 bg-amber-50 p-6 dark:border-amber-500/30 dark:bg-amber-500/10">
      <div className="flex items-start gap-4">
        <div className="flex h-12 w-12 items-center justify-center rounded-full bg-amber-100 dark:bg-amber-500/20">
          <Shield className="h-6 w-6 text-amber-600 dark:text-amber-400" />
        </div>
        <div className="flex-1">
          <h3 className="text-lg font-semibold text-amber-800 dark:text-amber-200">
            {t("accountPending")}
          </h3>
          <p className="mt-1 text-sm text-amber-700 dark:text-amber-300">
            {t("accountPendingDescription")}
          </p>
          <button
            type="button"
            onClick={onContinue}
            disabled={isLoading}
            className={cn(
              "mt-4 rounded-xl px-6 py-2.5 text-sm font-semibold text-white transition-all duration-200",
              isLoading
                ? "cursor-not-allowed bg-gray-300 dark:bg-gray-700"
                : "gradient-primary hover:shadow-glow active:scale-[0.98]",
            )}
          >
            {isLoading ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              t("continueOnboarding")
            )}
          </button>
        </div>
      </div>
    </div>
  )
}

interface SetupPaymentsCardProps {
  onSetup: () => void
  isLoading: boolean
}

function SetupPaymentsCard({ onSetup, isLoading }: SetupPaymentsCardProps) {
  const t = useTranslations("paymentInfo")

  return (
    <div className="rounded-2xl border border-slate-100 bg-white p-8 shadow-sm dark:border-slate-700 dark:bg-slate-800">
      <div className="flex flex-col items-center text-center">
        <div className="flex h-16 w-16 items-center justify-center rounded-full bg-rose-50 dark:bg-rose-500/10">
          <CreditCard className="h-8 w-8 text-rose-500" />
        </div>
        <h3 className="mt-4 text-lg font-semibold text-gray-900 dark:text-white">
          {t("setupPaymentsTitle")}
        </h3>
        <p className="mt-2 max-w-md text-sm text-gray-500 dark:text-gray-400">
          {t("setupPaymentsDescription")}
        </p>
        <button
          type="button"
          onClick={onSetup}
          disabled={isLoading}
          className={cn(
            "mt-6 rounded-xl px-8 py-3 text-sm font-semibold text-white transition-all duration-200",
            isLoading
              ? "cursor-not-allowed bg-gray-300 dark:bg-gray-700"
              : "gradient-primary hover:shadow-glow active:scale-[0.98]",
          )}
        >
          {isLoading ? (
            <Loader2 className="h-4 w-4 animate-spin" />
          ) : (
            t("setupPaymentsButton")
          )}
        </button>
      </div>
    </div>
  )
}
