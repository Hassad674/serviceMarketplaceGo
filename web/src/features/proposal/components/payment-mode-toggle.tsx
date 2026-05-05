"use client"

import { cn } from "@/shared/lib/utils"
import { useTranslations } from "next-intl"
import type { PaymentMode } from "../types"
import { Button } from "@/shared/components/ui/button"

// Soleil v2 — Payment-mode 2-card segmented toggle.
// One-shot vs phased: corail-soft active, ivoire off. Tablist semantics
// preserved for the existing keyboard / a11y contract.

type PaymentModeToggleProps = {
  value: PaymentMode
  onChange: (mode: PaymentMode) => void
  disabled?: boolean
}

export function PaymentModeToggle({
  value,
  onChange,
  disabled = false,
}: PaymentModeToggleProps) {
  const t = useTranslations("proposal.paymentMode")

  return (
    <div className="space-y-3">
      <p className="font-mono text-[11px] font-bold uppercase tracking-[0.1em] text-primary">
        {t("label")}
      </p>
      <div
        className="grid grid-cols-1 gap-3 sm:grid-cols-2"
        role="tablist"
        aria-label={t("label")}
      >
        <PaymentModeCard
          mode="one_time"
          active={value === "one_time"}
          disabled={disabled}
          onClick={() => onChange("one_time")}
          label={t("oneTime")}
          hint={t("oneTimeHint")}
        />
        <PaymentModeCard
          mode="milestone"
          active={value === "milestone"}
          disabled={disabled}
          onClick={() => onChange("milestone")}
          label={t("milestone")}
          hint={t("milestoneHint")}
        />
      </div>
    </div>
  )
}

type PaymentModeCardProps = {
  mode: PaymentMode
  active: boolean
  disabled: boolean
  onClick: () => void
  label: string
  hint: string
}

function PaymentModeCard({
  mode,
  active,
  disabled,
  onClick,
  label,
  hint,
}: PaymentModeCardProps) {
  return (
    <Button
      variant="ghost"
      size="auto"
      type="button"
      role="tab"
      aria-selected={active}
      aria-controls={`payment-mode-panel-${mode}`}
      onClick={onClick}
      disabled={disabled}
      className={cn(
        "group relative flex h-auto w-full flex-col items-start gap-1 rounded-2xl border px-5 py-4 text-left",
        "transition-all duration-200 ease-out",
        "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background",
        active
          ? "border-primary bg-primary-soft"
          : "border-border bg-card hover:border-border-strong",
        disabled && "cursor-not-allowed opacity-60",
      )}
    >
      <span
        className={cn(
          "font-serif text-[16px] font-medium tracking-[-0.01em]",
          active ? "text-primary-deep" : "text-foreground",
        )}
      >
        {label}
      </span>
      <span
        className={cn(
          "text-[12.5px] leading-snug",
          active ? "text-primary-deep/80" : "text-muted-foreground",
        )}
      >
        {hint}
      </span>
      <span
        aria-hidden="true"
        className={cn(
          "absolute right-4 top-4 block h-3 w-3 rounded-full border transition-colors duration-200",
          active ? "border-primary bg-primary" : "border-border",
        )}
      />
    </Button>
  )
}
