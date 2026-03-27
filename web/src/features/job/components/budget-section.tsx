"use client"

import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import type { JobFormData, BudgetType } from "../types"

type BudgetSectionProps = {
  formData: JobFormData
  updateField: <K extends keyof JobFormData>(field: K, value: JobFormData[K]) => void
}

const BUDGET_TYPES: BudgetType[] = ["one_shot", "long_term"]

export function BudgetSection({ formData, updateField }: BudgetSectionProps) {
  const t = useTranslations("job")

  const budgetLabelMap: Record<BudgetType, string> = {
    one_shot: t("oneShot"),
    long_term: t("longTerm"),
  }

  return (
    <div className="space-y-5">
      {/* Budget type toggle */}
      <div>
        <label className="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-300">
          {t("budgetType")}
        </label>
        <div className="inline-flex rounded-xl border border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 p-1">
          {BUDGET_TYPES.map((type) => (
            <button
              key={type}
              type="button"
              onClick={() => updateField("budgetType", type)}
              className={cn(
                "rounded-lg px-5 py-2 text-sm font-medium transition-all duration-200",
                formData.budgetType === type
                  ? "bg-white dark:bg-gray-900 text-gray-900 dark:text-white shadow-sm"
                  : "text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300",
              )}
            >
              {budgetLabelMap[type]}
            </button>
          ))}
        </div>
      </div>

      {/* Min / Max budget */}
      <div className="grid grid-cols-2 gap-4">
        <CurrencyInput
          label={t("minBudget")}
          value={formData.minBudget}
          onChange={(v) => updateField("minBudget", v)}
        />
        <CurrencyInput
          label={t("maxBudget")}
          value={formData.maxBudget}
          onChange={(v) => updateField("maxBudget", v)}
        />
      </div>
    </div>
  )
}

/* -------------------------------------------------- */
/* Currency input with EUR symbol                     */
/* -------------------------------------------------- */

type CurrencyInputProps = {
  label: string
  value: string
  onChange: (value: string) => void
}

function CurrencyInput({ label, value, onChange }: CurrencyInputProps) {
  return (
    <div>
      <label className="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-300">
        {label}
      </label>
      <div className="relative">
        <span className="pointer-events-none absolute left-4 top-1/2 -translate-y-1/2 text-sm font-medium text-gray-400 dark:text-gray-500">
          &euro;
        </span>
        <input
          type="text"
          inputMode="decimal"
          value={value}
          onChange={(e) => onChange(e.target.value)}
          className={cn(
            "h-12 w-full rounded-xl border border-gray-200 dark:border-gray-700",
            "bg-gray-50 dark:bg-gray-800 text-sm pl-9 pr-4",
            "text-gray-900 dark:text-white placeholder:text-gray-400 dark:placeholder:text-gray-500",
            "transition-all duration-200",
            "focus:border-rose-500 focus:bg-white dark:focus:bg-gray-900 focus:outline-none focus:ring-4 focus:ring-rose-500/10",
          )}
        />
      </div>
    </div>
  )
}
