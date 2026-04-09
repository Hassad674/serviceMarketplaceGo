"use client"

import { ShieldAlert, Clock, CheckCircle2, XCircle, ArrowRight, Ban } from "lucide-react"
import { useTranslations } from "next-intl"

import type { DisputeResponse } from "../types"

interface DisputeBannerProps {
  dispute: DisputeResponse
  currentUserId: string
  onCounterPropose?: () => void
  onAcceptProposal?: (cpId: string) => void
  onRejectProposal?: (cpId: string) => void
  onCancel?: () => void
  onAcceptCancellation?: () => void
  onRefuseCancellation?: () => void
}

const STATUS_CONFIG = {
  open: { icon: ShieldAlert, bg: "bg-red-50 dark:bg-red-500/10", border: "border-red-200 dark:border-red-500/20", text: "text-red-900 dark:text-red-300", iconColor: "text-red-500" },
  negotiation: { icon: Clock, bg: "bg-amber-50 dark:bg-amber-500/10", border: "border-amber-200 dark:border-amber-500/20", text: "text-amber-900 dark:text-amber-300", iconColor: "text-amber-500" },
  escalated: { icon: ShieldAlert, bg: "bg-orange-50 dark:bg-orange-500/10", border: "border-orange-200 dark:border-orange-500/20", text: "text-orange-900 dark:text-orange-300", iconColor: "text-orange-500" },
  resolved: { icon: CheckCircle2, bg: "bg-green-50 dark:bg-green-500/10", border: "border-green-200 dark:border-green-500/20", text: "text-green-900 dark:text-green-300", iconColor: "text-green-500" },
  cancelled: { icon: XCircle, bg: "bg-slate-50 dark:bg-slate-500/10", border: "border-slate-200 dark:border-slate-500/20", text: "text-slate-900 dark:text-slate-300", iconColor: "text-slate-500" },
} as const

