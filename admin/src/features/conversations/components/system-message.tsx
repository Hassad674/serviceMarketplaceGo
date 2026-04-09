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
    iconColor: "text-rose-600",
    iconBg: "bg-rose-100",
    cardBg: "bg-rose-50",
    cardBorder: "border-rose-200",
  },
  proposal_accepted: {
    icon: CheckCircle2,
    iconColor: "text-green-600",
    iconBg: "bg-green-100",
    cardBg: "bg-green-50",
    cardBorder: "border-green-200",
  },
  proposal_declined: {
    icon: XCircle,
    iconColor: "text-red-600",
    iconBg: "bg-red-100",
    cardBg: "bg-red-50",
    cardBorder: "border-red-200",
  },
  proposal_paid: {
    icon: DollarSign,
    iconColor: "text-blue-600",
    iconBg: "bg-blue-100",
    cardBg: "bg-blue-50",
    cardBorder: "border-blue-200",
  },
  proposal_payment_requested: {
    icon: CreditCard,
    iconColor: "text-blue-600",
    iconBg: "bg-blue-100",
    cardBg: "bg-blue-50",
    cardBorder: "border-blue-200",
  },
  proposal_completion_requested: {
    icon: Clock,
    iconColor: "text-amber-600",
    iconBg: "bg-amber-100",
    cardBg: "bg-amber-50",
    cardBorder: "border-amber-200",
  },
  proposal_completed: {
    icon: Trophy,
    iconColor: "text-emerald-600",
    iconBg: "bg-emerald-100",
    cardBg: "bg-emerald-50",
    cardBorder: "border-emerald-200",
  },
  proposal_completion_rejected: {
    icon: RotateCcw,
    iconColor: "text-slate-600",
    iconBg: "bg-slate-100",
    cardBg: "bg-slate-50",
    cardBorder: "border-slate-200",
  },
  proposal_modified: {
    icon: Pencil,
    iconColor: "text-purple-600",
    iconBg: "bg-purple-100",
    cardBg: "bg-purple-50",
    cardBorder: "border-purple-200",
  },
  evaluation_request: {
    icon: Star,
    iconColor: "text-amber-600",
    iconBg: "bg-amber-100",
    cardBg: "bg-amber-50",
    cardBorder: "border-amber-200",
  },
  call_ended: {
    icon: Phone,
    iconColor: "text-emerald-600",
    iconBg: "bg-emerald-100",
    cardBg: "bg-emerald-50",
    cardBorder: "border-emerald-200",
  },
  call_missed: {
    icon: PhoneMissed,
    iconColor: "text-red-600",
    iconBg: "bg-red-100",
    cardBg: "bg-red-50",
    cardBorder: "border-red-200",
  },
  dispute_opened: {
    icon: AlertTriangle,
    iconColor: "text-orange-600",
    iconBg: "bg-orange-100",
    cardBg: "bg-orange-50",
    cardBorder: "border-orange-200",
  },
  dispute_counter_proposal: {
    icon: Scale,
    iconColor: "text-amber-600",
    iconBg: "bg-amber-100",
    cardBg: "bg-amber-50",
    cardBorder: "border-amber-200",
  },
  dispute_counter_accepted: {
    icon: CheckCircle2,
    iconColor: "text-green-600",
    iconBg: "bg-green-100",
    cardBg: "bg-green-50",
    cardBorder: "border-green-200",
  },
  dispute_counter_rejected: {
    icon: XCircle,
    iconColor: "text-red-600",
    iconBg: "bg-red-100",
    cardBg: "bg-red-50",
    cardBorder: "border-red-200",
  },
  dispute_escalated: {
    icon: ShieldAlert,
    iconColor: "text-orange-600",
    iconBg: "bg-orange-100",
    cardBg: "bg-orange-50",
    cardBorder: "border-orange-200",
  },
  dispute_resolved: {
    icon: CheckCircle2,
    iconColor: "text-emerald-600",
    iconBg: "bg-emerald-100",
    cardBg: "bg-emerald-50",
    cardBorder: "border-emerald-200",
  },
  dispute_cancelled: {
    icon: XCircle,
    iconColor: "text-slate-600",
    iconBg: "bg-slate-100",
    cardBg: "bg-slate-50",
    cardBorder: "border-slate-200",
  },
  dispute_auto_resolved: {
    icon: Clock,
    iconColor: "text-amber-600",
    iconBg: "bg-amber-100",
    cardBg: "bg-amber-50",
    cardBorder: "border-amber-200",
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
