"use client"

import { useState } from "react"
import { ChevronDown, Check, ArrowLeft, Loader2 } from "lucide-react"
import { useTranslations } from "next-intl"
import { useRouter } from "@i18n/navigation"
import { cn } from "@/shared/lib/utils"
import { useUser } from "@/shared/hooks/use-user"
import type {
  JobFormData,
  JobResponse,
  DescriptionType,
  ApplicantType,
  BudgetType,
  PaymentFrequency,
} from "../types"
import { useUpdateJob } from "../hooks/use-jobs"
import { uploadVideo } from "@/shared/lib/upload-api"
import { JobDetailsSection } from "./job-details-section"
import { BudgetSection } from "./budget-section"
import { Button } from "@/shared/components/ui/button"

// W-07-edit · Edit job form — Soleil v2 chrome.
// Form behaviour, react-hook-form + zod + mutations preserved as-is.
// `JobDetailsSection`, `BudgetSection` and `ApplicantTypeSelector`
// are sibling-owned (creation lane) and rendered AS-IS so they auto
// inherit the sibling's Soleil styling once both PRs land.

type EditJobFormProps = {
  job: JobResponse
}

function jobToFormData(job: JobResponse): JobFormData {
  return {
    title: job.title,
    description: job.description,
    skills: job.skills ?? [],
    applicantType: job.applicant_type as ApplicantType,
    budgetType: job.budget_type as BudgetType,
    minBudget: String(job.min_budget),
    maxBudget: String(job.max_budget),
    paymentFrequency:
      (job.payment_frequency as PaymentFrequency) ?? "monthly",
    durationWeeks: job.duration_weeks ? String(job.duration_weeks) : "",
    isIndefinite: job.is_indefinite,
    descriptionType: job.description_type as DescriptionType,
    videoUrl: job.video_url ?? "",
    videoFile: null,
  }
}

type SectionId = "details" | "budget"

export function EditJobForm({ job }: EditJobFormProps) {
  const t = useTranslations("job")
  const router = useRouter()
  const updateJob = useUpdateJob(job.id)
  const { data: user } = useUser()
  const [formData, setFormData] = useState<JobFormData>(() =>
    jobToFormData(job),
  )
  const [openSections, setOpenSections] = useState<Set<SectionId>>(
    new Set(["details", "budget"]),
  )
  const [error, setError] = useState<string | null>(null)
  const [isSubmitting, setIsSubmitting] = useState(false)

  // Mirror the server role into local form state during render to avoid
  // setState-in-effect cascading renders. See create-job-form for the
  // matching pattern.
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
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  function handleCancel() {
    router.push(`/jobs/${job.id}`)
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

  async function handleSubmit() {
    setError(null)
    const validationError = validate()
    if (validationError) {
      setError(validationError)
      return
    }

    setIsSubmitting(true)
    try {
      let videoUrl: string | undefined = formData.videoUrl || undefined
      if (formData.videoFile) {
        const result = await uploadVideo(formData.videoFile)
        videoUrl = result.url
      }

      const minBudget = parseInt(formData.minBudget, 10)
      const maxBudget = parseInt(formData.maxBudget, 10)
      const durationWeeks = parseInt(formData.durationWeeks, 10)

      updateJob.mutate(
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
            formData.budgetType === "long_term" &&
            !formData.isIndefinite &&
            durationWeeks > 0
              ? durationWeeks
              : undefined,
          is_indefinite:
            formData.budgetType === "long_term" ? formData.isIndefinite : false,
          description_type: formData.descriptionType,
          video_url: videoUrl,
        },
        { onSuccess: () => router.push(`/jobs/${job.id}`) },
      )
    } catch {
      setError(t("errorVideoUpload"))
    } finally {
      setIsSubmitting(false)
    }
  }

  const isDetailsComplete =
    formData.title.trim() !== "" &&
    (formData.descriptionType === "video" || formData.description.trim() !== "")
  const isBudgetComplete = formData.minBudget !== "" && formData.maxBudget !== ""
  const isPending = updateJob.isPending || isSubmitting

  return (
    <div className="mx-auto max-w-[760px] space-y-6">
      {/* Back link */}
      <Button
        variant="ghost"
        size="auto"
        type="button"
        onClick={handleCancel}
        className="inline-flex items-center gap-1.5 rounded-full px-2 py-1 text-[13px] text-muted-foreground hover:text-foreground"
      >
        <ArrowLeft className="h-4 w-4" strokeWidth={1.7} />
        {t("title")}
      </Button>

      {/* Editorial header */}
      <header className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
        <div className="min-w-0">
          <p className="font-mono text-[11px] font-bold uppercase tracking-[0.1em] text-primary-deep">
            {t("jobDetail_w07edit_eyebrow")}
          </p>
          <h1 className="mt-2 font-serif text-[30px] font-medium leading-[1.1] tracking-[-0.025em] text-foreground sm:text-[36px]">
            {t("jobDetail_w07edit_titlePrefix")}{" "}
            <span className="italic text-primary">
              {t("jobDetail_w07edit_titleAccent")}
            </span>
          </h1>
          <p className="mt-3 max-w-xl text-[14px] leading-relaxed text-muted-foreground">
            {t("jobDetail_w07edit_subtitle")}
          </p>
        </div>
        <div className="flex shrink-0 items-center gap-2 sm:self-end">
          <Button
            variant="ghost"
            size="auto"
            type="button"
            onClick={handleCancel}
            className={cn(
              "inline-flex items-center justify-center rounded-full px-4 py-2",
              "border border-border-strong bg-card text-[13px] font-medium text-foreground",
              "transition-colors duration-150 hover:bg-primary-soft hover:text-primary-deep",
            )}
          >
            {t("jobDetail_w07edit_cancel")}
          </Button>
          <Button
            variant="ghost"
            size="auto"
            type="button"
            onClick={handleSubmit}
            disabled={isPending}
            className={cn(
              "inline-flex items-center justify-center gap-1.5 rounded-full px-5 py-2",
              "bg-primary text-[13px] font-semibold text-white",
              "transition-colors duration-150 hover:bg-primary-deep active:scale-[0.98]",
              "disabled:cursor-not-allowed disabled:bg-border-strong disabled:text-muted-foreground",
            )}
            style={
              isPending ? undefined : { boxShadow: "var(--shadow-message)" }
            }
          >
            {isPending && (
              <Loader2 className="h-4 w-4 animate-spin" strokeWidth={1.7} />
            )}
            {isPending
              ? t("jobDetail_w07edit_saving")
              : t("jobDetail_w07edit_save")}
          </Button>
        </div>
      </header>

      {/* Error banner */}
      {error && (
        <div
          role="alert"
          className={cn(
            "rounded-2xl border border-primary-deep/30 bg-primary-soft/70 px-4 py-3",
            "text-[13.5px] font-medium text-primary-deep",
          )}
        >
          {error}
        </div>
      )}

      {/* Sections */}
      <div className="space-y-4">
        <SoleilAccordion
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
        </SoleilAccordion>

        <SoleilAccordion
          number={2}
          title={t("budgetAndDuration")}
          isOpen={openSections.has("budget")}
          isComplete={isBudgetComplete}
          onToggle={() => toggleSection("budget")}
        >
          <BudgetSection formData={formData} updateField={updateField} />
          <div className="mt-6 flex justify-end">
            <Button
              variant="ghost"
              size="auto"
              type="button"
              onClick={handleSubmit}
              disabled={isPending}
              className={cn(
                "inline-flex items-center justify-center gap-1.5 rounded-full px-5 py-2.5",
                "bg-primary text-[13px] font-semibold text-white",
                "transition-colors duration-150 hover:bg-primary-deep active:scale-[0.98]",
                "disabled:cursor-not-allowed disabled:bg-border-strong disabled:text-muted-foreground",
              )}
              style={
                isPending ? undefined : { boxShadow: "var(--shadow-message)" }
              }
            >
              {isPending && (
                <Loader2 className="h-4 w-4 animate-spin" strokeWidth={1.7} />
              )}
              {isPending
                ? t("jobDetail_w07edit_saving")
                : t("jobDetail_w07edit_save")}
            </Button>
          </div>
        </SoleilAccordion>
      </div>
    </div>
  )
}

