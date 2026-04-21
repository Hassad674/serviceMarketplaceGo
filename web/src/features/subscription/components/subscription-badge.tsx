"use client"

import { Sparkles } from "lucide-react"
import { cn } from "@/shared/lib/utils"
import { useSubscription } from "../hooks/use-subscription"
import type { Subscription } from "../types"

type SubscriptionBadgeProps = {
  onClick: () => void
}

/**
 * Compact, clickable Premium badge for the navbar. The component
 * owns NO state beyond what the cached subscription gives it —
 * the parent decides which modal (upgrade / manage) opens on
 * click. Gating by role lives in the parent layout; here we only
 * render the visual state appropriate to the data.
 */
export function SubscriptionBadge({ onClick }: SubscriptionBadgeProps) {
  const { data: subscription, isLoading } = useSubscription()

  if (isLoading) return <BadgeSkeleton />

  const variant = pickVariant(subscription)

  return (
    <button
      type="button"
      onClick={onClick}
      aria-label={variant.ariaLabel}
      className={cn(
        "inline-flex h-6 items-center gap-1.5 rounded-full px-3 text-[11px] font-semibold",
        "transition-all duration-200 hover:scale-[1.02] active:scale-[0.98]",
        "focus:outline-none focus:ring-2 focus:ring-rose-500/40",
        variant.className,
      )}
    >
      {variant.icon ? (
        <Sparkles className="h-3 w-3" aria-hidden="true" strokeWidth={2} />
      ) : null}
      <span>{variant.label}</span>
    </button>
  )
}

type BadgeVariant = {
  label: string
  ariaLabel: string
  className: string
  icon: boolean
}

function pickVariant(sub: Subscription | null | undefined): BadgeVariant {
  if (!sub) {
    return {
      label: "Passer Premium",
      ariaLabel: "Passer Premium",
      className:
        "bg-gradient-to-r from-rose-500 to-rose-600 text-white shadow-sm hover:shadow-glow",
      icon: true,
    }
  }
  if (sub.status === "past_due") {
    // Paiement échoué : le libellé garde l'information critique (CTA
    // implicite "va regler ça") plutôt qu'un simple "gérer".
    return {
      label: "Paiement échoué · gérer",
      ariaLabel: "Paiement Premium échoué, gérer mon abonnement",
      className:
        "bg-orange-100 text-orange-700 border border-orange-300 dark:bg-orange-500/20 dark:text-orange-300 dark:border-orange-500/40",
      icon: false,
    }
  }
  // Any subscribed state (auto-renew on OR off) shows a single clear
  // action label. The date of expiration and the renewal toggle live
  // in the manage modal — the navbar stays succinct.
  if (sub.cancel_at_period_end) {
    return {
      label: "Gérer mon abonnement",
      ariaLabel: `Abonnement Premium actif, expire le ${formatShortDate(sub.current_period_end)}, gérer`,
      className:
        "border border-rose-500 bg-rose-50 text-rose-600 dark:bg-rose-500/10 dark:text-rose-300 dark:border-rose-400/60",
      icon: false,
    }
  }
  return {
    label: "Gérer mon abonnement",
    ariaLabel: "Abonnement Premium actif, gérer",
    className: "bg-rose-500 text-white shadow-sm hover:shadow-glow",
    icon: false,
  }
}

function BadgeSkeleton() {
  return (
    <div
      className="h-6 w-[110px] animate-shimmer rounded-full bg-slate-200 dark:bg-slate-700"
      aria-hidden="true"
    />
  )
}

function formatShortDate(iso: string): string {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return ""
  const day = String(d.getDate()).padStart(2, "0")
  const month = String(d.getMonth() + 1).padStart(2, "0")
  return `${day}/${month}`
}
