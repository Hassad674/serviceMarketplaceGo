"use client"

import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import type { ApplicantType } from "../types"

type ApplicantTypeSelectorProps = {
  value: ApplicantType
  onChange: (value: ApplicantType) => void
}

const APPLICANT_OPTIONS: ApplicantType[] = ["all", "freelancers", "agencies"]

export function ApplicantTypeSelector({ value, onChange }: ApplicantTypeSelectorProps) {
  const t = useTranslations("job")

  const labelMap: Record<ApplicantType, string> = {
    all: t("applicantAll"),
    freelancers: t("applicantFreelancers"),
    agencies: t("applicantAgencies"),
  }

  return (
    <div>
      <label className="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-300">
        {t("applicantType")}
      </label>
      <div className="space-y-2">
        {APPLICANT_OPTIONS.map((option) => (
          <label
            key={option}
            className={cn(
              "flex cursor-pointer items-center gap-3 rounded-xl border px-4 py-3",
              "transition-all duration-200",
              value === option
                ? "border-rose-500 bg-rose-50 dark:bg-rose-500/10 dark:border-rose-400"
                : "border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 hover:border-gray-300 dark:hover:border-gray-600",
            )}
          >
            <input
              type="radio"
              name="applicantType"
              value={option}
              checked={value === option}
              onChange={() => onChange(option)}
              className="h-4 w-4 border-gray-300 dark:border-gray-600 text-rose-500 focus:ring-rose-500/20"
            />
            <span
              className={cn(
                "text-sm font-medium",
                value === option
                  ? "text-rose-700 dark:text-rose-300"
                  : "text-gray-700 dark:text-gray-300",
              )}
            >
              {labelMap[option]}
            </span>
          </label>
        ))}
      </div>
    </div>
  )
}
