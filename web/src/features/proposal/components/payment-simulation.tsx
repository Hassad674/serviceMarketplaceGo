"use client"

import { useState, useEffect, useCallback } from "react"
import { useSearchParams } from "next/navigation"
import { useRouter } from "@i18n/navigation"
import { Shield, CheckCircle2, Loader2, ArrowLeft, CreditCard } from "lucide-react"
import { useTranslations } from "next-intl"
import { loadStripe } from "@stripe/stripe-js"
import { Elements, PaymentElement, useStripe, useElements } from "@stripe/react-stripe-js"
import { cn, formatCurrency } from "@/shared/lib/utils"
import { getProposal, initiatePayment, confirmPayment } from "../api/proposal-api"
import type { ProposalResponse, PaymentIntentResponse } from "../types"

const stripePromise = process.env.NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY
  ? loadStripe(process.env.NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY)
  : null

export function PaymentSimulation() {
  const t = useTranslations("proposal")
  const router = useRouter()
  const searchParams = useSearchParams()
  const proposalId = searchParams.get("proposal") ?? ""

  const [proposal, setProposal] = useState<ProposalResponse | null>(null)
  const [paymentData, setPaymentData] = useState<PaymentIntentResponse | null>(null)
  const [fetchError, setFetchError] = useState(false)
  const [loading, setLoading] = useState(true)
  const [paid, setPaid] = useState(false)

  useEffect(() => {
    if (!proposalId) return
    let cancelled = false

    getProposal(proposalId)
      .then((p) => {
        if (cancelled) return
        setProposal(p)
        // Only initiate payment if proposal is in accepted state
        if (p.status !== "accepted") {
          setPaid(p.status === "paid" || p.status === "active" || p.status === "completed" || p.status === "completion_requested")
          setLoading(false)
          return
        }
        return initiatePayment(proposalId).then((pd) => {
          if (cancelled) return
          if (pd.status === "paid") {
            setPaid(true)
          } else {
            setPaymentData(pd)
          }
        })
      })
      .catch(() => { if (!cancelled) setFetchError(true) })
      .finally(() => { if (!cancelled) setLoading(false) })

    return () => { cancelled = true }
  }, [proposalId])

  if (!proposalId || fetchError) {
    return <CenteredMessage>{t("proposalNotFound")}</CenteredMessage>
  }

  if (loading) {
    return <CenteredMessage><Loader2 className="h-6 w-6 animate-spin text-rose-500" /></CenteredMessage>
  }

  if (paid) {
    return (
      <CenteredMessage>
        <CheckCircle2 className="mx-auto h-12 w-12 text-green-500" />
        <p className="text-lg font-semibold text-slate-900 dark:text-white">{t("paymentSuccess")}</p>
      </CenteredMessage>
    )
  }

  if (!proposal || !paymentData) {
    return <CenteredMessage>{t("proposalNotFound")}</CenteredMessage>
  }

  // Stripe mode: render Elements
  if (paymentData.client_secret && stripePromise) {
    return (
      <PaymentLayout proposal={proposal} onBack={() => router.back()}>
        <FeeBreakdown amounts={paymentData.amounts} />
        <Elements
          stripe={stripePromise}
          options={{ clientSecret: paymentData.client_secret, appearance: { theme: "stripe" } }}
        >
          <StripePaymentForm proposalId={proposalId} onSuccess={() => {
            setPaid(true)
            setTimeout(() => router.push("/projects"), 2000)
          }} />
        </Elements>
      </PaymentLayout>
    )
  }

  // Simulation fallback (no Stripe key configured)
  return (
    <PaymentLayout proposal={proposal} onBack={() => router.back()}>
      <SimulationFallback proposalId={proposalId} onPaid={() => {
        setPaid(true)
        setTimeout(() => router.push("/projects"), 1500)
      }} />
    </PaymentLayout>
  )
}

function StripePaymentForm({ proposalId, onSuccess }: { proposalId: string; onSuccess: () => void }) {
  const t = useTranslations("proposal")
  const stripe = useStripe()
  const elements = useElements()
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState("")

  const handleSubmit = useCallback(async (e: React.FormEvent) => {
    e.preventDefault()
    if (!stripe || !elements) return

    setSubmitting(true)
    setError("")

    const { error: stripeError } = await stripe.confirmPayment({
      elements,
      confirmParams: {
        return_url: `${window.location.origin}${window.location.pathname}?paid=true`,
      },
      redirect: "if_required",
    })

    if (stripeError) {
      setError(stripeError.message ?? t("paymentError"))
      setSubmitting(false)
      return
    }

    // Payment succeeded — tell backend to activate the proposal
    try {
      await confirmPayment(proposalId)
    } catch {
      // Webhook will handle it as fallback
    }
    onSuccess()
  }, [stripe, elements, proposalId, onSuccess, t])

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <PaymentElement />
      <button
        type="submit"
        disabled={!stripe || submitting}
        className={cn(
          "w-full flex items-center justify-center gap-2 rounded-xl px-5 py-3",
          "text-sm font-semibold text-white transition-all duration-200",
          "gradient-primary hover:shadow-glow active:scale-[0.98]",
          "disabled:opacity-60 disabled:cursor-not-allowed",
        )}
      >
        {submitting ? (
          <Loader2 className="h-4 w-4 animate-spin" />
        ) : (
          <CreditCard className="h-4 w-4" strokeWidth={1.5} />
        )}
        {submitting ? t("processing") : t("confirmPayment")}
      </button>
      {error && <p className="text-center text-sm text-red-500">{error}</p>}
    </form>
  )
}

