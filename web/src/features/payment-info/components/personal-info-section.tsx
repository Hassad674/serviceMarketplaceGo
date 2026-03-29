"use client"

import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import { CountrySelect } from "./country-select"
import type { PaymentInfoFormData, BusinessRole } from "../types"

const ACTIVITY_SECTORS = [
  { mcc: "7372", labelKey: "sectorDev" },
  { mcc: "7333", labelKey: "sectorDesign" },
  { mcc: "7311", labelKey: "sectorMarketing" },
  { mcc: "7392", labelKey: "sectorConsulting" },
  { mcc: "7339", labelKey: "sectorAdmin" },
  { mcc: "7221", labelKey: "sectorPhoto" },
  { mcc: "7338", labelKey: "sectorWriting" },
  { mcc: "8299", labelKey: "sectorTraining" },
  { mcc: "8931", labelKey: "sectorAccounting" },
  { mcc: "8911", labelKey: "sectorEngineering" },
  { mcc: "8111", labelKey: "sectorLegal" },
  { mcc: "8099", labelKey: "sectorHealth" },
  { mcc: "8999", labelKey: "sectorOther" },
]

const BUSINESS_ROLES: { value: BusinessRole; labelKey: string }[] = [
  { value: "owner", labelKey: "roleOwner" },
  { value: "ceo", labelKey: "roleCeo" },
  { value: "director", labelKey: "roleDirector" },
  { value: "partner", labelKey: "rolePartner" },
  { value: "other", labelKey: "roleOther" },
]

type PersonalInfoSectionProps = {
  data: PaymentInfoFormData
  onChange: (field: keyof PaymentInfoFormData, value: string) => void
}

function InputField({
  label,
  value,
  onChange,
  type = "text",
  placeholder,
  required,
}: {
  label: string
  value: string
  onChange: (v: string) => void
  type?: string
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
        type={type}
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

export function PersonalInfoSection({ data, onChange }: PersonalInfoSectionProps) {
  const t = useTranslations("paymentInfo")
  const sectionTitle = data.isBusiness ? t("legalRepresentative") : t("personalInfo")

  return (
    <section className="rounded-2xl border border-gray-100 bg-white p-6 shadow-sm dark:border-gray-800 dark:bg-gray-900">
      <h2 className="mb-4 text-lg font-semibold text-gray-900 dark:text-white">
        {sectionTitle}
      </h2>

      <div className="grid gap-4 sm:grid-cols-2">
        <InputField
          label={t("firstName")}
          value={data.firstName}
          onChange={(v) => onChange("firstName", v)}
          required
        />
        <InputField
          label={t("lastName")}
          value={data.lastName}
          onChange={(v) => onChange("lastName", v)}
          required
        />
        <InputField
          label={t("dateOfBirth")}
          value={data.dateOfBirth}
          onChange={(v) => onChange("dateOfBirth", v)}
          type="date"
          required
        />
        <div>
          <label className="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-300">
            {t("nationality")}
            <span className="ml-0.5 text-red-500">*</span>
          </label>
          <CountrySelect
            value={data.nationality}
            onChange={(v) => onChange("nationality", v)}
          />
        </div>
        <InputField
          label={t("address")}
          value={data.address}
          onChange={(v) => onChange("address", v)}
          required
        />
        <InputField
          label={t("city")}
          value={data.city}
          onChange={(v) => onChange("city", v)}
          required
        />
        <InputField
          label={t("postalCode")}
          value={data.postalCode}
          onChange={(v) => onChange("postalCode", v)}
          required
        />
        <InputField
          label={t("phone")}
          value={data.phone}
          onChange={(v) => onChange("phone", v)}
          type="tel"
          placeholder="+33 6 12 34 56 78"
          required
        />
        <div>
          <label className="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-300">
            {t("activitySector")}
            <span className="ml-0.5 text-red-500">*</span>
          </label>
          <select
            value={data.activitySector}
            onChange={(e) => onChange("activitySector", e.target.value)}
            aria-label={t("activitySector")}
            className={cn(
              "h-10 w-full rounded-lg border border-gray-200 bg-white px-3 text-sm shadow-xs transition-all duration-200",
              "focus:border-rose-500 focus:ring-4 focus:ring-rose-500/10 focus:outline-none",
              "dark:border-gray-600 dark:bg-gray-800 dark:text-white",
            )}
          >
            {ACTIVITY_SECTORS.map((s) => (
              <option key={s.mcc} value={s.mcc}>{t(s.labelKey)}</option>
            ))}
          </select>
        </div>

        {/* Business role — only visible when isBusiness is true */}
        {data.isBusiness && (
          <div className="sm:col-span-2">
            <label className="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-300">
              {t("yourRole")}
              <span className="ml-0.5 text-red-500">*</span>
            </label>
            <select
              value={data.businessRole}
              onChange={(e) => onChange("businessRole", e.target.value)}
              aria-label={t("yourRole")}
              className={cn(
                "h-10 w-full rounded-lg border border-gray-200 bg-white px-3 text-sm shadow-xs transition-all duration-200",
                "focus:border-rose-500 focus:ring-4 focus:ring-rose-500/10 focus:outline-none",
                "dark:border-gray-700 dark:bg-gray-900 dark:text-gray-100",
              )}
            >
              <option value="">{t("yourRole")}</option>
              {BUSINESS_ROLES.map((r) => (
                <option key={r.value} value={r.value}>
                  {t(r.labelKey)}
                </option>
              ))}
            </select>
          </div>
        )}
      </div>
    </section>
  )
}
