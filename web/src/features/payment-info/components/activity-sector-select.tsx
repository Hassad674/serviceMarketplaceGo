"use client"

import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import { ACTIVITY_SECTORS, BUSINESS_ROLES } from "../types"

interface ActivitySectorSelectProps {
  value: string
  onChange: (value: string) => void
  isBusiness: boolean
  businessRole: string
  onBusinessRoleChange: (value: string) => void
}

export function ActivitySectorSelect({
  value, onChange, isBusiness, businessRole, onBusinessRoleChange,
}: ActivitySectorSelectProps) {
  const t = useTranslations("paymentInfo")

  return (
    <section className="rounded-2xl border border-gray-100 bg-white p-6 shadow-sm dark:border-gray-800 dark:bg-gray-900">
      <h2 className="mb-4 text-lg font-semibold text-gray-900 dark:text-white">
        {t("activitySector")}
      </h2>
      <div className="grid gap-4 sm:grid-cols-2">
        <div>
          <label className="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-300">
            {t("activitySector")}
            <span className="ml-0.5 text-red-500">*</span>
          </label>
          <select
            value={value}
            onChange={(e) => onChange(e.target.value)}
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

        {isBusiness && (
          <div>
            <label className="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-300">
              {t("yourRole")}
              <span className="ml-0.5 text-red-500">*</span>
            </label>
            <select
              value={businessRole}
              onChange={(e) => onBusinessRoleChange(e.target.value)}
              aria-label={t("yourRole")}
              className={cn(
                "h-10 w-full rounded-lg border border-gray-200 bg-white px-3 text-sm shadow-xs transition-all duration-200",
                "focus:border-rose-500 focus:ring-4 focus:ring-rose-500/10 focus:outline-none",
                "dark:border-gray-700 dark:bg-gray-900 dark:text-gray-100",
              )}
            >
              <option value="">{t("yourRole")}</option>
              {BUSINESS_ROLES.map((r) => (
                <option key={r.value} value={r.value}>{t(r.labelKey)}</option>
              ))}
            </select>
          </div>
        )}
      </div>
    </section>
  )
}
