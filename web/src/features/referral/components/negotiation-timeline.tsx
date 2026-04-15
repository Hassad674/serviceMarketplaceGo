import { ArrowDown, Check, Repeat, X } from "lucide-react"

import { useReferralNegotiations } from "../hooks/use-referrals"
import type { ReferralNegotiation } from "../types"

interface NegotiationTimelineProps {
  referralId: string
  // showRate hides the rate column for client viewers pre-activation
  // (Modèle A: the client must never see the historical rate).
  showRate: boolean
}

const ACTION_STYLES: Record<
  ReferralNegotiation["action"],
  { label: string; icon: React.ReactNode; tone: string }
> = {
  proposed: {
    label: "Proposition initiale",
    icon: <ArrowDown className="h-3.5 w-3.5" aria-hidden="true" />,
    tone: "bg-rose-50 text-rose-700",
  },
  countered: {
    label: "Contre-proposition",
    icon: <Repeat className="h-3.5 w-3.5" aria-hidden="true" />,
    tone: "bg-amber-50 text-amber-700",
  },
  accepted: {
    label: "Accepté",
    icon: <Check className="h-3.5 w-3.5" aria-hidden="true" />,
    tone: "bg-emerald-50 text-emerald-700",
  },
  rejected: {
    label: "Refusé",
    icon: <X className="h-3.5 w-3.5" aria-hidden="true" />,
    tone: "bg-slate-100 text-slate-600",
  },
}

const ROLE_LABELS: Record<ReferralNegotiation["actor_role"], string> = {
  referrer: "Apporteur",
  provider: "Prestataire",
  client: "Client",
}

// NegotiationTimeline renders the audit trail of negotiation events for a
// referral. Sorted oldest-first by the backend so we render in scroll order.
export function NegotiationTimeline({
  referralId,
  showRate,
}: NegotiationTimelineProps) {
  const { data, isLoading } = useReferralNegotiations(referralId)

  if (isLoading) {
    return (
      <div className="space-y-2">
        {[0, 1].map((i) => (
          <div
            key={i}
            className="h-12 animate-pulse rounded-lg border border-slate-200 bg-slate-50"
          />
        ))}
      </div>
    )
  }
  if (!data || data.length === 0) {
    return (
      <p className="rounded-lg border border-dashed border-slate-200 px-4 py-6 text-center text-sm text-slate-500">
        Aucun événement de négociation pour le moment.
      </p>
    )
  }

  return (
    <ol className="space-y-2">
      {data.map((event) => {
        const style = ACTION_STYLES[event.action]
        return (
          <li
            key={event.id}
            className="flex items-start gap-3 rounded-lg border border-slate-200 bg-white p-3"
          >
            <span
              className={`grid h-7 w-7 shrink-0 place-items-center rounded-full ${style.tone}`}
              aria-hidden="true"
            >
              {style.icon}
            </span>
            <div className="flex-1 text-sm">
              <div className="flex items-center gap-2">
                <span className="font-medium text-slate-900">
                  {ROLE_LABELS[event.actor_role]}
                </span>
                <span className="text-xs text-slate-500">
                  {style.label} · v{event.version}
                </span>
                {showRate && (
                  <span className="ml-auto rounded-full bg-slate-100 px-2 py-0.5 font-mono text-xs text-slate-700">
                    {event.rate_pct.toFixed(event.rate_pct % 1 === 0 ? 0 : 1)}%
                  </span>
                )}
              </div>
              {event.message && (
                <p className="mt-1 text-sm text-slate-600">&ldquo;{event.message}&rdquo;</p>
              )}
              <p className="mt-1 text-xs text-slate-500">
                {new Date(event.created_at).toLocaleString("fr-FR", {
                  dateStyle: "medium",
                  timeStyle: "short",
                })}
              </p>
            </div>
          </li>
        )
      })}
    </ol>
  )
}
