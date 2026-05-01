"use client"

import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import type { JobFormData, BudgetType, PaymentFrequency } from "../types"
import { Button } from "@/shared/components/ui/button"

import { Input } from "@/shared/components/ui/input"
type BudgetSectionProps = {
  formData: JobFormData
  updateField: <K extends keyof JobFormData>(field: K, value: JobFormData[K]) => void
}

const BUDGET_TYPES: BudgetType[] = ["one_shot", "long_term"]
const PAYMENT_FREQUENCIES: PaymentFrequency[] = ["weekly", "monthly"]

export function BudgetSection({ formData, updateField }: BudgetSectionProps) {
  const t = useTranslations("job")

  const budgetLabelMap: Record<BudgetType, string> = {
    one_shot: t("oneShot"),
    long_term: t("longTerm"),
  }

  const frequencyLabelMap: Record<PaymentFrequency, string> = {
    weekly: t("paymentWeekly"),
    monthly: t("paymentMonthly"),
  }

  const isLongTerm = formData.budgetType === "long_term"
  const suffix = formData.paymentFrequency === "weekly" ? t("perWeek") : t("perMonth")

  return (
    <div className="space-y-5">
      {/* Budget type toggle */}
      <div>
        <label className="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-300">
          {t("budgetType")}
        </label>
        <div className="inline-flex rounded-xl border border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 p-1">
          {BUDGET_TYPES.map((type) => (
            <Button variant="ghost" size="auto"
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
            </Button>
          ))}
        </div>
      </div>

      {/* Payment frequency tabs (long-term only) */}
      {isLongTerm && (
        <div>
          <label className="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-300">
            {t("paymentFrequency")}
          </label>
          <div className="inline-flex rounded-xl border border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 p-1">
            {PAYMENT_FREQUENCIES.map((freq) => (
              <Button variant="ghost" size="auto"
                key={freq}
                type="button"
                onClick={() => updateField("paymentFrequency", freq)}
                className={cn(
                  "rounded-lg px-5 py-2 text-sm font-medium transition-all duration-200",
                  formData.paymentFrequency === freq
                    ? "bg-white dark:bg-gray-900 text-gray-900 dark:text-white shadow-sm"
                    : "text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300",
                )}
              >
                {frequencyLabelMap[freq]}
              </Button>
            ))}
          </div>
        </div>
      )}

      {/* Min / Max budget */}
      <div className="grid grid-cols-2 gap-4">
        <CurrencyInput
          label={isLongTerm ? `${t("minPayment")} (${suffix})` : t("minBudget")}
          value={formData.minBudget}
          onChange={(v) => updateField("minBudget", v)}
        />
        <CurrencyInput
          label={isLongTerm ? `${t("maxPayment")} (${suffix})` : t("maxBudget")}
          value={formData.maxBudget}
          onChange={(v) => updateField("maxBudget", v)}
        />
      </div>

      {/* Duration fields (long-term only) */}
      {isLongTerm && (
        <div className="space-y-3">
          <label className="flex cursor-pointer items-center gap-3">
            <Input
              type="checkbox"
              checked={formData.isIndefinite}
              onChange={(e) => updateField("isIndefinite", e.target.checked)}
              className="h-4 w-4 rounded border-gray-300 dark:border-gray-600 text-rose-500 focus:ring-rose-500/20"
            />
            <span className="text-sm font-medium text-gray-700 dark:text-gray-300">
              {t("indefiniteDuration")}
            </span>
          </label>
          {!formData.isIndefinite && (
            <div>
              <label className="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-300">
                {t("projectDuration")} ({formData.paymentFrequency === "weekly" ? t("durationWeeks") : t("durationMonths")})
              </label>
              <Input
                type="text"
                inputMode="numeric"
                value={formData.durationWeeks}
                onChange={(e) => updateField("durationWeeks", e.target.value)}
                placeholder={formData.paymentFrequency === "weekly" ? "12" : "6"}
                className={cn(
                  "h-12 w-full rounded-xl border border-gray-200 dark:border-gray-700",
                  "bg-gray-50 dark:bg-gray-800 px-4 text-sm",
                  "text-gray-900 dark:text-white placeholder:text-gray-400 dark:placeholder:text-gray-500",
                  "transition-all duration-200",
                  "focus:border-rose-500 focus:bg-white dark:focus:bg-gray-900 focus:outline-none focus:ring-4 focus:ring-rose-500/10",
                )}
              />
            </div>
          )}
        </div>
      )}
    </div>
  )
}

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
        <Input
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
