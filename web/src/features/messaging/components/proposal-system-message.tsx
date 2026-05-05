"use client"

import {
  AlertTriangle,
  CheckCircle2,
  XCircle,
  DollarSign,
  CreditCard,
  Clock,
  RotateCcw,
  Pencil,
  Scale,
  ShieldAlert,
  Star,
  Trophy,
  ArrowRight,
} from "lucide-react"
import { useRouter } from "@i18n/navigation"
import { useTranslations } from "next-intl"
import { cn, formatCurrency } from "@/shared/lib/utils"
import type { ProposalMessageMetadata } from "../types"

// Soleil v2 — system / proposal status messages.
// 4 visual buckets (success / action / pending / closed) keyed off
// the existing message type. Card surface stays uniform ivoire
// (`bg-card` + `border-border`); the icon disc + title color carry
// the semantic differentiation, matching the Soleil pattern used
// across notifications and wallet history rows.

type SystemMessageConfig = {
  icon: React.ElementType
  iconBg: string
  iconColor: string
}

const SUCCESS: Pick<SystemMessageConfig, "iconBg" | "iconColor"> = {
  iconBg: "bg-success-soft",
  iconColor: "text-success",
}

const ACTION: Pick<SystemMessageConfig, "iconBg" | "iconColor"> = {
  iconBg: "bg-primary-soft",
  iconColor: "text-primary",
}

const PENDING: Pick<SystemMessageConfig, "iconBg" | "iconColor"> = {
  iconBg: "bg-[var(--amber-soft)]",
  iconColor: "text-[var(--warning)]",
}

const NEUTRAL: Pick<SystemMessageConfig, "iconBg" | "iconColor"> = {
  iconBg: "bg-muted",
  iconColor: "text-muted-foreground",
}

const SYSTEM_MESSAGE_STYLES: Record<string, SystemMessageConfig> = {
  // Success bucket
  proposal_accepted: { icon: CheckCircle2, ...SUCCESS },
  proposal_completed: { icon: Trophy, ...SUCCESS },
  milestone_released: { icon: CheckCircle2, ...SUCCESS },
  dispute_counter_accepted: { icon: CheckCircle2, ...SUCCESS },
  dispute_resolved: { icon: CheckCircle2, ...SUCCESS },

  // Action / Money bucket
  proposal_paid: { icon: DollarSign, ...ACTION },
  proposal_payment_requested: { icon: CreditCard, ...ACTION },
  evaluation_request: { icon: Star, ...ACTION },

  // Pending / Warning bucket
  proposal_completion_requested: { icon: Clock, ...PENDING },
  milestone_auto_approved: { icon: Clock, ...PENDING },
  dispute_opened: { icon: AlertTriangle, ...PENDING },
  dispute_counter_proposal: { icon: Scale, ...PENDING },
  dispute_escalated: { icon: ShieldAlert, ...PENDING },
  dispute_auto_resolved: { icon: Clock, ...PENDING },

  // Closed / Neutral bucket
  proposal_declined: { icon: XCircle, ...NEUTRAL },
  proposal_cancelled: { icon: XCircle, ...NEUTRAL },
  proposal_auto_closed: { icon: XCircle, ...NEUTRAL },
  proposal_completion_rejected: { icon: RotateCcw, ...NEUTRAL },
  proposal_modified: { icon: Pencil, ...NEUTRAL },
  dispute_counter_rejected: { icon: XCircle, ...NEUTRAL },
  dispute_cancelled: { icon: XCircle, ...NEUTRAL },
}

const CARD_CLASSES =
  "w-full max-w-[400px] rounded-xl border border-border bg-card p-4 animate-scale-in"

const CTA_CLASSES = cn(
  "mt-3 w-full inline-flex items-center justify-center gap-2 rounded-full px-4 py-2",
  "text-sm font-semibold text-white transition-all duration-200",
  "bg-primary hover:opacity-90 active:scale-[0.98]",
  "focus-visible:outline-none focus-visible:ring-4 focus-visible:ring-primary/20",
)

