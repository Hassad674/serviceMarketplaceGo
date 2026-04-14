"use client"

import { cn } from "@/shared/lib/utils"
import { useTranslations } from "next-intl"
import type { PaymentMode } from "../types"

type PaymentModeToggleProps = {
  value: PaymentMode
  onChange: (mode: PaymentMode) => void
  disabled?: boolean
}

/**
 * PaymentModeToggle is the segmented control at the top of the
 * create-proposal form. It lets the client choose between a
 * single-payment "one-time" mission and a multi-step "milestone"
 * project.
 *
 * The backend treats both modes identically (every proposal has at
 * least one milestone) — this toggle only swaps the form layout
 * between the simple amount field and the multi-milestone editor.
 */
export function PaymentModeToggle({
  value,
  onChange,
  disabled = false,
}: PaymentModeToggleProps) {
  const t = useTranslations("proposal.paymentMode")

  return (
    <div className="space-y-2">
      <p className="text-sm font-medium text-gray-700 dark:text-gray-300">
        {t("label")}
      </p>
      <div
        className={cn(
          "inline-flex rounded-xl border border-gray-200 bg-gray-50 p-1",
          "dark:border-gray-700 dark:bg-gray-800",
        )}
        role="tablist"
        aria-label={t("label")}
      >
        <PaymentModeButton
          mode="one_time"
          active={value === "one_time"}
          disabled={disabled}
          onClick={() => onChange("one_time")}
          label={t("oneTime")}
        />
        <PaymentModeButton
          mode="milestone"
          active={value === "milestone"}
          disabled={disabled}
          onClick={() => onChange("milestone")}
          label={t("milestone")}
        />
      </div>
      <p className="text-xs text-gray-500 dark:text-gray-400">
        {value === "one_time" ? t("oneTimeHint") : t("milestoneHint")}
      </p>
    </div>
  )
}

type PaymentModeButtonProps = {
  mode: PaymentMode
  active: boolean
  disabled: boolean
  onClick: () => void
  label: string
}

function PaymentModeButton({
  mode,
  active,
  disabled,
  onClick,
  label,
}: PaymentModeButtonProps) {
  return (
    <button
      type="button"
      role="tab"
      aria-selected={active}
      aria-controls={`payment-mode-panel-${mode}`}
      onClick={onClick}
      disabled={disabled}
      className={cn(
        "relative rounded-lg px-5 py-2 text-sm font-medium transition-all duration-200",
        "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-rose-500/50",
        active
          ? "bg-white text-gray-900 shadow-sm dark:bg-gray-700 dark:text-white"
          : "text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200",
        disabled && "cursor-not-allowed opacity-50",
      )}
    >
      {label}
    </button>
  )
}
