"use client"

import { useState, useCallback } from "react"
import {
  ConnectAccountOnboarding,
  ConnectComponentsProvider,
} from "@stripe/react-connect-js"
import { loadConnectAndInitialize } from "@stripe/connect-js"
import { Loader2, AlertTriangle } from "lucide-react"
import { useTranslations } from "next-intl"

interface ConnectOnboardingProps {
  clientSecret: string
  stripeAccountId: string
  onComplete: () => void
}

function createStripeConnectInstance(clientSecret: string) {
  const publishableKey = process.env.NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY
  if (!publishableKey) {
    throw new Error("NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY is not set")
  }

  return loadConnectAndInitialize({
    publishableKey,
    fetchClientSecret: async () => clientSecret,
    appearance: {
      overlays: "dialog",
      variables: {
        colorPrimary: "#F43F5E",
        colorBackground: "#FFFFFF",
        colorText: "#0F172A",
        colorDanger: "#EF4444",
        borderRadius: "12px",
        fontFamily: "inherit",
        colorSecondaryText: "#64748B",
        actionPrimaryColorText: "#FFFFFF",
        formHighlightColorBorder: "#F43F5E",
      },
    },
  })
}

export function ConnectOnboarding({
  clientSecret,
  stripeAccountId,
  onComplete,
}: ConnectOnboardingProps) {
  const t = useTranslations("paymentInfo")
  const [error, setError] = useState<string | null>(null)
  const [stripeInstance] = useState(() => createStripeConnectInstance(clientSecret))

  const handleExit = useCallback(() => {
    onComplete()
  }, [onComplete])

  if (error) {
    return (
      <div className="flex items-center gap-3 rounded-xl border border-red-200 bg-red-50 p-4 dark:border-red-500/30 dark:bg-red-500/10">
        <AlertTriangle className="h-5 w-5 shrink-0 text-red-600 dark:text-red-400" strokeWidth={1.5} />
        <p className="text-sm font-medium text-red-700 dark:text-red-300">{error}</p>
      </div>
    )
  }

  return (
    <div className="rounded-2xl border border-slate-100 bg-white p-6 shadow-sm dark:border-slate-700 dark:bg-slate-800">
      <p className="mb-4 text-xs text-gray-500 dark:text-gray-400">
        {t("stripeAccountId")}: {stripeAccountId}
      </p>
      <ConnectComponentsProvider connectInstance={stripeInstance}>
        <ConnectAccountOnboarding
          onExit={handleExit}
        />
      </ConnectComponentsProvider>
    </div>
  )
}