function getSystemMessageTitle(type: string, t: ReturnType<typeof useTranslations<"proposal">>) {
  const titles: Record<string, string> = {
    proposal_accepted: t("systemAccepted"),
    proposal_declined: t("systemDeclined"),
    proposal_paid: t("systemPaid"),
    proposal_completed: t("systemCompleted"),
    proposal_completion_requested: t("systemCompletionRequested"),
    proposal_completion_rejected: t("systemCompletionRejected"),
    proposal_modified: t("systemModified"),
    proposal_payment_requested: t("systemPaymentRequested"),
    evaluation_request: t("systemEvaluationRequest"),
    // Phase 12: milestone-scoped system messages.
    milestone_released: t("systemMilestoneReleased"),
    milestone_auto_approved: t("systemMilestoneAutoApproved"),
    proposal_cancelled: t("systemProposalCancelled"),
    proposal_auto_closed: t("systemProposalAutoClosed"),
    dispute_opened: "Litige ouvert",
    dispute_counter_proposal: "Contre-proposition",
    dispute_counter_accepted: "Proposition acceptee",
    dispute_counter_rejected: "Proposition refusee",
    dispute_escalated: "Escalade en mediation",
    dispute_resolved: "Litige resolu",
    dispute_cancelled: "Litige annule",
    dispute_auto_resolved: "Litige resolu automatiquement",
  }
  return titles[type] ?? ""
}

export function ProposalSystemMessage({
  type,
  metadata,
}: {
  type: string
  metadata: ProposalMessageMetadata
}) {
  const t = useTranslations("proposal")
  const config = SYSTEM_MESSAGE_STYLES[type]
  if (!config) return null

  const Icon = config.icon
  const title = getSystemMessageTitle(type, t)
  const subtitle = `${metadata.proposal_title} — ${formatCurrency(metadata.proposal_amount / 100)}`

  return (
    <div className="flex justify-center py-2">
      <div className={CARD_CLASSES}>
        <div className="flex items-start gap-3">
          <div className={cn("flex h-9 w-9 shrink-0 items-center justify-center rounded-full", config.iconBg)}>
            <Icon className={cn("h-4.5 w-4.5", config.iconColor)} strokeWidth={1.5} />
          </div>
          <div className="min-w-0 flex-1">
            <p className={cn("text-sm font-semibold", config.iconColor)}>
              {title}
            </p>
            <p className="mt-0.5 text-xs text-muted-foreground truncate">
              {subtitle}
            </p>
          </div>
        </div>
      </div>
    </div>
  )
}

export function PaymentRequestedMessage({
  metadata,
  currentUserId,
}: {
  metadata: ProposalMessageMetadata
  currentUserId: string
}) {
  const t = useTranslations("proposal")
  const router = useRouter()
  const config = SYSTEM_MESSAGE_STYLES.proposal_payment_requested
  const Icon = config.icon
  const title = getSystemMessageTitle("proposal_payment_requested", t)
  const subtitle = `${metadata.proposal_title} — ${formatCurrency(metadata.proposal_amount / 100)}`

  return (
    <div className="flex justify-center py-2">
      <div className={CARD_CLASSES}>
        <div className="flex items-start gap-3">
          <div className={cn("flex h-9 w-9 shrink-0 items-center justify-center rounded-full", config.iconBg)}>
            <Icon className={cn("h-4.5 w-4.5", config.iconColor)} strokeWidth={1.5} />
          </div>
          <div className="min-w-0 flex-1">
            <p className={cn("text-sm font-semibold", config.iconColor)}>
              {title}
            </p>
            <p className="mt-0.5 text-xs text-muted-foreground truncate">
              {subtitle}
            </p>
          </div>
        </div>
        {metadata.proposal_client_id === currentUserId && (
          <>
            <div className="mt-3 border-t border-border" />
            <button
              type="button"
              onClick={() => router.push(`/projects/pay?proposal=${metadata.proposal_id}`)}
              className={CTA_CLASSES}
            >
              {t("payNow")}
              <ArrowRight className="h-4 w-4" strokeWidth={1.5} />
            </button>
          </>
        )}
      </div>
    </div>
  )
}

