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
  Phone,
  PhoneMissed,
  Send,
} from "lucide-react"
import { cn } from "@/shared/lib/utils"
import { Badge } from "@/shared/components/ui/badge"
import type { AdminMessage } from "../types"

const SYSTEM_LABELS: Record<string, string> = {
  proposal_sent: "Proposition envoyee",
  proposal_modified: "Proposition modifiee",
  proposal_accepted: "Proposition acceptee",
  proposal_declined: "Proposition refusee",
  proposal_paid: "Paiement effectue",
  proposal_payment_requested: "Paiement demande",
  proposal_completion_requested: "Achevement demande",
  proposal_completed: "Mission terminee",
  proposal_completion_rejected: "Achevement rejete",
  evaluation_request: "Demande d'evaluation",
  call_ended: "Appel termine",
  call_missed: "Appel manque",
  dispute_opened: "Litige ouvert",
  dispute_counter_proposal: "Contre-proposition",
  dispute_counter_accepted: "Proposition acceptee",
  dispute_counter_rejected: "Proposition refusee",
  dispute_escalated: "Escalade en mediation",
  dispute_resolved: "Litige resolu",
  dispute_cancelled: "Litige annule",
  dispute_auto_resolved: "Litige resolu automatiquement",
}

type SystemConfig = {
  icon: React.ElementType
  iconColor: string
  iconBg: string
  cardBg: string
  cardBorder: string
}

const SYSTEM_STYLES: Record<string, SystemConfig> = {
  proposal_sent: {
    icon: Send,
    iconColor: "text-primary-deep",
    iconBg: "bg-primary/15",
    cardBg: "bg-primary-soft",
    cardBorder: "border-primary/30",
  },
  proposal_accepted: {
    icon: CheckCircle2,
    iconColor: "text-success",
    iconBg: "bg-success/15",
    cardBg: "bg-success-soft",
    cardBorder: "border-success/30",
  },
  proposal_declined: {
    icon: XCircle,
    iconColor: "text-destructive",
    iconBg: "bg-destructive/15",
    cardBg: "bg-destructive/5",
    cardBorder: "border-destructive/20",
  },
  proposal_paid: {
    icon: DollarSign,
    iconColor: "text-primary-deep",
    iconBg: "bg-primary/15",
    cardBg: "bg-primary-soft",
    cardBorder: "border-primary/30",
  },
  proposal_payment_requested: {
    icon: CreditCard,
    iconColor: "text-primary-deep",
    iconBg: "bg-primary/15",
    cardBg: "bg-primary-soft",
    cardBorder: "border-primary/30",
  },
  proposal_completion_requested: {
    icon: Clock,
    iconColor: "text-[var(--warning)]",
    iconBg: "bg-[var(--amber-soft)]",
    cardBg: "bg-[var(--amber-soft)]",
    cardBorder: "border-[var(--warning)]/30",
  },
  proposal_completed: {
    icon: Trophy,
    iconColor: "text-success",
    iconBg: "bg-success/15",
    cardBg: "bg-success-soft",
    cardBorder: "border-success/30",
  },
  proposal_completion_rejected: {
    icon: RotateCcw,
    iconColor: "text-muted-foreground",
    iconBg: "bg-muted",
    cardBg: "bg-muted",
    cardBorder: "border-border-strong",
  },
  proposal_modified: {
    icon: Pencil,
    iconColor: "text-primary-deep",
    iconBg: "bg-[var(--pink-soft)]",
    cardBg: "bg-[var(--pink-soft)]",
    cardBorder: "border-pink/30",
  },
  evaluation_request: {
    icon: Star,
    iconColor: "text-[var(--warning)]",
    iconBg: "bg-[var(--amber-soft)]",
    cardBg: "bg-[var(--amber-soft)]",
    cardBorder: "border-[var(--warning)]/30",
  },
  call_ended: {
    icon: Phone,
    iconColor: "text-success",
    iconBg: "bg-success/15",
    cardBg: "bg-success-soft",
    cardBorder: "border-success/30",
  },
  call_missed: {
    icon: PhoneMissed,
    iconColor: "text-destructive",
    iconBg: "bg-destructive/15",
    cardBg: "bg-destructive/5",
    cardBorder: "border-destructive/20",
  },
  dispute_opened: {
    icon: AlertTriangle,
    iconColor: "text-[var(--warning)]",
    iconBg: "bg-[var(--amber-soft)]",
    cardBg: "bg-[var(--amber-soft)]",
    cardBorder: "border-[var(--warning)]/30",
  },
  dispute_counter_proposal: {
    icon: Scale,
    iconColor: "text-[var(--warning)]",
    iconBg: "bg-[var(--amber-soft)]",
    cardBg: "bg-[var(--amber-soft)]",
    cardBorder: "border-[var(--warning)]/30",
  },
  dispute_counter_accepted: {
    icon: CheckCircle2,
    iconColor: "text-success",
    iconBg: "bg-success/15",
    cardBg: "bg-success-soft",
    cardBorder: "border-success/30",
  },
  dispute_counter_rejected: {
    icon: XCircle,
    iconColor: "text-destructive",
    iconBg: "bg-destructive/15",
    cardBg: "bg-destructive/5",
    cardBorder: "border-destructive/20",
  },
  dispute_escalated: {
    icon: ShieldAlert,
    iconColor: "text-[var(--warning)]",
    iconBg: "bg-[var(--amber-soft)]",
    cardBg: "bg-[var(--amber-soft)]",
    cardBorder: "border-[var(--warning)]/30",
  },
  dispute_resolved: {
    icon: CheckCircle2,
    iconColor: "text-success",
    iconBg: "bg-success/15",
    cardBg: "bg-success-soft",
    cardBorder: "border-success/30",
  },
  dispute_cancelled: {
    icon: XCircle,
    iconColor: "text-muted-foreground",
    iconBg: "bg-muted",
    cardBg: "bg-muted",
    cardBorder: "border-border-strong",
  },
  dispute_auto_resolved: {
    icon: Clock,
    iconColor: "text-[var(--warning)]",
    iconBg: "bg-[var(--amber-soft)]",
    cardBg: "bg-[var(--amber-soft)]",
    cardBorder: "border-[var(--warning)]/30",
  },
}

