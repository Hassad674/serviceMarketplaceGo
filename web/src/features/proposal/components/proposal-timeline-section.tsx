"use client"

import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import type { ProposalFormData } from "../types"

type ProposalTimelineSectionProps = {
  formData: ProposalFormData
  updateField: <K extends keyof ProposalFormData>(field: K, value: ProposalFormData[K]) => void
}

export function ProposalTimelineSection({ formData, updateField }: ProposalTimelineSectionProps) {
  const t = useTranslations("proposal")

  return (
    <section className="space-y-4">
      <h2 className="text-lg font-semibold text-gray-900 dark:text-white">
        Timeline
      </h2>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        {/* Start date */}
        <div>
          <label className="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-300">
            {t("startDate")}
          </label>
          <input
            type="date"
            value={formData.startDate}
            onChange={(e) => updateField("startDate", e.target.value)}
            className={cn(
              "h-12 w-full rounded-xl border border-gray-200 dark:border-gray-700",
              "bg-gray-50 dark:bg-gray-800 px-4 text-sm",
              "text-gray-900 dark:text-white",
              "transition-all duration-200",
              "focus:border-rose-500 focus:bg-white dark:focus:bg-gray-900 focus:outline-none focus:ring-4 focus:ring-rose-500/10",
            )}
          />
        </div>

        {/* Deadline */}
        <div>
          <label className="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-300">
            {t("deadline")}
          </label>
          <input
            type="date"
            value={formData.deadline}
            onChange={(e) => updateField("deadline", e.target.value)}
            disabled={formData.isOngoing}
            className={cn(
              "h-12 w-full rounded-xl border border-gray-200 dark:border-gray-700",
              "bg-gray-50 dark:bg-gray-800 px-4 text-sm",
              "text-gray-900 dark:text-white",
              "transition-all duration-200",
              "focus:border-rose-500 focus:bg-white dark:focus:bg-gray-900 focus:outline-none focus:ring-4 focus:ring-rose-500/10",
              formData.isOngoing && "cursor-not-allowed opacity-50",
            )}
          />
        </div>
      </div>

      {/* Ongoing checkbox */}
      <label className="flex items-center gap-2.5 cursor-pointer">
        <input
          type="checkbox"
          checked={formData.isOngoing}
          onChange={(e) => {
            updateField("isOngoing", e.target.checked)
            if (e.target.checked) updateField("deadline", "")
          }}
          className={cn(
            "h-4 w-4 rounded border-gray-300 dark:border-gray-600",
            "text-rose-500 focus:ring-rose-500/20",
          )}
        />
        <span className="text-sm text-gray-700 dark:text-gray-300">
          {t("ongoing")}
        </span>
      </label>

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
