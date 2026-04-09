"use client"

import {
  AlertTriangle, CheckCircle2, XCircle, Clock, Scale, ShieldAlert,
  ArrowRight, Ban,
} from "lucide-react"
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

export function DisputeSystemBubble({ type, metadata, currentUserId, conversationId }: DisputeSystemBubbleProps) {
  const router = useRouter()
  const config = DISPUTE_CONFIGS[type]
  if (!config) return null

  const Icon = config.icon
  const reason = (metadata.reason as string) ?? ""
  const proposalAmount = (metadata.proposal_amount as number) ?? 0
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
