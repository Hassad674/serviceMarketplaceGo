"use client"

import { useState, useEffect, useCallback } from "react"
import { useSearchParams } from "next/navigation"
import { useRouter } from "@i18n/navigation"
import {
  Shield,
  CheckCircle2,
  Loader2,
  ArrowLeft,
  CreditCard,
  AlertTriangle,
  RefreshCw,
} from "lucide-react"
import { useTranslations } from "next-intl"
import { loadStripe } from "@stripe/stripe-js"
import { Elements, PaymentElement, useStripe, useElements } from "@stripe/react-stripe-js"
import { trackPurchase } from "@/shared/lib/analytics-events"
import { ApiError } from "@/shared/lib/api-client"
import { cn, formatCurrency } from "@/shared/lib/utils"
import { getProposal, initiatePayment, confirmPayment } from "../api/proposal-api"
import type { ProposalResponse, PaymentIntentResponse } from "../types"
import { Button } from "@/shared/components/ui/button"
import { BillingProfileInlineModal } from "@/shared/components/billing-profile/billing-profile-inline-modal"
import { BillingProfileEmbed } from "@/shared/components/billing-profile/billing-profile-embed"
import { useBillingProfile } from "@/shared/hooks/billing-profile/use-billing-profile"

// Soleil v2 — Payment confirmation page (escrow). Soleil card with
// editorial header, Geist Mono amount summary, corail "Confirmer" pill.

const stripePromise = process.env.NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY
  ? loadStripe(process.env.NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY)
  : null

type PaymentErrorKind = "not_found" | "init_failed" | "billing_incomplete" | null

// classifyPaymentError maps an InitiatePayment failure onto the
// rendered error states. The 412 billing-profile gate is the sole
// non-failure surface — every other 4xx/5xx falls back to the generic
// "init_failed" retry view. Kept as a top-level pure function so the
// effect body stays readable and the mapping is unit-testable in
// isolation.
function classifyPaymentError(err: unknown): PaymentErrorKind {
  if (err instanceof ApiError) {
    if (err.status === 404) return "not_found"
    // 412 Precondition Required + the canonical
    // billing_profile_incomplete code → the client org has not yet
    // filled in its billing identity. The defensive branch (race
    // condition with a sibling tab) lands here too.
    if (err.status === 412 && err.code === "billing_profile_incomplete") {
      return "billing_incomplete"
    }
  }
  return "init_failed"
}

