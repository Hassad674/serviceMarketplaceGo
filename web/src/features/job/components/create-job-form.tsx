"use client"

import { useState } from "react"
import { ArrowLeft, ArrowRight, Check } from "lucide-react"
import { useTranslations } from "next-intl"
import { Link, useRouter } from "@i18n/navigation"
import { cn } from "@/shared/lib/utils"
import { useUser } from "@/shared/hooks/use-user"
import type { JobFormData } from "../types"
import { createDefaultJobFormData } from "../types"
import { useCreateJob } from "../hooks/use-jobs"
import { uploadVideo } from "@/shared/lib/upload-api"
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

  // Agency role: force applicantType to "freelancers". Mirroring the
  // server role into local form state happens in render via the
  // "set state during render" pattern so we avoid setState-in-effect
  // and the matching cascading-render warning.
  const isAgency = user?.role === "agency"
  const [lastIsAgency, setLastIsAgency] = useState(isAgency)
  if (lastIsAgency !== isAgency) {
    setLastIsAgency(isAgency)
    if (isAgency) {
      setFormData((prev) => ({ ...prev, applicantType: "freelancers" }))
    }
  }

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
  const isPending = createJob.isPending || isSubmitting

  return (
    <div className="mx-auto w-full max-w-[760px] px-4 py-8 sm:px-6 sm:py-12">
      {/* Back link */}
      <Link
        href="/jobs"
        className="mb-5 inline-flex items-center gap-1.5 text-[12.5px] font-medium text-muted-foreground transition-colors hover:text-foreground"
      >
        <ArrowLeft className="h-3.5 w-3.5" strokeWidth={1.8} />
        {t("createJob_w09_backToJobs")}
      </Link>

      {/* Editorial header */}
      <header className="mb-8">
        <p className="mb-2 font-mono text-[11px] font-bold uppercase tracking-[0.12em] text-primary">
          {t("createJob_w09_eyebrow")}
        </p>
        <h1 className="font-serif text-[30px] font-medium leading-[1.05] tracking-[-0.02em] text-foreground sm:text-[42px]">
          {t("createJob_w09_titlePrefix")}{" "}
          <span className="italic text-primary">{t("createJob_w09_titleAccent")}</span>
        </h1>
        <p className="mt-3 max-w-[620px] text-[15px] leading-relaxed text-muted-foreground">
          {t("createJob_w09_subtitle")}
        </p>
      </header>

      {/* Error */}
      {error && (
        <div
          role="alert"
          className="mb-6 rounded-2xl border border-destructive bg-destructive/10 px-4 py-3 text-[13.5px] text-destructive"
        >
          {error}
        </div>
      )}

      {/* Form sections */}
      <div className="space-y-4">
        <AccordionSection
          number={1}
          title={t("createJob_w09_sectionDetails")}
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
          title={t("createJob_w09_sectionBudget")}
          isOpen={openSections.has("budget")}
          isComplete={isBudgetComplete}
          onToggle={() => toggleSection("budget")}
        >
          <BudgetSection formData={formData} updateField={updateField} />
        </AccordionSection>
      </div>

      {/* Footer actions */}
      <div className="mt-8 flex flex-col-reverse items-stretch gap-3 sm:flex-row sm:items-center sm:justify-between">
        <button
          type="button"
          onClick={handleCancel}
          className={cn(
            "inline-flex items-center justify-center rounded-full px-5 py-2.5",
            "text-[13.5px] font-semibold text-muted-foreground transition-all duration-200",
            "hover:text-foreground",
            "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background",
          )}
        >
          {t("cancel")}
        </button>
        <button
          type="button"
          onClick={handleSubmit}
          disabled={isPending}
          className={cn(
            "inline-flex items-center justify-center gap-2 rounded-full",
            "bg-primary px-7 py-3 text-[13.5px] font-bold text-primary-foreground",
            "transition-all duration-200 ease-out",
            "hover:bg-primary-deep hover:shadow-[0_4px_14px_rgba(232,93,74,0.28)]",
            "active:scale-[0.98]",
            "disabled:cursor-not-allowed disabled:opacity-60",
            "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background",
          )}
        >
          {isPending ? "..." : (
            <>
              {t("createJob_w09_publishCta")}
              <ArrowRight className="h-4 w-4" strokeWidth={2} />
            </>
          )}
        </button>
      </div>
    </div>
  )
}

/* -------------------------------------------------- */
/* Accordion section — Soleil card (ivoire surface)   */
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
        "rounded-[20px] border bg-surface transition-all duration-200",
        isOpen
          ? "border-border-strong shadow-[0_2px_12px_rgba(42,31,21,0.04)]"
          : "border-border hover:border-border-strong",
      )}
    >
      <button
        type="button"
        onClick={onToggle}
        className={cn(
          "flex w-full items-center gap-4 px-6 py-5 text-left",
          "rounded-[20px] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background",
        )}
        aria-expanded={isOpen}
      >
        <span
          className={cn(
            "flex h-8 w-8 shrink-0 items-center justify-center rounded-full font-mono text-[12px] font-bold transition-all duration-200",
            isComplete
              ? "bg-primary text-primary-foreground"
              : isOpen
                ? "bg-primary-soft text-primary-deep"
                : "border border-border-strong bg-background text-muted-foreground",
          )}
        >
          {isComplete ? (
            <Check className="h-4 w-4" strokeWidth={2.5} />
          ) : (
            number
          )}
        </span>
        <span className="flex-1 font-serif text-[18px] font-medium text-foreground">
          {title}
        </span>
        <span
          aria-hidden="true"
          className={cn(
            "font-mono text-[16px] text-muted-foreground transition-transform duration-200",
            isOpen && "rotate-90",
          )}
        >
          ›
        </span>
      </button>

      <div
        className={cn(
          "overflow-hidden transition-all duration-300 ease-out",
          isOpen ? "max-h-[2400px] opacity-100" : "max-h-0 opacity-0",
        )}
      >
        <div className="border-t border-border px-6 py-6">{children}</div>
      </div>
    </section>
  )
}