export function DisputeBanner({
  dispute,
  currentUserId,
  onCounterPropose,
  onAcceptProposal,
  onRejectProposal,
  onCancel,
  onAcceptCancellation,
  onRefuseCancellation,
}: DisputeBannerProps) {
  const t = useTranslations("disputes")
  const config = STATUS_CONFIG[dispute.status]
  const Icon = config.icon
  const daysElapsed = Math.floor((Date.now() - new Date(dispute.created_at).getTime()) / (1000 * 60 * 60 * 24))
  const daysLeft = Math.max(0, 7 - daysElapsed)

  const lastCP = dispute.counter_proposals
    ?.filter((cp) => cp.status === "pending")
    .at(-1)

  // Can accept/reject: there's a pending proposal from the OTHER party
  const canRespond = lastCP && lastCP.proposer_id !== currentUserId

  // Feedback to the proposer after a refusal: if there is no pending CP, but
  // the most recent CP overall was rejected AND is from the current user,
  // show a "your last proposal was refused" block so they know what happened
  // without having to scroll the conversation.
  const allCPs = dispute.counter_proposals ?? []
  const latestCP = allCPs[allCPs.length - 1]
  const showRefusedFeedback =
    !lastCP &&
    latestCP &&
    latestCP.status === "rejected" &&
    latestCP.proposer_id === currentUserId

  // Cancellation request state.
  // Use a truthy check (not `!== null`) so missing/undefined fields from
  // older API responses are treated as "no request pending" — otherwise
  // `undefined !== null` wrongly flags every dispute as having a request.
  const hasCancellationRequest = !!dispute.cancellation_requested_by
  const isCancellationRequester =
    hasCancellationRequest && dispute.cancellation_requested_by === currentUserId
  const canRespondToCancellation =
    hasCancellationRequest && !isCancellationRequester

  return (
    <div
      role="alert"
      className={`rounded-xl border p-4 ${config.bg} ${config.border} animate-slide-up`}
    >
      <div className="flex items-start gap-3">
        <Icon className={`mt-0.5 h-5 w-5 shrink-0 ${config.iconColor}`} aria-hidden />
        <div className="flex-1 min-w-0">
          <p className={`text-sm font-semibold ${config.text}`}>
            {t(`status.${dispute.status}`)}
          </p>

          <p className="mt-1 text-sm text-slate-600 dark:text-slate-400">
            {t("reason." + dispute.reason)} — {formatEur(dispute.requested_amount)} {t("requested")}
          </p>

          {dispute.status === "open" || dispute.status === "negotiation" ? (
            <div className="mt-2 flex items-center gap-2 text-xs text-slate-500">
              <Clock className="h-3.5 w-3.5" aria-hidden />
              {daysLeft > 0
                ? t("daysLeft", { days: daysLeft })
                : t("escalationSoon")}
            </div>
          ) : null}

          {lastCP && (
            <div className="mt-2 rounded-lg bg-white/60 dark:bg-slate-800/60 p-3 text-sm">
              <p className="font-medium text-slate-700 dark:text-slate-300">{t("lastProposal")}</p>
              <p className="text-slate-600 dark:text-slate-400">
                {t("split", {
                  client: formatEur(lastCP.amount_client),
                  provider: formatEur(lastCP.amount_provider),
                })}
              </p>
              {lastCP.message && (
                <p className="mt-1 text-xs text-slate-500 italic">&quot;{lastCP.message}&quot;</p>
              )}
            </div>
          )}

          {showRefusedFeedback && latestCP && (
            <div className="mt-2 rounded-lg border border-red-200 bg-red-50/60 p-3 text-sm dark:border-red-500/30 dark:bg-red-500/10">
              <p className="flex items-center gap-1.5 font-medium text-red-700 dark:text-red-300">
                <XCircle className="h-3.5 w-3.5" aria-hidden />
                {t("yourLastProposalRefused")}
              </p>
              <p className="mt-1 text-xs text-red-600/80 dark:text-red-200/80">
                {t("split", {
                  client: formatEur(latestCP.amount_client),
                  provider: formatEur(latestCP.amount_provider),
                })}
              </p>
            </div>
          )}

          {hasCancellationRequest && !dispute.status.startsWith("resolved") && dispute.status !== "cancelled" && (
            <div className="mt-2 rounded-lg border border-amber-300 bg-amber-50/80 p-3 text-sm dark:border-amber-500/30 dark:bg-amber-500/10">
              <p className="flex items-center gap-1.5 font-medium text-amber-900 dark:text-amber-300">
                <Ban className="h-3.5 w-3.5" aria-hidden />
                {t("cancellationRequestPending")}
              </p>
              <p className="mt-1 text-xs text-amber-800/80 dark:text-amber-200/80">
                {isCancellationRequester
                  ? t("cancellationRequestWaiting")
                  : t("cancellationRequestConsent")}
              </p>
            </div>
          )}

          {dispute.status === "resolved" && dispute.resolution_note && (
            <div className="mt-2 rounded-lg bg-white/60 dark:bg-slate-800/60 p-3 text-sm">
              <p className="font-medium text-slate-700 dark:text-slate-300">{t("resolution")}</p>
              <p className="text-slate-600 dark:text-slate-400">{dispute.resolution_note}</p>
              {dispute.resolution_amount_client != null && dispute.resolution_amount_provider != null && (
                <p className="mt-1 text-xs text-slate-500">
                  {t("split", {
                    client: formatEur(dispute.resolution_amount_client),
                    provider: formatEur(dispute.resolution_amount_provider),
                  })}
                </p>
              )}
            </div>
          )}
        </div>
      </div>

      {(dispute.status === "open" || dispute.status === "negotiation") && (
        <div className="mt-3 flex flex-wrap gap-2 pl-8">
          {/* Cancellation request actions take priority — respondent must accept or refuse */}
          {canRespondToCancellation && onAcceptCancellation && onRefuseCancellation ? (
            <>
              <button
                type="button"
                onClick={onAcceptCancellation}
                className="inline-flex items-center gap-1 rounded-lg bg-slate-700 px-3 py-1.5 text-xs font-semibold text-white hover:bg-slate-800 transition-colors"
              >
                <CheckCircle2 className="h-3.5 w-3.5" aria-hidden />
                {t("acceptCancellation")}
              </button>
              <button
                type="button"
                onClick={onRefuseCancellation}
                className="inline-flex items-center gap-1 rounded-lg border border-slate-300 px-3 py-1.5 text-xs font-semibold text-slate-700 hover:bg-slate-100 transition-colors dark:border-slate-600 dark:text-slate-300 dark:hover:bg-slate-800"
              >
                <XCircle className="h-3.5 w-3.5" aria-hidden />
                {t("refuseCancellation")}
              </button>
            </>
          ) : (
            <>
              {/* Accept/Reject — only when there's a pending proposal from the other party */}
              {canRespond && lastCP && onAcceptProposal && onRejectProposal && (
                <>
                  <button
                    type="button"
                    onClick={() => onAcceptProposal(lastCP.id)}
                    className="inline-flex items-center gap-1 rounded-lg bg-green-600 px-3 py-1.5 text-xs font-semibold text-white hover:bg-green-700 transition-colors"
                  >
                    <CheckCircle2 className="h-3.5 w-3.5" aria-hidden />
                    {t("acceptCounter")}
                  </button>
                  <button
                    type="button"
                    onClick={() => onRejectProposal(lastCP.id)}
                    className="inline-flex items-center gap-1 rounded-lg border border-red-300 px-3 py-1.5 text-xs font-semibold text-red-700 hover:bg-red-50 transition-colors dark:border-red-600 dark:text-red-400"
                  >
                    <XCircle className="h-3.5 w-3.5" aria-hidden />
                    {t("rejectCounter")}
                  </button>
                </>
              )}
              {onCounterPropose && (
                <button
                  type="button"
                  onClick={onCounterPropose}
                  className="inline-flex items-center gap-1 rounded-lg bg-amber-600 px-3 py-1.5 text-xs font-semibold text-white hover:bg-amber-700 transition-colors"
                >
                  {t("counterPropose")}
                  <ArrowRight className="h-3.5 w-3.5" aria-hidden />
                </button>
              )}
              {/* Cancel button: only shown to initiator. If the respondent has
                  already replied this triggers a cancellation REQUEST;
                  otherwise the dispute is cancelled directly. Hidden when a
                  request is already pending (the requester is waiting). */}
              {onCancel && !hasCancellationRequest && (
                <button
                  type="button"
                  onClick={onCancel}
                  className="inline-flex items-center gap-1 rounded-lg border border-slate-300 px-3 py-1.5 text-xs font-semibold text-slate-600 hover:bg-slate-100 transition-colors dark:border-slate-600 dark:text-slate-400 dark:hover:bg-slate-800"
                >
                  {t("cancel")}
                </button>
              )}
            </>
          )}
        </div>
      )}
    </div>
  )
}

function formatEur(centimes: number): string {
  return new Intl.NumberFormat("fr-FR", { style: "currency", currency: "EUR" }).format(centimes / 100)
}
