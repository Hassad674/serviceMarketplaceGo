"use client"

import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import { Button } from "@/shared/components/ui/button"
import type { StatsPeriodDays } from "../api/stats-api"

// PeriodSelector is the segmented control that drives the /stats deep
// dive. Four day values — 7 / 30 / 90 / 365 — match the backend's
// allowed set; sending any other value would 400. The control is
// keyboard accessible (Tab + Enter / Space) and announces the current
// state to screen readers via `aria-pressed`.
//
// 365 is rendered as "1 an" (i18n pluralisation handles the singular
// year form). D3 added this option so the user can scan a long-tail
// trend without the chart noise of a 30-day window.

interface PeriodSelectorProps {
  value: StatsPeriodDays
  onChange: (next: StatsPeriodDays) => void
}

const OPTIONS: StatsPeriodDays[] = [7, 30, 90, 365]

export function PeriodSelector({ value, onChange }: PeriodSelectorProps) {
  const t = useTranslations("stats.period")
  return (
    <div
      className="inline-flex items-center gap-1 rounded-full border border-border bg-card p-1"
      role="group"
      aria-label={t("label")}
    >
      {OPTIONS.map((option) => {
        const active = option === value
        return (
          <Button
            key={option}
            type="button"
            variant="ghost"
            size="auto"
            aria-pressed={active}
            onClick={() => onChange(option)}
            className={cn(
              "rounded-full px-3.5 py-1.5 text-[13px] font-medium transition-colors",
              active
                ? "bg-primary text-primary-foreground hover:bg-primary"
                : "text-muted-foreground hover:bg-muted/50",
            )}
          >
            {t("days", { days: option })}
          </Button>
        )
      })}
    </div>
  )
}
