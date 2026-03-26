"use client"

import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import type { ProposalFormData } from "../types"
import { ProposalSkillsInput } from "./proposal-skills-input"

const TITLE_MAX_LENGTH = 100

type ProposalDetailsSectionProps = {
  formData: ProposalFormData
  updateField: <K extends keyof ProposalFormData>(field: K, value: ProposalFormData[K]) => void
}

export function ProposalDetailsSection({ formData, updateField }: ProposalDetailsSectionProps) {
  const t = useTranslations("proposal")

  return (
    <section className="space-y-4">
      <h2 className="text-lg font-semibold text-gray-900 dark:text-white">
        {t("details")}
      </h2>

      {/* Title */}
      <div>
        <div className="mb-1.5 flex items-center justify-between">
          <label className="text-sm font-medium text-gray-700 dark:text-gray-300">
            {t("proposalTitle")}
          </label>
          <span
            className={cn(
              "text-xs tabular-nums",
              formData.title.length > TITLE_MAX_LENGTH
                ? "text-red-500"
                : "text-gray-400 dark:text-gray-500",
            )}
          >
            {formData.title.length}/{TITLE_MAX_LENGTH}
          </span>
        </div>
        <input
          type="text"
          value={formData.title}
          onChange={(e) => updateField("title", e.target.value)}
          maxLength={TITLE_MAX_LENGTH}
          placeholder={t("proposalTitlePlaceholder")}
          className={cn(
            "h-12 w-full rounded-xl border border-gray-200 dark:border-gray-700",
            "bg-gray-50 dark:bg-gray-800 px-4 text-sm",
            "text-gray-900 dark:text-white placeholder:text-gray-400 dark:placeholder:text-gray-500",
            "transition-all duration-200",
            "focus:border-rose-500 focus:bg-white dark:focus:bg-gray-900 focus:outline-none focus:ring-4 focus:ring-rose-500/10",
          )}
        />
      </div>

      {/* Description */}
      <div>
        <label className="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-300">
          {t("proposalDescription")}
        </label>
        <textarea
          value={formData.description}
          onChange={(e) => updateField("description", e.target.value)}
          placeholder={t("proposalDescPlaceholder")}
          rows={4}
          className={cn(
            "w-full rounded-xl border border-gray-200 dark:border-gray-700",
            "bg-gray-50 dark:bg-gray-800 px-4 py-3 text-sm",
            "text-gray-900 dark:text-white placeholder:text-gray-400 dark:placeholder:text-gray-500",
            "resize-none transition-all duration-200",
            "focus:border-rose-500 focus:bg-white dark:focus:bg-gray-900 focus:outline-none focus:ring-4 focus:ring-rose-500/10",
          )}
        />
      </div>

      {/* Skills */}
      <ProposalSkillsInput
        skills={formData.skills}
        onChange={(s) => updateField("skills", s)}
      />
    </section>
  )
}
