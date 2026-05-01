"use client"

import {
  AlertTriangle, CheckCircle2, XCircle, Clock, Scale, ShieldAlert,
  ArrowRight, Ban, Calendar,
} from "lucide-react"
import { useTranslations } from "next-intl"
import { useRouter } from "@i18n/navigation"
import { cn, formatCurrency } from "@/shared/lib/utils"

type SystemConfig = {
  icon: React.ElementType
  iconColor: string
  iconBg: string
  cardBg: string
  cardBorder: string
  title: string
}

const DISPUTE_CONFIGS: Record<string, SystemConfig> = {
  dispute_opened: {
    icon: AlertTriangle, iconColor: "text-orange-600 dark:text-orange-400",
    iconBg: "bg-orange-100 dark:bg-orange-500/20",
    cardBg: "bg-orange-50 dark:bg-orange-900/20", cardBorder: "border-orange-200 dark:border-orange-800",
    title: "Litige ouvert",
  },
  dispute_counter_proposal: {
    icon: Scale, iconColor: "text-amber-600 dark:text-amber-400",
    iconBg: "bg-amber-100 dark:bg-amber-500/20",
    cardBg: "bg-amber-50 dark:bg-amber-900/20", cardBorder: "border-amber-200 dark:border-amber-800",
    title: "Proposition",
  },
  dispute_counter_accepted: {
    icon: CheckCircle2, iconColor: "text-green-600 dark:text-green-400",
    iconBg: "bg-green-100 dark:bg-green-500/20",
    cardBg: "bg-green-50 dark:bg-green-900/20", cardBorder: "border-green-200 dark:border-green-800",
    title: "Proposition acceptee",
  },
  dispute_counter_rejected: {
    icon: XCircle, iconColor: "text-red-600 dark:text-red-400",
    iconBg: "bg-red-100 dark:bg-red-500/20",
    cardBg: "bg-red-50 dark:bg-red-900/20", cardBorder: "border-red-200 dark:border-red-800",
    title: "Proposition refusee",
  },
  dispute_escalated: {
    icon: ShieldAlert, iconColor: "text-orange-600 dark:text-orange-400",
    iconBg: "bg-orange-100 dark:bg-orange-500/20",
    cardBg: "bg-orange-50 dark:bg-orange-900/20", cardBorder: "border-orange-200 dark:border-orange-800",
    title: "Escalade en mediation",
  },
  dispute_resolved: {
    icon: CheckCircle2, iconColor: "text-emerald-600 dark:text-emerald-400",
    iconBg: "bg-emerald-100 dark:bg-emerald-500/20",
    cardBg: "bg-emerald-50 dark:bg-emerald-900/20", cardBorder: "border-emerald-200 dark:border-emerald-800",
    title: "Litige resolu",
  },
  dispute_cancelled: {
    icon: XCircle, iconColor: "text-slate-600 dark:text-slate-400",
    iconBg: "bg-slate-100 dark:bg-slate-500/20",
    cardBg: "bg-slate-50 dark:bg-slate-800/50", cardBorder: "border-slate-200 dark:border-slate-700",
    title: "Litige annule",
  },
  dispute_auto_resolved: {
    icon: Clock, iconColor: "text-amber-600 dark:text-amber-400",
    iconBg: "bg-amber-100 dark:bg-amber-500/20",
    cardBg: "bg-amber-50 dark:bg-amber-900/20", cardBorder: "border-amber-200 dark:border-amber-800",
    title: "Litige resolu automatiquement",
  },
  dispute_cancellation_requested: {
    icon: Ban, iconColor: "text-amber-600 dark:text-amber-400",
    iconBg: "bg-amber-100 dark:bg-amber-500/20",
    cardBg: "bg-amber-50 dark:bg-amber-900/20", cardBorder: "border-amber-200 dark:border-amber-800",
    title: "Demande d'annulation",
  },
  dispute_cancellation_refused: {
    icon: XCircle, iconColor: "text-red-600 dark:text-red-400",
    iconBg: "bg-red-100 dark:bg-red-500/20",
    cardBg: "bg-red-50 dark:bg-red-900/20", cardBorder: "border-red-200 dark:border-red-800",
    title: "Annulation refusee",
  },
}

