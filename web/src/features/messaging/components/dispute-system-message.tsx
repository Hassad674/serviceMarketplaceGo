"use client"

import {
  AlertTriangle, CheckCircle2, XCircle, Clock, Scale, ShieldAlert,
  ArrowRight, Ban, Calendar,
} from "lucide-react"
import { useTranslations } from "next-intl"
import { useRouter } from "@i18n/navigation"
import { cn, formatCurrency } from "@/shared/lib/utils"

// Soleil v2 — dispute system messages.
// Same 4 visual buckets as proposal-system-message: success / action /
// pending / neutral. Card surface stays uniform ivoire (`bg-card` +
// `border-border`); icon disc + title color carry the semantic.

type SystemConfig = {
  icon: React.ElementType
  iconBg: string
  iconColor: string
  title: string
}

const SUCCESS = { iconBg: "bg-success-soft", iconColor: "text-success" } as const
const PENDING = { iconBg: "bg-[var(--amber-soft)]", iconColor: "text-[var(--warning)]" } as const
const NEUTRAL = { iconBg: "bg-muted", iconColor: "text-muted-foreground" } as const

const DISPUTE_CONFIGS: Record<string, SystemConfig> = {
  dispute_opened: { icon: AlertTriangle, ...PENDING, title: "Litige ouvert" },
  dispute_counter_proposal: { icon: Scale, ...PENDING, title: "Proposition" },
  dispute_counter_accepted: { icon: CheckCircle2, ...SUCCESS, title: "Proposition acceptee" },
  dispute_counter_rejected: { icon: XCircle, ...NEUTRAL, title: "Proposition refusee" },
  dispute_escalated: { icon: ShieldAlert, ...PENDING, title: "Escalade en mediation" },
  dispute_resolved: { icon: CheckCircle2, ...SUCCESS, title: "Litige resolu" },
  dispute_cancelled: { icon: XCircle, ...NEUTRAL, title: "Litige annule" },
  dispute_auto_resolved: { icon: Clock, ...PENDING, title: "Litige resolu automatiquement" },
  dispute_cancellation_requested: { icon: Ban, ...PENDING, title: "Demande d'annulation" },
  dispute_cancellation_refused: { icon: XCircle, ...NEUTRAL, title: "Annulation refusee" },
}

const CARD_CLASSES =
  "w-full max-w-[400px] rounded-xl border border-border bg-card p-4 animate-scale-in"

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
      <div className={CARD_CLASSES}>
        <div className="flex items-start gap-3">
          <div className={cn("flex h-9 w-9 shrink-0 items-center justify-center rounded-full", config.iconBg)}>
            <Icon className={cn("h-4.5 w-4.5", config.iconColor)} strokeWidth={1.5} />
          </div>
          <div className="min-w-0 flex-1">
            <p className={cn("text-sm font-semibold", config.iconColor)}>{config.title}</p>
            {subtitle && (
              <p className="mt-0.5 text-xs text-muted-foreground">{subtitle}</p>
            )}
          </div>
        </div>

        {/* Party message (if present) */}
        {partyMessage && (
          <p className="mt-2 text-xs text-muted-foreground italic pl-12">
            &quot;{partyMessage}&quot;
          </p>
        )}

        {/* Resolution note */}
        {resolutionNote && (
          <p className="mt-2 text-xs text-muted-foreground italic pl-12">
            {resolutionNote}
          </p>
        )}

        {/* "Voir les details" button — links to the project page */}
        {showDetailsBtn && (
          <>
            <div className="mt-3 border-t border-border" />
            <button
              type="button"
              onClick={() => router.push(`/projects/${proposalId}`)}
              className={cn(
                "mt-3 w-full inline-flex items-center justify-center gap-2 rounded-full px-4 py-2",
                "text-sm font-semibold text-white transition-all duration-200",
                "bg-primary hover:opacity-90 active:scale-[0.98]",
                "focus-visible:outline-none focus-visible:ring-4 focus-visible:ring-primary/20",
              )}
            >
              Voir les details
              <ArrowRight className="h-4 w-4" strokeWidth={1.5} />
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
      <div className="w-full max-w-[440px] rounded-xl border border-border bg-card p-4 animate-scale-in">
        <div className="flex items-start gap-3">
          <div className="mt-0.5 flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-success-soft">
            <Scale className="h-4 w-4 text-success" aria-hidden />
          </div>
          <div className="min-w-0 flex-1">
            <p className="text-sm font-semibold text-success">
              {t("decisionTitle")}
            </p>
            <p className="mt-0.5 text-xs text-muted-foreground">
              {t("decisionYourShare", { percent: myPct, amount: formatCurrency(myAmount / 100) })}
            </p>

            <div className="mt-3 grid grid-cols-2 gap-2 rounded-lg bg-background p-3 text-sm">
              <DecisionSplitCell label={t("client")} amount={clientAmount} percent={clientPct} highlighted={isClient} />
              <DecisionSplitCell label={t("provider")} amount={providerAmount} percent={providerPct} highlighted={!isClient} />
            </div>

            {resolutionNote && (
              <div className="mt-3 rounded-lg bg-background p-3 text-sm">
                <p className="mb-1 text-xs font-medium text-success">
                  {t("decisionMessage")}
                </p>
                <p className="whitespace-pre-wrap text-foreground">
                  {resolutionNote}
                </p>
              </div>
            )}

            {resolvedAt && (
              <p className="mt-3 flex items-center gap-1 text-xs text-muted-foreground">
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
          ? "rounded-md border border-success/30 bg-card p-2"
          : "p-2"
      }
    >
      <p className="flex items-center gap-1 text-xs text-muted-foreground">
        {highlighted && <CheckCircle2 className="h-3 w-3 text-success" aria-hidden />}
        {label}
      </p>
      <p className="font-mono text-base font-semibold text-foreground">
        {formatCurrency(amount / 100)}
      </p>
      <p className="text-xs text-muted-foreground">{percent}%</p>
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