function getProposalSubtitle(metadata?: Record<string, unknown>): string | null {
  if (!metadata) return null
  const title = metadata.proposal_title as string | undefined
  const amount = metadata.proposal_amount as number | undefined
  if (!title) return null
  const formatted = amount != null ? `${(amount / 100).toLocaleString("fr-FR")} EUR` : ""
  return formatted ? `${title} — ${formatted}` : title
}

export function SystemMessage({ message }: { message: AdminMessage }) {
  const config = SYSTEM_STYLES[message.type]
  const label = SYSTEM_LABELS[message.type] ?? message.type
  const subtitle = getProposalSubtitle(message.metadata)

  if (!config) {
    return (
      <div className="flex justify-center py-2">
        <Badge variant="outline">{label}</Badge>
      </div>
    )
  }

  const Icon = config.icon

  return (
    <div className="flex justify-center py-3">
      <div
        className={cn(
          "w-full max-w-[400px] rounded-xl border p-4",
          config.cardBg,
          config.cardBorder,
        )}
      >
        <div className="flex items-start gap-3">
          <div
            className={cn(
              "flex h-9 w-9 shrink-0 items-center justify-center rounded-full",
              config.iconBg,
            )}
          >
            <Icon className={cn("h-4 w-4", config.iconColor)} strokeWidth={1.5} />
          </div>
          <div className="min-w-0 flex-1">
            <p className={cn("text-sm font-semibold", config.iconColor)}>{label}</p>
            {subtitle && (
              <p className="mt-0.5 truncate text-xs text-muted-foreground">
                {subtitle}
              </p>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}

export function isSystemMessage(type: string): boolean {
  return type !== "text" && type !== "file" && type !== "voice"
}