// ─── Soleil-styled accordion ────────────────────────────────────────

type SoleilAccordionProps = {
  number: number
  title: string
  isOpen: boolean
  isComplete: boolean
  onToggle: () => void
  children: React.ReactNode
}

function SoleilAccordion({
  number,
  title,
  isOpen,
  isComplete,
  onToggle,
  children,
}: SoleilAccordionProps) {
  return (
    <section
      className={cn(
        "rounded-2xl border bg-card transition-colors duration-150",
        isOpen ? "border-border-strong" : "border-border hover:border-border-strong",
      )}
      style={{ boxShadow: "var(--shadow-card)" }}
    >
      <Button
        variant="ghost"
        size="auto"
        type="button"
        onClick={onToggle}
        className="flex w-full items-center gap-4 px-5 py-4 text-left sm:px-6 sm:py-5"
        aria-expanded={isOpen}
      >
        <div
          className={cn(
            "flex h-8 w-8 shrink-0 items-center justify-center rounded-full",
            "font-mono text-[12.5px] font-semibold transition-colors duration-150",
            isComplete
              ? "bg-primary text-white"
              : "border-2 border-border-strong bg-card text-muted-foreground",
          )}
        >
          {isComplete ? (
            <Check className="h-4 w-4" strokeWidth={2.2} />
          ) : (
            number
          )}
        </div>
        <span className="flex-1 font-serif text-[18px] font-medium tracking-[-0.005em] text-foreground">
          {title}
        </span>
        <ChevronDown
          className={cn(
            "h-5 w-5 text-muted-foreground transition-transform duration-200",
            isOpen && "rotate-180",
          )}
          strokeWidth={1.6}
        />
      </Button>

      <div
        className={cn(
          "overflow-hidden transition-all duration-300 ease-out",
          isOpen ? "max-h-[2400px] opacity-100" : "max-h-0 opacity-0",
        )}
      >
        <div className="px-5 pb-6 sm:px-6">{children}</div>
      </div>
    </section>
  )
}