function FeeBreakdown({ amounts }: { amounts?: PaymentIntentResponse["amounts"] }) {
  const t = useTranslations("proposal")
  if (!amounts) return null

  return (
    <div className="space-y-2 rounded-xl bg-slate-50 dark:bg-slate-700/30 p-4">
      <DetailRow label={t("totalAmount")} value={formatCurrency(amounts.proposal_amount / 100)} />
      <DetailRow label={t("stripeFees")} value={formatCurrency(amounts.stripe_fee / 100)} />
      <div className="border-t border-slate-200 dark:border-slate-600 pt-2">
        <DetailRow label={t("totalToPay")} value={formatCurrency(amounts.client_total / 100)} highlight />
      </div>
    </div>
  )
}

function SimulationFallback({ proposalId, onPaid }: { proposalId: string; onPaid: () => void }) {
  const t = useTranslations("proposal")
  const [submitting, setSubmitting] = useState(false)

  function handlePay() {
    setSubmitting(true)
    initiatePayment(proposalId)
      .then(() => onPaid())
      .catch(() => setSubmitting(false))
  }

  return (
    <>
      <p className="text-xs text-amber-600 dark:text-amber-400 text-center">
        {t("paymentSimulationDesc")}
      </p>
      <button
        type="button"
        onClick={handlePay}
        disabled={submitting}
        className={cn(
          "w-full flex items-center justify-center gap-2 rounded-xl px-5 py-3",
          "text-sm font-semibold text-white transition-all duration-200",
          "gradient-primary hover:shadow-glow active:scale-[0.98]",
          "disabled:opacity-60 disabled:cursor-not-allowed",
        )}
      >
        {submitting ? <Loader2 className="h-4 w-4 animate-spin" /> : <Shield className="h-4 w-4" strokeWidth={1.5} />}
        {submitting ? t("processing") : t("confirmPayment")}
      </button>
    </>
  )
}

function PaymentLayout({ proposal, onBack, children }: { proposal: ProposalResponse; onBack: () => void; children: React.ReactNode }) {
  const t = useTranslations("proposal")
  return (
    <div className="mx-auto max-w-lg px-4 py-12">
      <button type="button" onClick={onBack} className="mb-6 flex items-center gap-1.5 text-sm text-slate-500 hover:text-slate-700 dark:text-slate-400 dark:hover:text-slate-200 transition-colors">
        <ArrowLeft className="h-4 w-4" />
        {t("proposalCancel")}
      </button>
      <div className="rounded-2xl border border-slate-200 bg-white shadow-sm dark:border-slate-700 dark:bg-slate-800/80 overflow-hidden">
        <div className="h-1.5 gradient-primary" />
        <div className="px-6 pt-6 pb-8 space-y-6">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-rose-100 dark:bg-rose-500/20">
              <CreditCard className="h-5 w-5 text-rose-600 dark:text-rose-400" strokeWidth={1.5} />
            </div>
            <div>
              <h1 className="text-lg font-bold text-slate-900 dark:text-white">{t("payment")}</h1>
              <p className="text-xs text-slate-500 dark:text-slate-400">{proposal.title}</p>
            </div>
          </div>
          <div className="border-t border-slate-100 dark:border-slate-700" />
          {children}
        </div>
      </div>
    </div>
  )
}

function DetailRow({ label, value, highlight }: { label: string; value: string; highlight?: boolean }) {
  return (
    <div className="flex items-center justify-between">
      <span className="text-sm text-slate-500 dark:text-slate-400">{label}</span>
      <span className={cn("text-sm", highlight ? "font-bold text-slate-900 dark:text-white" : "font-medium text-slate-700 dark:text-slate-300")}>{value}</span>
    </div>
  )
}

function CenteredMessage({ children }: { children: React.ReactNode }) {
  return <div className="flex min-h-[60vh] items-center justify-center"><div className="text-center space-y-3">{children}</div></div>
}
