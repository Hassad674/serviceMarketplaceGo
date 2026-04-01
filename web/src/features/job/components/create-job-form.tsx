"use client"

import { useState, useEffect } from "react"
import { ChevronDown, Check } from "lucide-react"
import { useTranslations } from "next-intl"
import { useRouter } from "@i18n/navigation"
import { cn } from "@/shared/lib/utils"
import { useUser } from "@/shared/hooks/use-user"
import type { JobFormData } from "../types"
import { createDefaultJobFormData } from "../types"
import { useCreateJob } from "../hooks/use-jobs"
import { uploadVideo } from "@/features/provider/api/upload-api"
import { JobDetailsSection } from "./job-details-section"
import { BudgetSection } from "./budget-section"

type SectionId = "details" | "budget"

export function CreateJobForm() {
  const t = useTranslations("job")
  const router = useRouter()
  const createJob = useCreateJob()
  const { data: user } = useUser()
  const [formData, setFormData] = useState<JobFormData>(createDefaultJobFormData)
  const [openSections, setOpenSections] = useState<Set<SectionId>>(new Set(["details"]))
  const [error, setError] = useState<string | null>(null)

  // Agency role: force applicantType to "freelancers"
  const isAgency = user?.role === "agency"
  useEffect(() => {
    if (isAgency) {
      setFormData((prev) => ({ ...prev, applicantType: "freelancers" }))
    }
  }, [isAgency])

  function updateField<K extends keyof JobFormData>(
    field: K,
    value: JobFormData[K],
  ) {
    setFormData((prev) => ({ ...prev, [field]: value }))
  }

  function toggleSection(id: SectionId) {
    setOpenSections((prev) => {
      const next = new Set(prev)
      if (next.has(id)) {
        next.delete(id)
      } else {
        next.add(id)
      }
      return next
    })
  }

  function handleCancel() {
    router.push("/jobs")
  }

  function validate(): string | null {
    if (!formData.title.trim()) return t("errorTitleRequired")
    if (formData.descriptionType !== "video" && !formData.description.trim()) {
      return t("errorDescriptionRequired")
    }
    const minBudget = parseInt(formData.minBudget, 10)
    const maxBudget = parseInt(formData.maxBudget, 10)
    if (!minBudget || minBudget <= 0) return t("errorBudgetRequired")
    if (!maxBudget || maxBudget <= 0) return t("errorBudgetRequired")
    if (minBudget > maxBudget) return t("errorMinExceedsMax")
    return null
  }

  const [isSubmitting, setIsSubmitting] = useState(false)

  async function handleSubmit() {
    setError(null)
    const validationError = validate()
    if (validationError) {
      setError(validationError)
      return
    }

    setIsSubmitting(true)
    try {
      // Upload video file if present
      let videoUrl: string | undefined
      if (formData.videoFile) {
        const result = await uploadVideo(formData.videoFile)
        videoUrl = result.url
      }

      const minBudget = parseInt(formData.minBudget, 10)
      const maxBudget = parseInt(formData.maxBudget, 10)
      const durationWeeks = parseInt(formData.durationWeeks, 10)

      createJob.mutate(
        {
          title: formData.title.trim(),
          description: formData.description.trim(),
          skills: formData.skills,
          applicant_type: formData.applicantType,
          budget_type: formData.budgetType,
          min_budget: minBudget,
          max_budget: maxBudget,
          payment_frequency:
            formData.budgetType === "long_term"
              ? formData.paymentFrequency
              : undefined,
          duration_weeks:
            formData.budgetType === "long_term" && !formData.isIndefinite && durationWeeks > 0
              ? durationWeeks
              : undefined,
          is_indefinite:
            formData.budgetType === "long_term" ? formData.isIndefinite : false,
          description_type: formData.descriptionType,
          video_url: videoUrl,
        },
        { onSuccess: () => router.push("/jobs") },
      )
    } catch {
      setError(t("errorVideoUpload"))
    } finally {
      setIsSubmitting(false)
    }
  }

  const isDetailsComplete = formData.title.trim() !== "" && (
    formData.descriptionType === "video" || formData.description.trim() !== ""
  )
  const isBudgetComplete = formData.minBudget !== "" && formData.maxBudget !== ""

  return (
    <div className="mx-auto max-w-[680px]">
      {/* Header */}
      <div className="mb-8 flex items-center justify-between">
        <h1 className="text-2xl font-bold tracking-tight text-gray-900 dark:text-white">
          {t("createJob")}
        </h1>
        <div className="flex items-center gap-3">
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
            onClick={handleSubmit}
            disabled={createJob.isPending || isSubmitting}
            className={cn(
              "rounded-xl px-6 py-2.5 text-sm font-semibold text-white",
              "gradient-primary transition-all duration-200",
              "hover:shadow-glow active:scale-[0.98]",
              "disabled:opacity-50 disabled:cursor-not-allowed",
            )}
          >
            {(createJob.isPending || isSubmitting) ? "..." : t("publish")}
          </button>
        </div>
      </div>

      {/* Error */}
      {error && (
        <div className="mb-4 rounded-xl border border-red-200 dark:border-red-800 bg-red-50 dark:bg-red-900/20 p-3 text-sm text-red-700 dark:text-red-300">
          {error}
        </div>
      )}

      {/* Sections */}
      <div className="space-y-4">
        <AccordionSection
          number={1}
          title={t("jobDetails")}
          isOpen={openSections.has("details")}
          isComplete={isDetailsComplete}
          onToggle={() => toggleSection("details")}
        >
          <JobDetailsSection
            formData={formData}
            updateField={updateField}
            hideApplicantType={isAgency}
          />
        </AccordionSection>

        <AccordionSection
          number={2}
          title={t("budgetAndDuration")}
          isOpen={openSections.has("budget")}
          isComplete={isBudgetComplete}
          onToggle={() => toggleSection("budget")}
        >
          <BudgetSection formData={formData} updateField={updateField} />
          <div className="mt-6 flex justify-end">
            <button
              type="button"
              onClick={handleSubmit}
              disabled={createJob.isPending || isSubmitting}
              className={cn(
                "rounded-xl px-6 py-2.5 text-sm font-semibold text-white",
                "gradient-primary transition-all duration-200",
                "hover:shadow-glow active:scale-[0.98]",
                "disabled:opacity-50 disabled:cursor-not-allowed",
              )}
            >
              {(createJob.isPending || isSubmitting) ? "..." : t("publish")}
            </button>
          </div>
        </AccordionSection>
      </div>
    </div>
  )
}

