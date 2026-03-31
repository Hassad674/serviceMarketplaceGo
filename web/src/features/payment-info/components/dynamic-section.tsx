"use client"

import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import { CountrySelect } from "./country-select"
import type { FieldSection, FieldSpec } from "../api/payment-info-api"

interface DynamicSectionProps {
  section: FieldSection
  values: Record<string, string>
  onChange: (key: string, value: string) => void
}

export function DynamicSection({ section, values, onChange }: DynamicSectionProps) {
  const t = useTranslations("paymentInfo")

  return (
    <section className="rounded-2xl border border-gray-100 bg-white p-6 shadow-sm dark:border-gray-800 dark:bg-gray-900">
      <h2 className="mb-4 text-lg font-semibold text-gray-900 dark:text-white">
        {t(section.title_key)}
      </h2>
      <div className="grid gap-4 sm:grid-cols-2">
        {section.fields.map((field) => (
          <DynamicField
            key={field.key}
            field={field}
            value={values[field.key] ?? ""}
            onChange={(v) => onChange(field.key, v)}
          />
        ))}
      </div>
    </section>
  )
}

interface DynamicFieldProps {
  field: FieldSpec
  value: string
  onChange: (value: string) => void
}

function DynamicField({ field, value, onChange }: DynamicFieldProps) {
  const t = useTranslations("paymentInfo")
  const label = t(field.label_key)

  if (field.type === "select") {
    return <SelectField field={field} value={value} onChange={onChange} label={label} />
  }

  return (
    <div>
      <label className="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-300">
        {label}
        {field.required && <span className="ml-0.5 text-red-500">*</span>}
      </label>
      <input
        type={inputType(field.type)}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={field.placeholder ?? ""}
        className={cn(
          "h-10 w-full rounded-lg border border-gray-200 bg-white px-3 text-sm shadow-xs transition-all duration-200",
          "placeholder:text-gray-400",
          "focus:border-rose-500 focus:ring-4 focus:ring-rose-500/10 focus:outline-none",
          "dark:border-gray-700 dark:bg-gray-900 dark:text-gray-100 dark:placeholder:text-gray-500",
        )}
      />
    </div>
  )
}

function SelectField({ field, value, onChange, label }: DynamicFieldProps & { label: string }) {
  if (field.label_key === "nationality" || field.label_key === "country" || field.label_key === "bankCountry") {
    return (
      <div>
        <label className="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-300">
          {label}
          {field.required && <span className="ml-0.5 text-red-500">*</span>}
        </label>
        <CountrySelect value={value} onChange={onChange} />
      </div>
    )
  }

  if (field.label_key === "politicalExposure") {
    return (
      <div>
        <label className="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-300">
          {label}
        </label>
        <select
          value={value}
          onChange={(e) => onChange(e.target.value)}
          className={cn(
            "h-10 w-full rounded-lg border border-gray-200 bg-white px-3 text-sm shadow-xs transition-all duration-200",
            "focus:border-rose-500 focus:ring-4 focus:ring-rose-500/10 focus:outline-none",
            "dark:border-gray-600 dark:bg-gray-800 dark:text-white",
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
      <label className="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-300">
        {label}
      </label>
      <input
        type="text"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={field.placeholder ?? ""}
        className={cn(
          "h-10 w-full rounded-lg border border-gray-200 bg-white px-3 text-sm shadow-xs transition-all duration-200",
          "placeholder:text-gray-400",
          "focus:border-rose-500 focus:ring-4 focus:ring-rose-500/10 focus:outline-none",
          "dark:border-gray-700 dark:bg-gray-900 dark:text-gray-100 dark:placeholder:text-gray-500",
        )}
      />
    </div>
  )
}

/** Map our field types to HTML input types. */
function inputType(fieldType: string): string {
  switch (fieldType) {
    case "email": return "email"
    case "phone": return "tel"
    case "date": return "date"
    default: return "text"
  }
}
