"use client"

import { Scale, CheckCircle2, XCircle, Calendar } from "lucide-react"
import { useTranslations } from "next-intl"

import type { DisputeResponse } from "../types"

interface DisputeResolutionCardProps {
  dispute: DisputeResponse
  currentUserId: string
}

// DisputeResolutionCard renders the historical decision of a dispute
// (resolved or cancelled) on the project detail page, AFTER the dispute
// banner has gone away because the proposal was restored to active or
// completed. Lets the parties always see what happened and on what terms.
export function DisputeResolutionCard({ dispute, currentUserId }: DisputeResolutionCardProps) {
  const t = useTranslations("disputes")

  if (dispute.status === "resolved") {
    return <ResolvedCard dispute={dispute} currentUserId={currentUserId} t={t} />
  }
  if (dispute.status === "cancelled") {
    return <CancelledCard dispute={dispute} t={t} />
  }
  return null
}

interface ResolvedCardProps {
  dispute: DisputeResponse
  currentUserId: string
  t: ReturnType<typeof useTranslations>
}

function ResolvedCard({ dispute, currentUserId, t }: ResolvedCardProps) {
  const clientAmount = dispute.resolution_amount_client ?? 0
  const providerAmount = dispute.resolution_amount_provider ?? 0
  const total = clientAmount + providerAmount
  const clientPct = total > 0 ? Math.round((clientAmount / total) * 100) : 0
  const providerPct = 100 - clientPct

  // Highlight the user's own share so they see at a glance what they got.
  const isClient = currentUserId === dispute.client_id
  const myAmount = isClient ? clientAmount : providerAmount
  const myPct = isClient ? clientPct : providerPct

  return (
    <div
      role="status"
      className="mb-4 rounded-xl border border-emerald-200 bg-emerald-50/60 p-4 dark:border-emerald-500/30 dark:bg-emerald-500/10 animate-slide-up"
    >
      <div className="flex items-start gap-3">
        <div className="mt-0.5 flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-emerald-100 dark:bg-emerald-500/20">
          <Scale className="h-4 w-4 text-emerald-700 dark:text-emerald-300" aria-hidden />
        </div>
        <div className="flex-1 min-w-0">
          <p className="text-sm font-semibold text-emerald-900 dark:text-emerald-200">
            {t("decisionTitle")}
          </p>
          <p className="mt-0.5 text-xs text-emerald-800/80 dark:text-emerald-200/80">
            {t("decisionYourShare", {
              percent: myPct,
              amount: formatEur(myAmount),
            })}
          </p>

          <div className="mt-3 grid grid-cols-2 gap-2 rounded-lg bg-white/60 p-3 text-sm dark:bg-slate-800/40">
            <SplitCell
              label={t("client")}
              amount={clientAmount}
              percent={clientPct}
              highlighted={isClient}
            />
            <SplitCell
              label={t("provider")}
              amount={providerAmount}
              percent={providerPct}
              highlighted={!isClient}
            />
          </div>

          {dispute.resolution_note && (
            <div className="mt-3 rounded-lg bg-white/60 p-3 text-sm dark:bg-slate-800/40">
              <p className="mb-1 text-xs font-medium text-emerald-900 dark:text-emerald-200">
                {t("decisionMessage")}
              </p>
              <p className="whitespace-pre-wrap text-slate-700 dark:text-slate-300">
                {dispute.resolution_note}
              </p>
            </div>
          )}

          {dispute.resolved_at && (
            <p className="mt-3 flex items-center gap-1 text-xs text-emerald-700/80 dark:text-emerald-300/80">
              <Calendar className="h-3 w-3" aria-hidden />
              {t("decisionRenderedOn", { date: formatDate(dispute.resolved_at) })}
            </p>
          )}
        </div>
      </div>
    </div>
  )
}

interface CancelledCardProps {
  dispute: DisputeResponse
  t: ReturnType<typeof useTranslations>
}

function CancelledCard({ dispute, t }: CancelledCardProps) {
  return (
    <div
      role="status"
      className="mb-4 rounded-xl border border-slate-200 bg-slate-50/60 p-4 dark:border-slate-700 dark:bg-slate-800/40 animate-slide-up"
    >
      <div className="flex items-start gap-3">
        <div className="mt-0.5 flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-slate-100 dark:bg-slate-700/40">
          <XCircle className="h-4 w-4 text-slate-600 dark:text-slate-300" aria-hidden />
        </div>
        <div className="flex-1 min-w-0">
          <p className="text-sm font-semibold text-slate-900 dark:text-slate-200">
            {t("disputeCancelledTitle")}
          </p>
          <p className="mt-0.5 text-xs text-slate-600 dark:text-slate-400">
            {t("disputeCancelledSubtitle")}
          </p>
          {dispute.resolved_at && (
            <p className="mt-2 text-xs text-slate-500">
              {formatDate(dispute.resolved_at)}
            </p>
          )}
        </div>
      </div>
    </div>
  )
}

interface SplitCellProps {
  label: string
  amount: number
  percent: number
  highlighted: boolean
}

function SplitCell({ label, amount, percent, highlighted }: SplitCellProps) {
  return (
    <div
      className={
        highlighted
          ? "rounded-md border border-emerald-300 bg-white p-2 dark:border-emerald-500/40 dark:bg-slate-800"
          : "p-2"
      }
    >
      <p className="flex items-center gap-1 text-xs text-slate-500">
        {highlighted && <CheckCircle2 className="h-3 w-3 text-emerald-600" aria-hidden />}
        {label}
      </p>
      <p className="font-mono text-base font-semibold text-slate-900 dark:text-slate-100">
        {formatEur(amount)}
      </p>
      <p className="text-xs text-slate-500">{percent}%</p>
    </div>
  )
}

function formatEur(centimes: number): string {
  return new Intl.NumberFormat("fr-FR", { style: "currency", currency: "EUR" }).format(
    centimes / 100,
  )
}

function formatDate(iso: string): string {
  try {
    return new Date(iso).toLocaleDateString("fr-FR", {
      day: "numeric",
      month: "long",
      year: "numeric",
    })
  } catch {
    return iso
  }
}
