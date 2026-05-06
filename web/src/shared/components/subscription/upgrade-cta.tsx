"use client"

import { Sparkles } from "lucide-react"
import { cn } from "@/shared/lib/utils"

import { Button } from "@/shared/components/ui/button"
type UpgradeCtaProps = {
  variant: "inline" | "banner"
  onClick: () => void
  /** 19 for freelance, 49 for agency — in euros, no cents. */
  monthlyPrice: number
}

/**
 * Reusable call-to-action pointing at the Upgrade modal. The two
 * variants cover:
 *
 * - `banner` — horizontal rose-tinted card, shown on pages where
 *   we want the upgrade pitch to breathe (wallet, dashboard).
 * - `inline` — compact single-line version that slots cleanly
 *   under an existing card (FeePreview), without stealing focus
 *   from the primary content.
 */
export function UpgradeCta({ variant, onClick, monthlyPrice }: UpgradeCtaProps) {
  if (variant === "banner") return <BannerCta onClick={onClick} monthlyPrice={monthlyPrice} />
  return <InlineCta onClick={onClick} monthlyPrice={monthlyPrice} />
}

function BannerCta({
  onClick,
  monthlyPrice,
}: {
  onClick: () => void
  monthlyPrice: number
}) {
  return (
    <div
      className={cn(
        "flex items-center gap-4 rounded-2xl border border-primary/30 bg-gradient-to-r from-primary-soft to-white p-4",
      )}
    >
      <div className="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-xl bg-primary-soft">
        <Sparkles className="h-5 w-5 text-primary-deep" aria-hidden="true" />
      </div>
      <div className="min-w-0 flex-1">
        <p className="text-sm font-semibold text-foreground">
          Passe Premium, paie 0 € de frais
        </p>
        <p className="text-xs text-muted-foreground">
          {monthlyPrice} €/mois · rentable dès 2 missions
        </p>
      </div>
      <Button variant="ghost" size="auto"
        type="button"
        onClick={onClick}
        className={cn(
          "flex-shrink-0 rounded-full bg-gradient-to-r from-primary to-primary-deep px-4 py-2 text-xs font-semibold text-white",
          "transition-all duration-200 hover:shadow-card active:scale-[0.98]",
          "focus:outline-none focus:ring-2 focus:ring-primary/40",
        )}
      >
        Passer Premium →
      </Button>
    </div>
  )
}

function InlineCta({
  onClick,
  monthlyPrice,
}: {
  onClick: () => void
  monthlyPrice: number
}) {
  return (
    <Button variant="ghost" size="auto"
      type="button"
      onClick={onClick}
      className={cn(
        "flex w-full items-center justify-between gap-3 rounded-xl border border-primary/30 bg-primary-soft/60 px-3 py-2 text-left",
        "transition-colors duration-200 hover:bg-primary-soft hover:border-primary/40",
        "focus:outline-none focus:ring-2 focus:ring-primary/40",
      )}
    >
      <span className="flex items-center gap-2 text-xs text-foreground">
        <Sparkles className="h-3.5 w-3.5 text-primary" aria-hidden="true" />
        <span>
          Premium à <strong className="font-semibold">{monthlyPrice} €/mois</strong> → 0 € de frais
        </span>
      </span>
      <span className="text-xs font-semibold text-primary-deep">
        Passer Premium →
      </span>
    </Button>
  )
}
