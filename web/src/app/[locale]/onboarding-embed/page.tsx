"use client"

import { useEffect, useState, useCallback } from "react"
import {
  ConnectAccountOnboarding,
  ConnectComponentsProvider,
} from "@stripe/react-connect-js"
import { loadConnectAndInitialize } from "@stripe/connect-js"
import { Loader2 } from "lucide-react"
import { apiClient, API_BASE_URL } from "@/shared/lib/api-client"

/**
 * Standalone onboarding page for mobile WebView.
 * No sidebar/header — minimal layout for embedding.
 * Reads auth token from cookie (set by the mobile app).
 * On completion, sends postMessage to the parent/WebView.
 */
export default function OnboardingEmbedPage() {
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [sessionData, setSessionData] = useState<{
    clientSecret: string
    stripeAccountId: string
    connectInstance: ReturnType<typeof loadConnectAndInitialize>
  } | null>(null)

  useEffect(() => {
    async function init() {
      try {
        const result = await apiClient<{
          client_secret: string
          stripe_account_id: string
        }>("/api/v1/payment-info/account-session", {
          method: "POST",
          body: { email: "" },
        })

        const publishableKey = process.env.NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY
        if (!publishableKey) {
          throw new Error("Stripe publishable key not configured")
        }

        const connectInstance = loadConnectAndInitialize({
          publishableKey,
          fetchClientSecret: async () => result.client_secret,
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

        setSessionData({
          clientSecret: result.client_secret,
          stripeAccountId: result.stripe_account_id,
          connectInstance,
        })
      } catch (err) {
        setError(err instanceof Error ? err.message : "Failed to initialize")
      } finally {
        setLoading(false)
      }
    }

    init()
  }, [])

  const handleExit = useCallback(() => {
    // Notify the mobile WebView that onboarding is complete
    if (typeof window !== "undefined") {
      window.postMessage(
        { type: "onboarding-complete" },
        "*",
      )
    }
  }, [])

  if (loading) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-white">
        <Loader2 className="h-8 w-8 animate-spin text-rose-500" />
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-white p-6">
        <div className="text-center">
          <p className="text-sm text-red-600">{error}</p>
        </div>
      </div>
    )
  }

  if (!sessionData) return null

  return (
    <div className="min-h-screen bg-white p-4">
      <ConnectComponentsProvider connectInstance={sessionData.connectInstance}>
        <ConnectAccountOnboarding onExit={handleExit} />
      </ConnectComponentsProvider>
    </div>
  )
}
