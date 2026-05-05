"use client"

import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import type { JobFormData, BudgetType, PaymentFrequency } from "../types"

// W-09 — Budget & durée section, Soleil v2 visual port.
//
// Public prop interface (`formData`, `updateField`) is intentionally
// unchanged: the sibling agent's `edit-job-form.tsx` consumes this same
// component and must keep working unmodified.

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
    <div className="space-y-6">
      {/* Budget type — segmented pills */}
      <div>
        <p className="mb-2 font-mono text-[11px] font-bold uppercase tracking-[0.08em] text-muted-foreground">
          {t("budgetType")}
        </p>
        <div
          className="inline-flex rounded-full border border-border bg-background p-1"
          role="radiogroup"
          aria-label={t("budgetType")}
        >
          {BUDGET_TYPES.map((type) => {
            const isActive = formData.budgetType === type
            return (
              <button
                key={type}
                type="button"
                role="radio"
                aria-checked={isActive}
                onClick={() => updateField("budgetType", type)}
                className={cn(
                  "rounded-full px-5 py-2 text-[13px] font-semibold transition-all duration-200",
                  "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background",
                  isActive
                    ? "bg-primary text-primary-foreground shadow-[0_2px_8px_rgba(232,93,74,0.18)]"
                    : "text-muted-foreground hover:text-foreground",
                )}
              >
                {budgetLabelMap[type]}
              </button>
            )
          })}
        </div>
      </div>

      {/* Payment frequency tabs (long-term only) */}
      {isLongTerm && (
        <div>
          <p className="mb-2 font-mono text-[11px] font-bold uppercase tracking-[0.08em] text-muted-foreground">
            {t("paymentFrequency")}
          </p>
          <div
            className="inline-flex rounded-full border border-border bg-background p-1"
            role="radiogroup"
            aria-label={t("paymentFrequency")}
          >
            {PAYMENT_FREQUENCIES.map((freq) => {
              const isActive = formData.paymentFrequency === freq
              return (
                <button
                  key={freq}
                  type="button"
                  role="radio"
                  aria-checked={isActive}
                  onClick={() => updateField("paymentFrequency", freq)}
                  className={cn(
                    "rounded-full px-5 py-2 text-[13px] font-semibold transition-all duration-200",
                    "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background",
                    isActive
                      ? "bg-primary text-primary-foreground shadow-[0_2px_8px_rgba(232,93,74,0.18)]"
                      : "text-muted-foreground hover:text-foreground",
                  )}
                >
                  {frequencyLabelMap[freq]}
                </button>
              )
            })}
          </div>
        </div>
      )}

      {/* Min / Max budget */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
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
          <label className="flex cursor-pointer items-center gap-3 select-none">
            <input
              type="checkbox"
              checked={formData.isIndefinite}
              onChange={(e) => updateField("isIndefinite", e.target.checked)}
              className="h-4 w-4 rounded border-border-strong text-primary focus:ring-2 focus:ring-ring focus:ring-offset-2 focus:ring-offset-background"
            />
            <span className="text-[14px] font-medium text-foreground">
              {t("indefiniteDuration")}
            </span>
          </label>
          {!formData.isIndefinite && (
            <div>
              <p className="mb-2 font-mono text-[11px] font-bold uppercase tracking-[0.08em] text-muted-foreground">
                {t("projectDuration")} ({formData.paymentFrequency === "weekly" ? t("durationWeeks") : t("durationMonths")})
              </p>
              <input
                type="text"
                inputMode="numeric"
                value={formData.durationWeeks}
                onChange={(e) => updateField("durationWeeks", e.target.value)}
                placeholder={formData.paymentFrequency === "weekly" ? "12" : "6"}
                className={cn(
                  "h-12 w-full rounded-2xl border border-border-strong bg-surface px-4",
                  "font-mono text-[14px] text-foreground placeholder:text-muted-foreground",
                  "transition-all duration-200",
                  "focus:border-primary focus:outline-none focus:ring-4 focus:ring-primary-soft",
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
      <p className="mb-2 font-mono text-[11px] font-bold uppercase tracking-[0.08em] text-muted-foreground">
        {label}
      </p>
      <div className="relative">
        <input
          type="text"
          inputMode="decimal"
          value={value}
          onChange={(e) => onChange(e.target.value)}
          className={cn(
            "h-12 w-full rounded-2xl border border-border-strong bg-surface pl-4 pr-10",
            "font-mono text-[14px] text-foreground placeholder:text-muted-foreground",
            "transition-all duration-200",
            "focus:border-primary focus:outline-none focus:ring-4 focus:ring-primary-soft",
          )}
        />
        <span
          aria-hidden="true"
          className="pointer-events-none absolute right-4 top-1/2 -translate-y-1/2 font-serif text-[14px] text-muted-foreground"
        >
          &euro;
        </span>
      </div>
    </div>
  )
}