/* -------------------------------------------------- */
/* Accordion section with number + validation circle  */
/* -------------------------------------------------- */

type AccordionSectionProps = {
  number: number
  title: string
  isOpen: boolean
  isComplete: boolean
  onToggle: () => void
  children: React.ReactNode
}

function AccordionSection({
  number,
  title,
  isOpen,
  isComplete,
  onToggle,
  children,
}: AccordionSectionProps) {
  return (
    <section
      className={cn(
        "rounded-2xl border transition-all duration-200",
        isOpen
          ? "border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900"
          : "border-gray-100 dark:border-gray-800 bg-white dark:bg-gray-900 hover:border-gray-200 dark:hover:border-gray-700",
      )}
    >
      <button
        type="button"
        onClick={onToggle}
        className="flex w-full items-center gap-4 px-6 py-5 text-left"
        aria-expanded={isOpen}
      >
        <div
          className={cn(
            "flex h-8 w-8 shrink-0 items-center justify-center rounded-full text-sm font-semibold transition-all duration-200",
            isComplete
              ? "bg-rose-500 text-white"
              : "border-2 border-gray-300 dark:border-gray-600 text-gray-400 dark:text-gray-500",
          )}
        >
          {isComplete ? (
            <Check className="h-4 w-4" strokeWidth={2.5} />
          ) : (
            number
          )}
        </div>
        <span className="flex-1 text-base font-semibold text-gray-900 dark:text-white">
          {title}
        </span>
        <ChevronDown
          className={cn(
            "h-5 w-5 text-gray-400 dark:text-gray-500 transition-transform duration-200",
            isOpen && "rotate-180",
          )}
          strokeWidth={1.5}
        />
      </button>

      <div
        className={cn(
          "overflow-hidden transition-all duration-300 ease-out",
          isOpen ? "max-h-[2000px] opacity-100" : "max-h-0 opacity-0",
        )}
      >
        <div className="px-6 pb-6">
          {children}
        </div>
      </div>
    </section>
  )
}
