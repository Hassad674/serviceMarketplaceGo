"use client"

import { useState, useRef } from "react"
import {
  ArrowLeft,
  Briefcase,
  Calendar,
  Clock,
  Users,
  Ticket,
  MoreVertical,
  Flag,
  Send,
} from "lucide-react"
import { useTranslations } from "next-intl"
import { useRouter } from "@i18n/navigation"
import { cn } from "@/shared/lib/utils"
import { useHasPermission } from "@/shared/hooks/use-permissions"
import { useHasApplied } from "../hooks/use-job-applications"
import { useCredits } from "../hooks/use-jobs"
import { ApplyModal } from "./apply-modal"
import { ReportDialog } from "@/shared/components/reporting/report-dialog"
import { Portrait } from "@/shared/components/ui/portrait"
import { getJob } from "../api/job-api"
import { useQuery } from "@tanstack/react-query"
import { Button } from "@/shared/components/ui/button"

interface OpportunityDetailProps {
  jobId: string
}

// W-13 · Détail opportunité — Soleil v2 layout.
// Main column (2x): hero + budget + video + description + skills.
// Sticky sidebar (1x) on ≥lg: gradient apply CTA + poster mini-card +
// quick info card (date/applicant). Below lg the sidebar collapses
// under the main flow with the apply CTA pinned right above the modal.
//
// Reference: design/assets/sources/phase1/soleil-lotC.jsx lines 246-398.
function portraitId(seed: string): number {
  let hash = 0
  for (let i = 0; i < seed.length; i += 1) hash = (hash * 31 + seed.charCodeAt(i)) | 0
  return Math.abs(hash) % 6
}

