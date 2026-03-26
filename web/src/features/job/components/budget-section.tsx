"use client"

import { Minus, Plus } from "lucide-react"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import type { JobFormData, BudgetType, PaymentFrequency } from "../types"

type BudgetSectionProps = {
  formData: JobFormData
  updateField: <K extends keyof JobFormData>(field: K, value: JobFormData[K]) => void
}

const BUDGET_TYPES: BudgetType[] = ["ongoing", "one_time"]
const PAYMENT_FREQUENCIES: PaymentFrequency[] = ["hourly", "weekly", "monthly"]

const RATE_SUFFIX: Record<PaymentFrequency, string> = {
  hourly: "/hr",
  weekly: "/wk",
  monthly: "/mo",
}

export function BudgetSection({ formData, updateField }: BudgetSectionProps) {
  const t = useTranslations("job")

  const budgetLabelMap: Record<BudgetType, string> = {
    ongoing: t("ongoing"),
    one_time: t("oneTime"),
  }

  const frequencyLabelMap: Record<PaymentFrequency, string> = {
    hourly: t("hourly"),
    weekly: t("weekly"),
    monthly: t("monthly"),
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

      {/* Ongoing fields */}
      {formData.budgetType === "ongoing" && (
        <>
          {/* Payment frequency */}
          <div>
            <label className="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-300">
              {t("paymentFrequency")}
            </label>
            <div className="inline-flex rounded-xl border border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 p-1">
              {PAYMENT_FREQUENCIES.map((freq) => (
                <button
                  key={freq}
                  type="button"
                  onClick={() => updateField("paymentFrequency", freq)}
                  className={cn(
                    "rounded-lg px-4 py-2 text-sm font-medium transition-all duration-200",
                    formData.paymentFrequency === freq
                      ? "bg-white dark:bg-gray-900 text-gray-900 dark:text-white shadow-sm"
                      : "text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300",
                  )}
                >
                  {frequencyLabelMap[freq]}
                </button>
              ))}
            </div>
          </div>

          {/* Min / Max rate */}
          <div className="grid grid-cols-2 gap-4">
            <CurrencyInput
              label={t("minRate")}
              value={formData.minRate}
              onChange={(v) => updateField("minRate", v)}
              suffix={RATE_SUFFIX[formData.paymentFrequency]}
            />
            <CurrencyInput
              label={t("maxRate")}
              value={formData.maxRate}
              onChange={(v) => updateField("maxRate", v)}
              suffix={RATE_SUFFIX[formData.paymentFrequency]}
            />
          </div>

          {/* Max hours per week (if hourly) */}
          {formData.paymentFrequency === "hourly" && (
            <HoursInput
              label={t("maxHoursPerWeek")}
              value={formData.maxHoursPerWeek}
              onChange={(v) => updateField("maxHoursPerWeek", v)}
            />
          )}
        </>
      )}

      {/* One-time fields */}
      {formData.budgetType === "one_time" && (
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
      )}

      {/* Estimated duration */}
      <div>
        <label className="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-300">
          {t("estimatedDuration")}
        </label>
        <div className="flex items-center gap-3">
          <input
            type="text"
            value={formData.isIndefinite ? "" : formData.estimatedDuration}
            onChange={(e) => updateField("estimatedDuration", e.target.value)}
            disabled={formData.isIndefinite}
            placeholder="ex. 3"
            className={cn(
              "h-12 w-24 rounded-xl border border-gray-200 dark:border-gray-700",
              "bg-gray-50 dark:bg-gray-800 px-4 text-sm text-center",
              "text-gray-900 dark:text-white placeholder:text-gray-400 dark:placeholder:text-gray-500",
              "transition-all duration-200",
              "focus:border-rose-500 focus:bg-white dark:focus:bg-gray-900 focus:outline-none focus:ring-4 focus:ring-rose-500/10",
              formData.isIndefinite && "opacity-50 cursor-not-allowed",
            )}
          />
          <span className="text-sm text-gray-500 dark:text-gray-400">
            {t("months")}
          </span>
        </div>
        <label className="mt-2 flex items-center gap-2.5 cursor-pointer">
          <input
            type="checkbox"
            checked={formData.isIndefinite}
            onChange={(e) => updateField("isIndefinite", e.target.checked)}
            className={cn(
              "h-4 w-4 rounded border-gray-300 dark:border-gray-600",
              "text-rose-500 focus:ring-rose-500/20",
            )}
          />
          <span className="text-sm text-gray-700 dark:text-gray-300">
            {t("indefiniteDuration")}
          </span>
        </label>
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
  suffix?: string
}

function CurrencyInput({ label, value, onChange, suffix }: CurrencyInputProps) {
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
            "bg-gray-50 dark:bg-gray-800 text-sm",
            "text-gray-900 dark:text-white placeholder:text-gray-400 dark:placeholder:text-gray-500",
            "transition-all duration-200",
            "focus:border-rose-500 focus:bg-white dark:focus:bg-gray-900 focus:outline-none focus:ring-4 focus:ring-rose-500/10",
            suffix ? "pl-9 pr-12" : "pl-9 pr-4",
          )}
        />
        {suffix && (
          <span className="pointer-events-none absolute right-4 top-1/2 -translate-y-1/2 text-sm text-gray-400 dark:text-gray-500">
            {suffix}
          </span>
        )}
      </div>
    </div>
  )
}

/* -------------------------------------------------- */
/* Hours input with +/- buttons                       */
/* -------------------------------------------------- */

type HoursInputProps = {
  label: string
  value: number
  onChange: (value: number) => void
}

const MIN_HOURS = 1
const MAX_HOURS = 80

function HoursInput({ label, value, onChange }: HoursInputProps) {
  function decrement() {
    if (value > MIN_HOURS) onChange(value - 1)
  }

  function increment() {
    if (value < MAX_HOURS) onChange(value + 1)
  }

  return (
    <div>
      <label className="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-300">
        {label}
      </label>
      <div className="inline-flex items-center gap-3 rounded-xl border border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 px-3 py-2">
        <button
          type="button"
          onClick={decrement}
          disabled={value <= MIN_HOURS}
          className={cn(
            "flex h-8 w-8 items-center justify-center rounded-lg transition-all duration-200",
            value <= MIN_HOURS
              ? "cursor-not-allowed text-gray-300 dark:text-gray-600"
              : "text-gray-600 dark:text-gray-300 hover:bg-gray-200 dark:hover:bg-gray-700 active:scale-95",
          )}
          aria-label="Decrease hours"
        >
          <Minus className="h-4 w-4" strokeWidth={2} />
        </button>
        <span className="min-w-[2rem] text-center text-sm font-semibold tabular-nums text-gray-900 dark:text-white">
          {value}
        </span>
        <button
          type="button"
          onClick={increment}
          disabled={value >= MAX_HOURS}
          className={cn(
            "flex h-8 w-8 items-center justify-center rounded-lg transition-all duration-200",
            value >= MAX_HOURS
              ? "cursor-not-allowed text-gray-300 dark:text-gray-600"
              : "text-gray-600 dark:text-gray-300 hover:bg-gray-200 dark:hover:bg-gray-700 active:scale-95",
          )}
          aria-label="Increase hours"
        >
          <Plus className="h-4 w-4" strokeWidth={2} />
        </button>
      </div>
    </div>
  )
}