const REASON_LABELS: Record<string, string> = {
  work_not_conforming: "Travail non conforme",
  non_delivery: "Non-livraison",
  insufficient_quality: "Qualite insuffisante",
  client_ghosting: "Client injoignable",
  scope_creep: "Hors du scope",
  refusal_to_validate: "Refus de valider",
  harassment: "Harcelement",
  other: "Autre",
}

interface DisputeSystemBubbleProps {
  type: string
  metadata: Record<string, unknown>
  currentUserId: string
  conversationId: string
}

export function DisputeSystemBubble({ type, metadata, currentUserId, conversationId: _conversationId }: DisputeSystemBubbleProps) {
  const router = useRouter()
  const t = useTranslations("disputes")

  // Resolved (admin-rendered or auto-resolved): show the full decision card
  // with split + user share highlight + admin note + date, mirroring the
  // historical DisputeResolutionCard on the project page so both views stay
  // in sync and the user never misses the decision.
  if (type === "dispute_resolved" || type === "dispute_auto_resolved") {
    return <ResolvedDecisionCard metadata={metadata} currentUserId={currentUserId} t={t} />
  }

  const config = DISPUTE_CONFIGS[type]
  if (!config) return null

  const Icon = config.icon
  const reason = (metadata.reason as string) ?? ""
  const _proposalAmount = (metadata.proposal_amount as number) ?? 0
  const requestedAmount = (metadata.requested_amount as number) ?? 0
  const partyMessage = (metadata.message as string) ?? ""
  const amountClient = (metadata.amount_client as number) ?? (metadata.resolution_amount_client as number) ?? 0
  const amountProvider = (metadata.amount_provider as number) ?? (metadata.resolution_amount_provider as number) ?? 0
  const resolutionNote = (metadata.resolution_note as string) ?? ""
  const proposalId = (metadata.proposal_id as string) ?? ""

  // Determine subtitle — counter-proposal events (including rejected and
  // accepted) all need the amounts so the chat history is self-explanatory.
  let subtitle = ""
  if (type === "dispute_opened") {
    subtitle = `${REASON_LABELS[reason] ?? reason} — ${formatCurrency(requestedAmount / 100)}`
  } else if (
    type === "dispute_counter_proposal" ||
    type === "dispute_counter_accepted" ||
    type === "dispute_counter_rejected" ||
    type === "dispute_resolved" ||
    type === "dispute_auto_resolved"
  ) {
    subtitle = `Client: ${formatCurrency(amountClient / 100)} · Prestataire: ${formatCurrency(amountProvider / 100)}`
  } else if (type === "dispute_cancellation_requested") {
    subtitle = "Consentement requis pour annuler le litige"
  } else if (type === "dispute_cancellation_refused") {
    subtitle = "Le litige continue"
  } else {
    subtitle = REASON_LABELS[reason] ?? ""
  }

  // Show "Voir les details" button for actionable messages (not resolved/cancelled)
  const showDetailsBtn = proposalId && (
    type === "dispute_opened" ||
    type === "dispute_counter_proposal" ||
    type === "dispute_escalated" ||
    type === "dispute_cancellation_requested"
  )

  return (
    <div className="flex justify-center py-2">
      <div className={cn("w-full max-w-[400px] rounded-xl border p-4 animate-scale-in", config.cardBg, config.cardBorder)}>
        <div className="flex items-start gap-3">
          <div className={cn("flex h-9 w-9 shrink-0 items-center justify-center rounded-full", config.iconBg)}>
            <Icon className={cn("h-4.5 w-4.5", config.iconColor)} strokeWidth={1.5} />
          </div>
          <div className="min-w-0 flex-1">
            <p className={cn("text-sm font-semibold", config.iconColor)}>{config.title}</p>
            {subtitle && (
              <p className="mt-0.5 text-xs text-slate-600 dark:text-slate-400">{subtitle}</p>
            )}
          </div>
        </div>

        {/* Party message (if present) */}
        {partyMessage && (
          <p className="mt-2 text-xs text-slate-500 dark:text-slate-400 italic pl-12">
            &quot;{partyMessage}&quot;
          </p>
        )}

        {/* Resolution note */}
        {resolutionNote && (
          <p className="mt-2 text-xs text-slate-500 dark:text-slate-400 italic pl-12">
            {resolutionNote}
          </p>
        )}

        {/* "Voir les details" button — links to the project page */}
        {showDetailsBtn && (
          <>
            <div className="mt-3 border-t border-inherit" />
            <button
              type="button"
              onClick={() => router.push(`/projects/${proposalId}`)}
              className={cn(
                "mt-3 w-full flex items-center justify-center gap-1.5 rounded-lg px-3 py-2",
                "text-xs font-semibold transition-all duration-200",
                "border border-slate-300 text-slate-700 hover:bg-slate-100",
                "dark:border-slate-600 dark:text-slate-400 dark:hover:bg-slate-800",
              )}
            >
              Voir les details
              <ArrowRight className="h-3.5 w-3.5" />
            </button>
          </>
        )}
      </div>
    </div>
  )
}