export function PaymentSimulation() {
  const t = useTranslations("proposal")
  const router = useRouter()
  const searchParams = useSearchParams()
  const proposalId = searchParams.get("proposal") ?? ""

  const [proposal, setProposal] = useState<ProposalResponse | null>(null)
  const [paymentData, setPaymentData] = useState<PaymentIntentResponse | null>(null)
  const [errorKind, setErrorKind] = useState<PaymentErrorKind>(null)
  const [loading, setLoading] = useState(true)
  const [paid, setPaid] = useState(false)
  const [retryNonce, setRetryNonce] = useState(0)
  // billingModalOpen is the visibility flag for the inline form modal.
  // It opens when the backend rejects InitiatePayment with 412 and
  // closes either when the user dismisses or when the form's onSaved
  // callback retries the payment intent.
  const [billingModalOpen, setBillingModalOpen] = useState(false)

  // Inline billing-profile embed state owned at the top of the flow so
  // both branches (Stripe Elements + simulation fallback) can render
  // and gate on it. `null` means "follow snapshot.is_complete" — true
  // forces edit mode, false forces summary mode.
  const { data: billingSnapshot } = useBillingProfile()
  const isBillingComplete = Boolean(billingSnapshot?.is_complete)
  const [isEditingBilling, setIsEditingBilling] = useState<boolean | null>(null)
  const billingMode: "summary" | "form" = (() => {
    if (isEditingBilling === true) return "form"
    if (isEditingBilling === false) return "summary"
    return isBillingComplete ? "summary" : "form"
  })()
  // Stripe Payment Element + Confirm button only render once the
  // billing profile is complete AND the user is not actively editing.
  const isPaymentReady = isBillingComplete && billingMode === "summary"

  useEffect(() => {
    if (!proposalId) return
    let cancelled = false

    setLoading(true)
    setErrorKind(null)

    getProposal(proposalId)
      .then((p) => {
        if (cancelled) return
        setProposal(p)
        const currentMs = p.milestones?.find(
          (m) => m.sequence === p.current_milestone_sequence,
        )
        const validAccepted = p.status === "accepted"
        const validActivePending =
          p.status === "active" && currentMs?.status === "pending_funding"
        if (!validAccepted && !validActivePending) {
          router.replace(`/projects/${proposalId}`)
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
      .catch((err) => {
        if (cancelled) return
        const kind = classifyPaymentError(err)
        setErrorKind(kind)
        if (kind === "billing_incomplete") {
          setBillingModalOpen(true)
        }
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })

    return () => {
      cancelled = true
    }
  }, [proposalId, router, retryNonce])

  if (!proposalId || errorKind === "not_found") {
    return <CenteredMessage>{t("proposalNotFound")}</CenteredMessage>
  }

  if (errorKind === "billing_incomplete") {
    // Render the inline modal next to a static "fix your billing"
    // page placeholder. When the user saves, retryNonce bumps and
    // the effect re-runs initiatePayment with the same proposal id.
    return (
      <>
        <CenteredMessage>
          <AlertTriangle
            className="mx-auto h-12 w-12 text-warning"
            strokeWidth={1.5}
            aria-hidden="true"
          />
          <p className="font-serif text-[18px] font-medium text-foreground">
            {t("billingIncompleteTitle")}
          </p>
          <p className="text-[13.5px] text-muted-foreground">
            {t("billingIncompleteHint")}
          </p>
          <Button
            variant="ghost"
            size="auto"
            type="button"
            onClick={() => setBillingModalOpen(true)}
            className={cn(
              "mt-4 inline-flex items-center gap-2 rounded-full px-5 py-2.5",
              "text-[13.5px] font-bold text-primary-foreground transition-all duration-200 ease-out",
              "bg-primary hover:bg-primary-deep hover:shadow-[0_4px_14px_rgba(232,93,74,0.28)] active:scale-[0.98]",
            )}
          >
            {t("billingIncompleteCta")}
          </Button>
        </CenteredMessage>
        <BillingProfileInlineModal
          open={billingModalOpen}
          onClose={() => setBillingModalOpen(false)}
          onSaved={() => {
            // Close the modal and retry the payment-intent creation.
            // setRetryNonce flips a state value referenced by the
            // effect dependency array, so the entire load-then-init
            // pipeline reruns from scratch.
            setBillingModalOpen(false)
            setErrorKind(null)
            setRetryNonce((n) => n + 1)
          }}
        />
      </>
    )
  }

  if (errorKind === "init_failed") {
    return (
      <CenteredMessage>
        <AlertTriangle
          className="mx-auto h-12 w-12 text-warning"
          strokeWidth={1.5}
          aria-hidden="true"
        />
        <p className="font-serif text-[18px] font-medium text-foreground">
          {t("paymentInitFailed")}
        </p>
        <p className="text-[13.5px] text-muted-foreground">
          {t("paymentInitFailedHint")}
        </p>
        <Button
          variant="ghost"
          size="auto"
          type="button"
          onClick={() => setRetryNonce((n) => n + 1)}
          className={cn(
            "mt-4 inline-flex items-center gap-2 rounded-full px-5 py-2.5",
            "text-[13.5px] font-bold text-primary-foreground transition-all duration-200 ease-out",
            "bg-primary hover:bg-primary-deep hover:shadow-[0_4px_14px_rgba(232,93,74,0.28)] active:scale-[0.98]",
          )}
        >
          <RefreshCw className="h-4 w-4" strokeWidth={1.7} aria-hidden="true" />
          {t("paymentRetry")}
        </Button>
      </CenteredMessage>
    )
  }

  if (loading) {
    return (
      <CenteredMessage>
        <Loader2 className="h-6 w-6 animate-spin text-primary" aria-hidden="true" />
      </CenteredMessage>
    )
  }

  if (paid) {
    return (
      <CenteredMessage>
        <CheckCircle2
          className="mx-auto h-12 w-12 text-success"
          strokeWidth={1.5}
          aria-hidden="true"
        />
        <p className="font-serif text-[18px] font-medium text-foreground">
          {t("paymentSuccess")}
        </p>
      </CenteredMessage>
    )
  }

  if (!proposal || !paymentData) {
    return <CenteredMessage>{t("proposalNotFound")}</CenteredMessage>
  }

  const billingEmbedBlock = (
    <BillingProfileEmbed
      mode={billingMode}
      onEdit={() => setIsEditingBilling(true)}
      onSaved={() => setIsEditingBilling(false)}
    />
  )

  if (paymentData.client_secret && stripePromise) {
    return (
      <PaymentLayout proposal={proposal} onBack={() => router.back()}>
        <FeeBreakdown amounts={paymentData.amounts} />
        {billingEmbedBlock}
        {isPaymentReady && (
          <Elements
            stripe={stripePromise}
            options={{
              clientSecret: paymentData.client_secret,
              appearance: { theme: "stripe" },
            }}
          >
            <StripePaymentForm
              proposalId={proposalId}
              onSuccess={() => {
                // GA4 + PostHog: ecommerce schema. Amounts arrive in
                // cents from the backend → divide by 100 for the
                // currency-typed `value`. The marketplace ships EUR
                // only; revisit when multi-currency lands. transaction_id
                // uses the backend-issued payment_record_id when
                // present, falling back to the proposal id which is
                // still unique per purchase.
                const amounts = paymentData.amounts
                if (amounts) {
                  trackPurchase({
                    value: amounts.client_total / 100,
                    currency: "EUR",
                    transactionId:
                      paymentData.payment_record_id ?? proposalId,
                    items: [
                      {
                        item_id: proposalId,
                        item_name: proposal.title,
                        price: amounts.proposal_amount / 100,
                        quantity: 1,
                      },
                    ],
                  })
                }
                setPaid(true)
                setTimeout(() => router.push("/projects"), 2000)
              }}
            />
          </Elements>
        )}
        {/* Closing comment for downstream readers: the legacy 412 gate
            modal stays mounted here as a defensive fallback. The pre-
            payment gate was removed in fcb072d5; if a future deploy
            ever re-introduces it, the modal renders without breaking
            the page. */}
      </PaymentLayout>
    )
  }

  return (
    <PaymentLayout proposal={proposal} onBack={() => router.back()}>
      {billingEmbedBlock}
      {isPaymentReady && (
        <SimulationFallback
          proposalId={proposalId}
          onPaid={() => {
            // Simulation mode (no Stripe): still fire the GA4 purchase
            // so the conversion funnel registers the test transactions
            // in dev / preview deployments.
            if (paymentData.amounts) {
              trackPurchase({
                value: paymentData.amounts.client_total / 100,
                currency: "EUR",
                transactionId: paymentData.payment_record_id ?? proposalId,
                items: [
                  {
                    item_id: proposalId,
                    item_name: proposal.title,
                    price: paymentData.amounts.proposal_amount / 100,
                    quantity: 1,
                  },
                ],
              })
            }
            setPaid(true)
            setTimeout(() => router.push("/projects"), 1500)
          }}
        />
      )}
    </PaymentLayout>
  )
}

function StripePaymentForm({
  proposalId,
  onSuccess,
}: {
  proposalId: string
  onSuccess: () => void
}) {
  const t = useTranslations("proposal")
  const stripe = useStripe()
  const elements = useElements()
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState("")

  const handleSubmit = useCallback(
    async (e: React.FormEvent) => {
      e.preventDefault()
      if (!stripe || !elements) return

      setSubmitting(true)
      setError("")

      // Hand off to Stripe. The Payment Element collects address +
      // name natively per the Elements options.fields.billingDetails
      // config — those land on payment_method.billing_details and are
      // projected by the webhook adapter into PaymentBillingDetails.
      // The receipt-identity fields (legal_name, SIRET, VAT) are
      // already on file via the BillingProfileEmbed rendered above
      // this form — the parent gates rendering on `is_complete` so
      // we can ship a clean submit path here.
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

      try {
        await confirmPayment(proposalId)
      } catch {
        // Webhook will handle it as fallback
      }
      onSuccess()
    },
    [stripe, elements, proposalId, onSuccess, t],
  )

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <PaymentElement
        options={{
          fields: {
            billingDetails: {
              name: "auto",
              address: {
                line1: "auto",
                line2: "auto",
                city: "auto",
                postalCode: "auto",
                country: "auto",
              },
            },
          },
        }}
      />
      <Button
        variant="ghost"
        size="auto"
        type="submit"
        disabled={!stripe || submitting}
        className={cn(
          "flex w-full items-center justify-center gap-2 rounded-full px-5 py-3",
          "text-[13.5px] font-bold text-primary-foreground transition-all duration-200 ease-out",
          "bg-primary hover:bg-primary-deep hover:shadow-[0_4px_14px_rgba(232,93,74,0.28)] active:scale-[0.98]",
          "disabled:cursor-not-allowed disabled:opacity-60",
        )}
      >
        {submitting ? (
          <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
        ) : (
          <CreditCard className="h-4 w-4" strokeWidth={1.7} aria-hidden="true" />
        )}
        {submitting ? t("processing") : t("confirmPayment")}
      </Button>
      {error && (
        <p role="alert" className="text-center text-[13px] text-destructive">
          {error}
        </p>
      )}
    </form>
  )
}

function FeeBreakdown({ amounts }: { amounts?: PaymentIntentResponse["amounts"] }) {
  const t = useTranslations("proposal")
  if (!amounts) return null

  return (
    <div className="space-y-2 rounded-2xl border border-border bg-background p-4">
      <DetailRow
        label={t("totalAmount")}
        value={formatCurrency(amounts.proposal_amount / 100)}
      />
      <DetailRow
        label={t("stripeFees")}
        value={formatCurrency(amounts.stripe_fee / 100)}
      />
      <div className="border-t border-dashed border-border pt-2">
        <DetailRow
          label={t("totalToPay")}
          value={formatCurrency(amounts.client_total / 100)}
          highlight
        />
      </div>
    </div>
  )
}

function SimulationFallback({
  proposalId,
  onPaid,
}: {
  proposalId: string
  onPaid: () => void
}) {
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
      <p className="text-center text-[12px] text-warning">
        {t("paymentSimulationDesc")}
      </p>
      <Button
        variant="ghost"
        size="auto"
        type="button"
        onClick={handlePay}
        disabled={submitting}
        className={cn(
          "flex w-full items-center justify-center gap-2 rounded-full px-5 py-3",
          "text-[13.5px] font-bold text-primary-foreground transition-all duration-200 ease-out",
          "bg-primary hover:bg-primary-deep hover:shadow-[0_4px_14px_rgba(232,93,74,0.28)] active:scale-[0.98]",
          "disabled:cursor-not-allowed disabled:opacity-60",
        )}
      >
        {submitting ? (
          <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
        ) : (
          <Shield className="h-4 w-4" strokeWidth={1.7} aria-hidden="true" />
        )}
        {submitting ? t("processing") : t("confirmPayment")}
      </Button>
    </>
  )
}

function PaymentLayout({
  proposal,
  onBack,
  children,
}: {
  proposal: ProposalResponse
  onBack: () => void
  children: React.ReactNode
}) {
  const t = useTranslations("proposal")
  return (
    <div className="mx-auto max-w-lg px-4 py-12">
      <Button
        variant="ghost"
        size="auto"
        type="button"
        onClick={onBack}
        className={cn(
          "mb-6 inline-flex items-center gap-1.5 px-2 py-1 text-[13px] font-medium",
          "text-muted-foreground hover:text-primary transition-colors duration-150",
        )}
      >
        <ArrowLeft className="h-4 w-4" strokeWidth={1.7} aria-hidden="true" />
        {t("proposalCancel")}
      </Button>

      {/* Editorial header */}
      <div className="mb-6 space-y-2">
        <p className="font-mono text-[11px] font-bold uppercase tracking-[0.12em] text-primary">
          {t("proposalFlow_pay_eyebrow")}
        </p>
        <h1 className="font-serif text-[28px] font-medium leading-[1.05] tracking-[-0.02em] text-foreground sm:text-[32px]">
          {t("proposalFlow_pay_titlePrefix")}{" "}
          <span className="italic text-primary">
            {t("proposalFlow_pay_titleAccent")}
          </span>
        </h1>
        <p className="text-[14.5px] leading-relaxed text-muted-foreground">
          {t("proposalFlow_pay_subtitle")}
        </p>
      </div>

      <div
        className="overflow-hidden rounded-2xl border border-border bg-card"
        style={{ boxShadow: "var(--shadow-card)" }}
      >
        <div className="space-y-5 px-6 pb-8 pt-6">
          <div className="flex items-center gap-3">
            <div className="flex h-11 w-11 shrink-0 items-center justify-center rounded-full bg-primary-soft text-primary">
              <CreditCard className="h-5 w-5" strokeWidth={1.7} aria-hidden="true" />
            </div>
            <div className="min-w-0">
              <p className="font-mono text-[10.5px] font-bold uppercase tracking-[0.1em] text-primary">
                {t("proposalFlow_pay_summary")}
              </p>
              <p className="truncate text-[14.5px] font-semibold text-foreground">
                {proposal.title}
              </p>
            </div>
          </div>
          <div className="border-t border-dashed border-border" />
          {children}
          <p className="text-center font-mono text-[11px] font-medium text-subtle-foreground">
            {t("proposalFlow_pay_secureNotice")}
          </p>
        </div>
      </div>
    </div>
  )
}

function DetailRow({
  label,
  value,
  highlight,
}: {
  label: string
  value: string
  highlight?: boolean
}) {
  return (
    <div className="flex items-center justify-between">
      <span className="text-[13px] text-muted-foreground">{label}</span>
      <span
        className={cn(
          "font-mono text-[14px]",
          highlight ? "font-bold text-foreground" : "font-medium text-foreground",
        )}
      >
        {value}
      </span>
    </div>
  )
}

function CenteredMessage({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex min-h-[60vh] items-center justify-center px-4">
      <div className="space-y-3 text-center">{children}</div>
    </div>
  )
}
