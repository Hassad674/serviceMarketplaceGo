import { cn } from "@/shared/lib/utils"
import type { ReferralStatus } from "../types"
import { statusTone } from "../types"

interface ReferralStatusBadgeProps {
  status: ReferralStatus
  className?: string
}

// labelFor returns the French human label for a status.
function labelFor(status: ReferralStatus): string {
  switch (status) {
    case "pending_provider":
      return "En attente du prestataire"
    case "pending_referrer":
      return "En attente de l'apporteur"
    case "pending_client":
      return "En attente du client"
    case "active":
      return "Active"
    case "rejected":
      return "Refusée"
    case "expired":
      return "Expirée"
    case "cancelled":
      return "Annulée"
    case "terminated":
      return "Terminée"
  }
}

const TONE_CLASSES: Record<ReturnType<typeof statusTone>, string> = {
  pending: "bg-amber-50 text-amber-700 ring-amber-200",
  active: "bg-emerald-50 text-emerald-700 ring-emerald-200",
  "terminal-success": "bg-slate-50 text-slate-700 ring-slate-200",
  "terminal-failure": "bg-rose-50 text-rose-700 ring-rose-200",
}

export function ReferralStatusBadge({ status, className }: ReferralStatusBadgeProps) {
  const tone = statusTone(status)
  return (
    <span
      className={cn(
        "inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ring-1 ring-inset",
        TONE_CLASSES[tone],
        className,
      )}
    >
      {labelFor(status)}
    </span>
  )
}