interface ResolvedDecisionCardProps {
  metadata: Record<string, unknown>
  currentUserId: string
  t: ReturnType<typeof useTranslations>
}

function ResolvedDecisionCard({ metadata, currentUserId, t }: ResolvedDecisionCardProps) {
  const clientAmount = (metadata.resolution_amount_client as number) ?? 0
  const providerAmount = (metadata.resolution_amount_provider as number) ?? 0
  const total = clientAmount + providerAmount
  const clientPct = total > 0 ? Math.round((clientAmount / total) * 100) : 0
  const providerPct = 100 - clientPct

  const clientId = (metadata.client_id as string) ?? ""
  const isClient = currentUserId === clientId
  const myAmount = isClient ? clientAmount : providerAmount
  const myPct = isClient ? clientPct : providerPct

  const resolutionNote = (metadata.resolution_note as string) ?? ""
  const resolvedAt = (metadata.resolved_at as string) ?? ""

  return (
    <div className="flex justify-center py-2">
      <div className="w-full max-w-[440px] rounded-xl border border-emerald-200 bg-emerald-50/60 p-4 dark:border-emerald-500/30 dark:bg-emerald-500/10 animate-scale-in">
        <div className="flex items-start gap-3">
          <div className="mt-0.5 flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-emerald-100 dark:bg-emerald-500/20">
            <Scale className="h-4 w-4 text-emerald-700 dark:text-emerald-300" aria-hidden />
          </div>
          <div className="min-w-0 flex-1">
            <p className="text-sm font-semibold text-emerald-900 dark:text-emerald-200">
              {t("decisionTitle")}
            </p>
            <p className="mt-0.5 text-xs text-emerald-800/80 dark:text-emerald-200/80">
              {t("decisionYourShare", { percent: myPct, amount: formatCurrency(myAmount / 100) })}
            </p>

            <div className="mt-3 grid grid-cols-2 gap-2 rounded-lg bg-white/60 p-3 text-sm dark:bg-slate-800/40">
              <DecisionSplitCell label={t("client")} amount={clientAmount} percent={clientPct} highlighted={isClient} />
              <DecisionSplitCell label={t("provider")} amount={providerAmount} percent={providerPct} highlighted={!isClient} />
            </div>

            {resolutionNote && (
              <div className="mt-3 rounded-lg bg-white/60 p-3 text-sm dark:bg-slate-800/40">
                <p className="mb-1 text-xs font-medium text-emerald-900 dark:text-emerald-200">
                  {t("decisionMessage")}
                </p>
                <p className="whitespace-pre-wrap text-slate-700 dark:text-slate-300">
                  {resolutionNote}
                </p>
              </div>
            )}

            {resolvedAt && (
              <p className="mt-3 flex items-center gap-1 text-xs text-emerald-700/80 dark:text-emerald-300/80">
                <Calendar className="h-3 w-3" aria-hidden />
                {t("decisionRenderedOn", { date: formatResolvedDate(resolvedAt) })}
              </p>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}

interface DecisionSplitCellProps {
  label: string
  amount: number
  percent: number
  highlighted: boolean
}

function DecisionSplitCell({ label, amount, percent, highlighted }: DecisionSplitCellProps) {
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
        {formatCurrency(amount / 100)}
      </p>
      <p className="text-xs text-slate-500">{percent}%</p>
    </div>
  )
}

function formatResolvedDate(iso: string): string {
  try {
    return new Date(iso).toLocaleDateString("fr-FR", { day: "numeric", month: "long", year: "numeric" })
  } catch {
    return iso
  }
}
