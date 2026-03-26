"use client"

import { useState } from "react"
import { useTranslations } from "next-intl"
import { useRouter } from "@i18n/navigation"
import { cn } from "@/shared/lib/utils"
import type { ProjectFormData } from "../types"
import { createDefaultFormData } from "../types"
import { PaymentTypeSelector } from "./payment-type-selector"
import { MilestoneEditor } from "./milestone-editor"
import { SkillsInput } from "./skills-input"
import { ProjectPreview } from "./project-preview"
import { StructureSection } from "./structure-section"
import { DetailsSection } from "./details-section"
import { TimelineSection } from "./timeline-section"
import { ApplicantSection } from "./applicant-section"

export function CreateProjectForm() {
  const t = useTranslations("projects")
  const router = useRouter()
  const [formData, setFormData] = useState<ProjectFormData>(createDefaultFormData)

  function updateField<K extends keyof ProjectFormData>(
    field: K,
    value: ProjectFormData[K],
  ) {
    setFormData((prev) => ({ ...prev, [field]: value }))
  }

  function handleCancel() {
    router.push("/projects")
  }

  function handlePublish() {
    // Frontend-only: no API call, just log
    console.log("Publishing project:", formData)
  }

  return (
    <div className="mx-auto max-w-6xl">
      <div className="flex flex-col gap-8 lg:flex-row">
        {/* Left: Form */}
        <div className="flex-1 space-y-8 min-w-0">
          {/* Section 1: Payment type */}
          <PaymentTypeSelector
            value={formData.paymentType}
            onChange={(v) => updateField("paymentType", v)}
          />

          {/* Section 2: Structure */}
          <StructureSection formData={formData} updateField={updateField} />

          {/* Section 3: Details */}
          <DetailsSection formData={formData} updateField={updateField} />

          {/* Section 4: Timeline */}
          <TimelineSection formData={formData} updateField={updateField} />

          {/* Section 5: Who can apply */}
          <ApplicantSection formData={formData} updateField={updateField} />

          {/* Bottom actions */}
          <div className="flex items-center justify-end gap-3 border-t border-gray-200 dark:border-gray-700 pt-6">
            <button
              type="button"
              onClick={handleCancel}
              className={cn(
                "rounded-xl px-5 py-2.5 text-sm font-medium",
                "text-gray-600 dark:text-gray-400 transition-all duration-200",
                "hover:bg-gray-100 dark:hover:bg-gray-800 hover:text-gray-900 dark:hover:text-white",
              )}
            >
              {t("cancel")}
            </button>
            <button
              type="button"
              onClick={handlePublish}
              className={cn(
                "rounded-xl px-6 py-2.5 text-sm font-semibold text-white",
                "gradient-primary transition-all duration-200",
                "hover:shadow-glow active:scale-[0.98]",
              )}
            >
              {t("publish")}
            </button>
          </div>
        </div>

        {/* Right: Preview */}
        <div className="hidden w-[340px] shrink-0 lg:block">
          <ProjectPreview formData={formData} />
        </div>
      </div>
    </div>
  )
}
