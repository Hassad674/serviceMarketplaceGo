"use client"

import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import { CountrySelect } from "./country-select"
import type { PaymentInfoFormData } from "../types"

type BusinessInfoSectionProps = {
  data: PaymentInfoFormData
  onChange: (field: keyof PaymentInfoFormData, value: string) => void
}

function InputField({
  label,
  value,
  onChange,
  placeholder,
  required,
}: {
  label: string
  value: string
  onChange: (v: string) => void
  placeholder?: string
  required?: boolean
}) {
  return (
    <div>
      <label className="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-300">
        {label}
        {required && <span className="ml-0.5 text-red-500">*</span>}
      </label>
      <input
        type="text"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
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

export function BusinessInfoSection({ data, onChange }: BusinessInfoSectionProps) {
  const t = useTranslations("paymentInfo")

  return (
    <section className="animate-slide-up rounded-2xl border border-gray-100 bg-white p-6 shadow-sm dark:border-gray-800 dark:bg-gray-900">
      <h2 className="mb-4 text-lg font-semibold text-gray-900 dark:text-white">
        {t("businessInfo")}
      </h2>

      <div className="grid gap-4 sm:grid-cols-2">
        <div className="sm:col-span-2">
          <InputField
            label={t("businessName")}
            value={data.businessName}
            onChange={(v) => onChange("businessName", v)}
            required
          />
        </div>
        <InputField
          label={t("businessAddress")}
          value={data.businessAddress}
          onChange={(v) => onChange("businessAddress", v)}
          required
        />
        <InputField
          label={t("businessCity")}
          value={data.businessCity}
          onChange={(v) => onChange("businessCity", v)}
          required
        />
        <InputField
          label={t("businessPostalCode")}
          value={data.businessPostalCode}
          onChange={(v) => onChange("businessPostalCode", v)}
          required
        />
        <div>
          <label className="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-300">
            {t("businessCountry")}
            <span className="ml-0.5 text-red-500">*</span>
          </label>
          <CountrySelect
            value={data.businessCountry}
            onChange={(v) => onChange("businessCountry", v)}
          />
        </div>
        <InputField
          label={t("taxId")}
          value={data.taxId}
          onChange={(v) => onChange("taxId", v)}
          placeholder={t("taxIdPlaceholder")}
          required
        />
        <InputField
          label={t("vatNumber")}
          value={data.vatNumber}
          onChange={(v) => onChange("vatNumber", v)}
          placeholder={t("vatNumberPlaceholder")}
        />
      </div>
    </section>
  )
}