export function OpportunityDetail({ jobId }: OpportunityDetailProps) {
  const t = useTranslations("opportunity")
  const router = useRouter()
  const tReport = useTranslations("reporting")
  const [showApplyModal, setShowApplyModal] = useState(false)
  const [showReportDialog, setShowReportDialog] = useState(false)
  const [showMenu, setShowMenu] = useState(false)
  const menuRef = useRef<HTMLDivElement>(null)

  const { data: job, isLoading } = useQuery({
    queryKey: ["jobs", jobId],
    queryFn: () => getJob(jobId),
  })
  const { data: appliedData } = useHasApplied(jobId)
  const hasAlreadyApplied = appliedData?.has_applied ?? false
  const { data: creditsData } = useCredits()
  const noCredits = creditsData?.credits === 0
  const canApply = useHasPermission("proposals.create")

  if (isLoading) {
    return (
      <div className="space-y-4">
        <div className="h-8 w-2/3 rounded-full bg-card animate-shimmer" />
        <div className="h-5 w-1/3 rounded-full bg-card animate-shimmer" />
        <div className="h-48 rounded-2xl border border-border bg-card animate-shimmer" />
      </div>
    )
  }

  if (!job) return null

  const applicantLabel =
    job.applicant_type === "all"
      ? t("allTypes")
      : job.applicant_type === "freelancers"
        ? t("freelancersOnly")
        : t("agenciesOnly")

  const kindLabel = job.budget_type === "long_term" ? t("longTerm") : t("oneShot")
  const seedId = portraitId(job.id)
  const isCtaDisabled = hasAlreadyApplied || noCredits

  return (
    <div className="space-y-6">
      {/* Back link */}
      <Button
        variant="ghost"
        size="auto"
        type="button"
        onClick={() => router.back()}
        className="inline-flex items-center gap-1.5 rounded-full px-2 py-1 text-[13px] text-muted-foreground hover:text-foreground"
      >
        <ArrowLeft className="h-4 w-4" />
        {t("back")}
      </Button>

      <div className="grid gap-6 lg:grid-cols-[minmax(0,1.7fr)_minmax(0,1fr)]">
        {/* ─── Main column ─────────────────────────────────────── */}
        <div className="space-y-5">
          {/* Header card — kind + Fraunces title + meta + report */}
          <header
            className="relative overflow-hidden rounded-2xl border border-border bg-card p-6 sm:p-8"
            style={{ boxShadow: "var(--shadow-card)" }}
          >
            <div className="flex items-start justify-between gap-4">
              <div className="min-w-0">
                <p className="mb-2 font-mono text-[11px] font-semibold uppercase tracking-[0.08em] text-muted-foreground">
                  {kindLabel}
                </p>
                <h1 className="font-serif text-[28px] font-normal leading-[1.1] tracking-[-0.025em] text-foreground sm:text-[36px]">
                  {job.title}
                </h1>
                <div className="mt-4 flex flex-wrap items-center gap-x-5 gap-y-2 text-[13px] text-muted-foreground">
                  <span className="inline-flex items-center gap-1.5">
                    <Calendar className="h-4 w-4" strokeWidth={1.6} />
                    {new Date(job.created_at).toLocaleDateString("fr-FR", {
                      day: "numeric",
                      month: "long",
                      year: "numeric",
                    })}
                  </span>
                  <span className="inline-flex items-center gap-1.5">
                    <Users className="h-4 w-4" strokeWidth={1.6} />
                    {applicantLabel}
                  </span>
                </div>
              </div>
              {/* Report menu */}
              <div className="relative shrink-0" ref={menuRef}>
                <Button
                  variant="ghost"
                  size="auto"
                  type="button"
                  onClick={() => setShowMenu((v) => !v)}
                  aria-label={tReport("report")}
                  className="flex h-9 w-9 items-center justify-center rounded-full text-muted-foreground hover:bg-background hover:text-foreground"
                >
                  <MoreVertical className="h-5 w-5" />
                </Button>
                {showMenu && (
                  <div
                    className="absolute right-0 top-full z-20 mt-1.5 w-48 overflow-hidden rounded-xl border border-border bg-card p-1"
                    style={{ boxShadow: "var(--shadow-card-strong)" }}
                  >
                    <Button
                      variant="ghost"
                      size="auto"
                      type="button"
                      onClick={() => {
                        setShowMenu(false)
                        setShowReportDialog(true)
                      }}
                      className="flex w-full items-center gap-2 rounded-lg px-3 py-2 text-sm text-primary-deep hover:bg-primary-soft"
                    >
                      <Flag className="h-4 w-4" />
                      {tReport("reportJob")}
                    </Button>
                  </div>
                )}
              </div>
            </div>
          </header>

          {/* Budget card */}
          <section
            className="rounded-2xl border border-border bg-card p-5 sm:p-6"
            style={{ boxShadow: "var(--shadow-card)" }}
          >
            <div className="flex items-center gap-4">
              <div className="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-primary-soft text-primary">
                <Briefcase className="h-5 w-5" strokeWidth={1.6} />
              </div>
              <div className="min-w-0">
                <p className="font-mono text-[11px] font-semibold uppercase tracking-[0.08em] text-muted-foreground">
                  {t("budget")}
                </p>
                <p className="font-serif text-[24px] font-medium leading-tight tracking-[-0.015em] text-foreground sm:text-[28px]">
                  {job.min_budget.toLocaleString("fr-FR")}€ — {job.max_budget.toLocaleString("fr-FR")}€
                  {job.payment_frequency && (
                    <span className="ml-1 font-sans text-[14px] font-normal text-muted-foreground">
                      / {job.payment_frequency === "weekly" ? t("weekly") : t("monthly")}
                    </span>
                  )}
                </p>
              </div>
            </div>
            {(job.duration_weeks || job.is_indefinite) && (
              <p className="mt-4 inline-flex items-center gap-1.5 text-[13px] text-muted-foreground">
                <Clock className="h-4 w-4" strokeWidth={1.6} />
                {job.is_indefinite
                  ? t("indefinite")
                  : `${job.duration_weeks} ${t("weeks")}`}
              </p>
            )}
          </section>

          {/* Video */}
          {job.video_url && (
            <section
              className="rounded-2xl border border-border bg-card p-5 sm:p-6"
              style={{ boxShadow: "var(--shadow-card)" }}
            >
              <h2 className="mb-4 font-serif text-[20px] font-medium tracking-[-0.01em] text-foreground">
                {t("watchVideo")}
              </h2>
              <div className="aspect-video max-h-[360px] overflow-hidden rounded-xl bg-foreground">
                <video
                  src={job.video_url}
                  controls
                  className="h-full w-full object-contain"
                  aria-label={job.title}
                >
                  <track kind="captions" />
                </video>
              </div>
            </section>
          )}

          {/* Description */}
          <section
            className="rounded-2xl border border-border bg-card p-5 sm:p-6"
            style={{ boxShadow: "var(--shadow-card)" }}
          >
            <h2 className="mb-4 font-serif text-[20px] font-medium tracking-[-0.01em] text-foreground">
              {t("description")}
            </h2>
            <p className="whitespace-pre-wrap break-words text-[15px] leading-[1.7] text-foreground overflow-wrap-anywhere">
              {job.description}
            </p>
          </section>

          {/* Skills */}
          {job.skills.length > 0 && (
            <section
              className="rounded-2xl border border-border bg-card p-5 sm:p-6"
              style={{ boxShadow: "var(--shadow-card)" }}
            >
              <h2 className="mb-4 font-serif text-[20px] font-medium tracking-[-0.01em] text-foreground">
                {t("requiredSkills")}
              </h2>
              <div className="flex flex-wrap gap-2">
                {job.skills.map((skill) => (
                  <span
                    key={skill}
                    className="rounded-full bg-primary-soft px-3.5 py-1.5 text-[12.5px] font-semibold text-primary-deep"
                  >
                    {skill}
                  </span>
                ))}
              </div>
            </section>
          )}
        </div>

        {/* ─── Sidebar (sticky on ≥lg) ─────────────────────────── */}
        <aside className="lg:sticky lg:top-6 lg:h-fit lg:self-start">
          <div className="space-y-4">
            {/* Apply CTA card — gradient warm coral */}
            {canApply && (
              <div
                className="relative overflow-hidden rounded-2xl border border-border p-5 sm:p-6"
                style={{
                  background: "var(--gradient-warm)",
                  boxShadow: "var(--shadow-card)",
                }}
              >
                <p className="mb-2 font-mono text-[11px] font-bold uppercase tracking-[0.1em] text-primary-deep">
                  {t("applySidebarEyebrow")}
                </p>
                <h3 className="mb-2 font-serif text-[19px] font-medium leading-[1.3] text-foreground">
                  {t("applySidebarTitle")}
                </h3>
                <p className="mb-5 text-[13px] leading-relaxed text-muted-foreground">
                  {t("applySidebarBody")}
                </p>
                <Button
                  variant="ghost"
                  size="auto"
                  type="button"
                  onClick={() => setShowApplyModal(true)}
                  disabled={isCtaDisabled}
                  className={cn(
                    "inline-flex w-full items-center justify-center gap-2 rounded-full px-4 py-3 text-[14px] font-semibold transition-all",
                    isCtaDisabled
                      ? "cursor-not-allowed bg-card/60 text-muted-foreground"
                      : "bg-primary text-white hover:bg-primary-deep active:scale-[0.98]",
                  )}
                  style={
                    !isCtaDisabled
                      ? { boxShadow: "var(--shadow-message)" }
                      : undefined
                  }
                >
                  <Send className="h-4 w-4" strokeWidth={1.8} />
                  {hasAlreadyApplied ? t("alreadyApplied") : t("apply")}
                </Button>
                {noCredits && !hasAlreadyApplied && (
                  <p className="mt-3 inline-flex items-center gap-1 text-[12px] font-medium text-primary-deep">
                    <Ticket className="h-3 w-3" strokeWidth={1.8} />
                    {t("noCreditsLeft")}
                  </p>
                )}
              </div>
            )}

            {/* Poster mini-card — Portrait + applicant + kind */}
            <div
              className="rounded-2xl border border-border bg-card p-5"
              style={{ boxShadow: "var(--shadow-card)" }}
            >
              <p className="mb-3 font-mono text-[10.5px] font-semibold uppercase tracking-[0.1em] text-subtle-foreground">
                {t("postedOn")}
              </p>
              <div className="flex items-center gap-3">
                <Portrait id={seedId} size={44} rounded="md" />
                <div className="min-w-0">
                  <p className="truncate text-[14px] font-semibold text-foreground">
                    {applicantLabel}
                  </p>
                  <p className="font-mono text-[11.5px] text-muted-foreground">
                    {new Date(job.created_at).toLocaleDateString("fr-FR", {
                      day: "numeric",
                      month: "short",
                      year: "numeric",
                    })}
                  </p>
                </div>
              </div>
            </div>
          </div>
        </aside>
      </div>

      <ApplyModal
        open={showApplyModal}
        onClose={() => setShowApplyModal(false)}
        jobId={jobId}
      />
      <ReportDialog
        open={showReportDialog}
        onClose={() => setShowReportDialog(false)}
        targetType="job"
        targetId={jobId}
      />
    </div>
  )
}
