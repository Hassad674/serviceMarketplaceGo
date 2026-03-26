"use client"

import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import type { ProjectFormData, ApplicantType } from "../types"

type ApplicantSectionProps = {
  formData: ProjectFormData
  updateField: <K extends keyof ProjectFormData>(field: K, value: ProjectFormData[K]) => void
}

const APPLICANT_OPTIONS: ApplicantType[] = ["all", "freelancers", "agencies"]

export function ApplicantSection({ formData, updateField }: ApplicantSectionProps) {
  const t = useTranslations("projects")

  const labelMap: Record<ApplicantType, string> = {
    all: t("freelancersAndAgencies"),
    freelancers: t("freelancersOnly"),
    agencies: t("agenciesOnly"),
  }

  return (
    <section className="space-y-4">
      <h2 className="text-lg font-semibold text-gray-900 dark:text-white">
        {t("whoCanApply")}
      </h2>

      {/* Radio buttons */}
      <div className="space-y-2">
        {APPLICANT_OPTIONS.map((option) => (
          <label
            key={option}
            className={cn(
              "flex cursor-pointer items-center gap-3 rounded-xl border px-4 py-3",
              "transition-all duration-200",
              formData.applicantType === option
                ? "border-rose-500 bg-rose-50 dark:bg-rose-500/10 dark:border-rose-400"
                : "border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 hover:border-gray-300 dark:hover:border-gray-600",
            )}
          >
            <input
              type="radio"
              name="applicantType"
              value={option}
              checked={formData.applicantType === option}
              onChange={() => updateField("applicantType", option)}
              className="h-4 w-4 border-gray-300 dark:border-gray-600 text-rose-500 focus:ring-rose-500/20"
            />
            <span
              className={cn(
                "text-sm font-medium",
                formData.applicantType === option
                  ? "text-rose-700 dark:text-rose-300"
                  : "text-gray-700 dark:text-gray-300",
              )}
            >
              {labelMap[option]}
            </span>
          </label>
        ))}
      </div>

      {/* Negotiable checkbox */}
      <label className="flex items-center gap-2.5 cursor-pointer">
        <input
          type="checkbox"
          checked={formData.isNegotiable}
          onChange={(e) => updateField("isNegotiable", e.target.checked)}
          className={cn(
            "h-4 w-4 rounded border-gray-300 dark:border-gray-600",
            "text-rose-500 focus:ring-rose-500/20",
          )}
        />
        <span className="text-sm text-gray-700 dark:text-gray-300">
          {t("negotiable")}
        </span>
      </label>
    </section>
  )
}