export function CompletionRequestedMessage({
  metadata,
  currentUserId,
}: {
  metadata: ProposalMessageMetadata
  currentUserId: string
}) {
  const t = useTranslations("proposal")
  const router = useRouter()
  const config = SYSTEM_MESSAGE_STYLES.proposal_completion_requested
  const Icon = config.icon
  const title = getSystemMessageTitle("proposal_completion_requested", t)
  const subtitle = `${metadata.proposal_title} — ${formatCurrency(metadata.proposal_amount / 100)}`

  return (
    <div className="flex justify-center py-2">
      <div className={CARD_CLASSES}>
        <div className="flex items-start gap-3">
          <div className={cn("flex h-9 w-9 shrink-0 items-center justify-center rounded-full", config.iconBg)}>
            <Icon className={cn("h-4.5 w-4.5", config.iconColor)} strokeWidth={1.5} />
          </div>
          <div className="min-w-0 flex-1">
            <p className={cn("text-sm font-semibold", config.iconColor)}>
              {title}
            </p>
            <p className="mt-0.5 text-xs text-muted-foreground truncate">
              {subtitle}
            </p>
          </div>
        </div>
        {metadata.proposal_client_id === currentUserId && (
          <>
            <div className="mt-3 border-t border-border" />
            <button
              type="button"
              onClick={() => router.push(`/projects/${metadata.proposal_id}`)}
              className={CTA_CLASSES}
            >
              {t("viewDetails")}
              <ArrowRight className="h-4 w-4" strokeWidth={1.5} />
            </button>
          </>
        )}
      </div>
    </div>
  )
}

export function EvaluationRequestMessage({
  metadata,
  onReview,
}: {
  metadata: ProposalMessageMetadata
  // Double-blind reviews: the consumer needs the proposal's client
  // and provider ORGANIZATION ids to derive the viewer's review side.
  // We forward them straight from the system-message metadata instead
  // of re-fetching the proposal. The user-level client/provider ids
  // would NOT work — they don't match useUser().organization.id in
  // the post-phase-4 org model.
  onReview?: (
    proposalId: string,
    proposalTitle: string,
    participants: { clientOrganizationId: string; providerOrganizationId: string },
  ) => void
}) {
  const t = useTranslations("review")
  const tp = useTranslations("proposal")
  const config = SYSTEM_MESSAGE_STYLES.evaluation_request
  const Icon = config.icon
  const title = tp("systemEvaluationRequest")

  // Legacy messages (emitted before the backend started enriching
  // metadata with org ids) cannot open the modal correctly, so we hide
  // the CTA entirely on them rather than silently drop the click.
  const clientOrgId = metadata.proposal_client_organization_id
  const providerOrgId = metadata.proposal_provider_organization_id
  const ctaEnabled = Boolean(clientOrgId && providerOrgId)

  return (
    <div className="flex justify-center py-2">
      <div className={CARD_CLASSES}>
        <div className="flex items-start gap-3">
          <div className={cn("flex h-9 w-9 shrink-0 items-center justify-center rounded-full", config.iconBg)}>
            <Icon className={cn("h-4.5 w-4.5", config.iconColor)} strokeWidth={1.5} />
          </div>
          <div className="min-w-0 flex-1">
            <p className={cn("text-sm font-semibold", config.iconColor)}>
              {t("evaluationRequest")}
            </p>
            <p className="mt-0.5 text-xs text-muted-foreground truncate">
              {title}
            </p>
          </div>
        </div>
        {ctaEnabled && (
          <>
            <div className="mt-3 border-t border-border" />
            <button
              type="button"
              onClick={() =>
                onReview?.(metadata.proposal_id, metadata.proposal_title, {
                  clientOrganizationId: clientOrgId!,
                  providerOrganizationId: providerOrgId!,
                })
              }
              className={CTA_CLASSES}
            >
              {t("leaveReview")}
              <ArrowRight className="h-4 w-4" strokeWidth={1.5} />
            </button>
          </>
        )}
      </div>
    </div>
  )
}
