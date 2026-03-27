"use client"

import { useState, useEffect } from "react"
import { useSearchParams } from "next/navigation"
import { useRouter } from "@i18n/navigation"
import { Shield, CheckCircle2, Loader2, ArrowLeft } from "lucide-react"
import { useTranslations } from "next-intl"
import { cn, formatCurrency } from "@/shared/lib/utils"
import { getProposal } from "../api/proposal-api"
import { useSimulatePayment } from "../hooks/use-proposals"
import type { ProposalResponse } from "../types"

export function PaymentSimulation() {
  const t = useTranslations("proposal")
  const router = useRouter()
  const searchParams = useSearchParams()
  const proposalId = searchParams.get("proposal") ?? ""

  const [proposal, setProposal] = useState<ProposalResponse | null>(null)
  const [fetchError, setFetchError] = useState(false)
  const [paid, setPaid] = useState(false)

  const payMutation = useSimulatePayment()

  useEffect(() => {
    if (!proposalId) return
    getProposal(proposalId)
      .then(setProposal)
      .catch(() => setFetchError(true))
  }, [proposalId])

  function handlePay() {
    if (!proposalId) return
    payMutation.mutate(proposalId, {
      onSuccess: () => {
        setPaid(true)
        setTimeout(() => router.push("/projects"), 1500)
      },
    })
  }

  if (!proposalId || fetchError) {
    return (
      <div className="flex min-h-[60vh] items-center justify-center">
        <p className="text-sm text-gray-500">{t("proposalNotFound")}</p>
      </div>
    )
  }

  if (!proposal) {
    return (
      <div className="flex min-h-[60vh] items-center justify-center">
        <Loader2 className="h-6 w-6 animate-spin text-rose-500" />
      </div>
    )
  }

  if (paid) {
    return (
      <div className="flex min-h-[60vh] items-center justify-center">
        <div className="text-center space-y-3">
          <CheckCircle2 className="mx-auto h-12 w-12 text-green-500" />
          <p className="text-lg font-semibold text-gray-900 dark:text-white">
            {t("paymentSuccess")}
          </p>
        </div>
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-lg px-4 py-12">
      <button
        type="button"
        onClick={() => router.back()}
        className="mb-6 flex items-center gap-1.5 text-sm text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 transition-colors"
      >
        <ArrowLeft className="h-4 w-4" />
        {t("proposalCancel")}
      </button>

      <div className="rounded-2xl border border-gray-200 bg-white shadow-sm dark:border-gray-700 dark:bg-gray-800/80 overflow-hidden">
        <div className="h-1.5 gradient-primary" />

        <div className="px-6 pt-6 pb-8 space-y-6">
          {/* Header */}
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-rose-100 dark:bg-rose-500/20">
              <Shield className="h-5 w-5 text-rose-600 dark:text-rose-400" strokeWidth={1.5} />
            </div>
            <div>
              <h1 className="text-lg font-bold text-gray-900 dark:text-white">
                {t("paymentSimulation")}
              </h1>
              <p className="text-xs text-gray-500 dark:text-gray-400">
                {t("paymentSimulationDesc")}
              </p>
            </div>
          </div>

          <div className="border-t border-gray-100 dark:border-gray-700" />

          {/* Details */}
          <div className="space-y-4">
            <DetailRow label={t("proposalTitle")} value={proposal.title} />
            <DetailRow
              label={t("totalAmount")}
              value={formatCurrency(proposal.amount / 100)}
              highlight
            />
            {proposal.deadline && (
              <DetailRow
                label={t("proposalDeadline")}
                value={new Intl.DateTimeFormat("fr-FR", {
                  day: "numeric",
                  month: "long",
                  year: "numeric",
                }).format(new Date(proposal.deadline))}
              />
            )}
          </div>

          <div className="border-t border-gray-100 dark:border-gray-700" />

          {/* Pay button */}
          <button
            type="button"
            onClick={handlePay}
            disabled={payMutation.isPending}
            className={cn(
              "w-full flex items-center justify-center gap-2 rounded-xl px-5 py-3",
              "text-sm font-semibold text-white transition-all duration-200",
              "gradient-primary hover:shadow-glow active:scale-[0.98]",
              "disabled:opacity-60 disabled:cursor-not-allowed",
            )}
          >
            {payMutation.isPending ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <Shield className="h-4 w-4" strokeWidth={1.5} />
            )}
            {payMutation.isPending ? t("processing") : t("confirmPayment")}
          </button>

          {payMutation.isError && (
            <p className="text-center text-sm text-red-500">{t("paymentError")}</p>
          )}
        </div>
      </div>
    </div>
  )
}

function DetailRow({ label, value, highlight }: { label: string; value: string; highlight?: boolean }) {
  return (
    <div className="flex items-center justify-between">
      <span className="text-sm text-gray-500 dark:text-gray-400">{label}</span>
      <span
        className={cn(
          "text-sm",
          highlight
            ? "font-bold text-gray-900 dark:text-white"
            : "font-medium text-gray-700 dark:text-gray-300",
        )}
      >
        {value}
      </span>
    </div>
  )
}
