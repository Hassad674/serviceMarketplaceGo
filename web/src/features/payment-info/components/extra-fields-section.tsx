"use client"

import { Settings } from "lucide-react"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import type { FieldSpec } from "../api/payment-info-api"

interface ExtraFieldsSectionProps {
  fields: FieldSpec[]
  values: Record<string, string>
  onChange: (key: string, value: string) => void
}

export function ExtraFieldsSection({ fields, values, onChange }: ExtraFieldsSectionProps) {
  const t = useTranslations("paymentInfo")

  if (fields.length === 0) return null

  return (
    <div className="rounded-2xl border border-slate-100 bg-white shadow-sm dark:border-slate-700 dark:bg-slate-800/80 overflow-hidden">
      <div className="h-1 bg-gradient-to-r from-amber-500 to-orange-500" />
      <div className="p-6 space-y-4">
        <div className="flex items-center gap-3">
          <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-amber-100 dark:bg-amber-500/20">
            <Settings className="h-5 w-5 text-amber-600 dark:text-amber-400" strokeWidth={1.5} />
          </div>
          <div>
            <h2 className="text-base font-semibold text-slate-900 dark:text-white">
              {t("countrySpecificFields")}
            </h2>
            <p className="text-xs text-slate-500 dark:text-slate-400">
              {t("countrySpecificFieldsDesc")}
            </p>
          </div>
        </div>

        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          {fields.map((field) => (
            <ExtraField
              key={field.key}
              field={field}
              value={values[field.key] ?? ""}
              onChange={(v) => onChange(field.key, v)}
            />
          ))}
        </div>
      </div>
    </div>
  )
}

function ExtraField({ field, value, onChange }: {
  field: FieldSpec
  value: string
  onChange: (value: string) => void
}) {
  const t = useTranslations("paymentInfo")
  const label = getFieldLabel(field.key, t)

  if (field.type === "select" && field.key === "political_exposure") {
    return (
      <div>
        <label className="mb-1 block text-sm font-medium text-slate-700 dark:text-slate-300">
          {label}
        </label>
        <select
          value={value}
          onChange={(e) => onChange(e.target.value)}
          className={cn(
            "h-10 w-full rounded-lg border border-slate-200 bg-white px-3 text-sm",
            "focus:border-rose-500 focus:ring-4 focus:ring-rose-500/10 focus:outline-none",
            "dark:border-slate-600 dark:bg-slate-800 dark:text-white",
          )}
        >
          <option value="">--</option>
          <option value="none">None</option>
          <option value="existing">Existing</option>
        </select>
      </div>
    )
  }

  return (
    <div>
      <label className="mb-1 block text-sm font-medium text-slate-700 dark:text-slate-300">
        {label}
      </label>
      <input
        type="text"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={field.placeholder ?? ""}
        className={cn(
          "h-10 w-full rounded-lg border border-slate-200 bg-white px-3 text-sm",
          "focus:border-rose-500 focus:ring-4 focus:ring-rose-500/10 focus:outline-none",
          "dark:border-slate-600 dark:bg-slate-800 dark:text-white",
        )}
      />
    </div>
  )
}

function getFieldLabel(key: string, t: (key: string) => string): string {
  const labelMap: Record<string, string> = {
    id_number: t("idNumber"),
    ssn_last_4: t("ssnLast4"),
    state: t("stateProvince"),
    political_exposure: t("politicalExposure"),
    first_name_kana: t("firstNameKana"),
    last_name_kana: t("lastNameKana"),
    first_name_kanji: t("firstNameKanji"),
    last_name_kanji: t("lastNameKanji"),
  }
  return labelMap[key] ?? key
}
