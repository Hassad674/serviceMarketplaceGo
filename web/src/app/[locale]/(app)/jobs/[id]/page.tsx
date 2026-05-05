"use client"

import { useState, useEffect } from "react"
import { useParams } from "next/navigation"
import { useQuery } from "@tanstack/react-query"
import {
  ArrowLeft,
  Briefcase,
  Calendar,
  Users,
  Clock,
  Pencil,
  Sparkles,
} from "lucide-react"
import { useTranslations } from "next-intl"
import { useRouter, Link } from "@i18n/navigation"
import { cn } from "@/shared/lib/utils"
import { getJob } from "@/features/job/api/job-api"
import { CandidatesList } from "@/features/job/components/candidates-list"
import { useMarkApplicationsViewed } from "@/features/job/hooks/use-jobs"
import { useJobApplications } from "@/features/job/hooks/use-job-applications"

import { Button } from "@/shared/components/ui/button"
import type { JobResponse } from "@/features/job/types"

// W-07 (description) + W-08 (candidatures) — Soleil v2 layout for the
// entreprise's own listing.
//   * Editorial header with mono eyebrow + status pill + Fraunces title
//     and a ghost "Modifier l'annonce" link.
//   * Pill tab bar (Description / Candidatures (N)) — corail-soft active.
//   * Two-column grid on ≥lg: main column (description, video, skills,
//     budget) + sticky récap sidebar.
//   * Candidates tab swaps the layout for `<CandidatesList />` (Soleil
//     cards + side panel, untouched API contract).
//
// All data hooks (useQuery + useMarkApplicationsViewed) are unchanged.

type Tab = "details" | "candidates"

export default function JobDetailPage() {
  const params = useParams<{ id: string }>()
  const t = useTranslations("job")
  const router = useRouter()
  const jobId = params.id
  const [activeTab, setActiveTab] = useState<Tab>("details")
  const markViewed = useMarkApplicationsViewed()

  const { data: job, isLoading } = useQuery({
    queryKey: ["jobs", jobId],
    queryFn: () => getJob(jobId),
  })

  // The candidatures tab badge needs an accurate count even when the
  // tab is not active yet. We rely on the same hook that powers the
  // candidates list (cached, no extra request when the tab opens).
  const { data: candidatesData } = useJobApplications(jobId)
  const candidatesCount = candidatesData?.data.length ?? 0

  // Mark candidatures as viewed when the candidates tab opens.
  useEffect(() => {
    if (activeTab === "candidates" && jobId) {
      markViewed.mutate(jobId)
    }
  }, [activeTab, jobId]) // eslint-disable-line react-hooks/exhaustive-deps

  if (isLoading) {
    return (
      <div className="space-y-6 animate-shimmer">
        <div className="h-7 w-1/2 rounded-full bg-card" />
        <div className="h-10 w-2/3 rounded-2xl bg-card" />
        <div className="h-44 rounded-2xl bg-card" />
      </div>
    )
  }

  if (!job) return null

  return (
    <div className="space-y-6">
      {/* Back link */}
      <Button
        variant="ghost"
        size="auto"
        type="button"
        onClick={() => router.push("/jobs")}
        className="inline-flex items-center gap-1.5 rounded-full px-2 py-1 text-[13px] text-muted-foreground hover:text-foreground"
      >
        <ArrowLeft className="h-4 w-4" strokeWidth={1.7} />
        {t("jobDetail_w07_backToJobs")}
      </Button>

      {/* Editorial header */}
      <DetailHeader job={job} />

      {/* Pill tabs */}
      <PillTabs
        active={activeTab}
        candidatesCount={candidatesCount}
        onSelect={setActiveTab}
      />

      {activeTab === "details" ? (
        <DescriptionLayout job={job} candidatesCount={candidatesCount} />
      ) : (
        <CandidatesLayout jobId={jobId} />
      )}
    </div>
  )
}

// ─── Header ──────────────────────────────────────────────────────────

interface DetailHeaderProps {
  job: JobResponse
}

function DetailHeader({ job }: DetailHeaderProps) {
  const t = useTranslations("job")
  const isOpen = job.status === "open"
  return (
    <header className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
      <div className="min-w-0">
        <p className="font-mono text-[11px] font-bold uppercase tracking-[0.1em] text-primary-deep">
          {isOpen
            ? t("jobDetail_w07_eyebrowOpen")
            : t("jobDetail_w07_eyebrowClosed")}
        </p>
        <h1 className="mt-2 font-serif text-[30px] font-medium leading-[1.1] tracking-[-0.025em] text-foreground sm:text-[36px]">
          {job.title}
        </h1>
        <div className="mt-3 inline-flex items-center gap-2">
          <StatusPill isOpen={isOpen} />
          <span className="font-mono text-[11px] font-medium uppercase tracking-[0.06em] text-subtle-foreground">
            {new Date(job.created_at).toLocaleDateString("fr-FR", {
              day: "numeric",
              month: "long",
              year: "numeric",
            })}
          </span>
        </div>
      </div>

      <Link
        href={`/jobs/${job.id}/edit`}
        className={cn(
          "inline-flex shrink-0 items-center gap-1.5 self-start rounded-full px-4 py-2",
          "border border-border-strong bg-card text-[13px] font-medium text-foreground",
          "transition-colors duration-150 hover:bg-primary-soft hover:text-primary-deep",
          "sm:self-end",
        )}
      >
        <Pencil className="h-4 w-4" strokeWidth={1.7} />
        {t("jobDetail_w07_editLink")}
      </Link>
    </header>
  )
}

function StatusPill({ isOpen }: { isOpen: boolean }) {
  const t = useTranslations("job")
  return (
    <span
      className={cn(
        "inline-flex items-center rounded-full px-2.5 py-0.5",
        "text-[11px] font-bold leading-tight",
        isOpen
          ? "bg-success-soft text-success"
          : "bg-border text-muted-foreground",
      )}
    >
      {isOpen ? t("statusOpen") : t("statusClosed")}
    </span>
  )
}

// ─── Pill tab bar ────────────────────────────────────────────────────

interface PillTabsProps {
  active: Tab
  candidatesCount: number
  onSelect: (tab: Tab) => void
}

function PillTabs({ active, candidatesCount, onSelect }: PillTabsProps) {
  const t = useTranslations("job")
  return (
    <div
      role="tablist"
      aria-label={t("jobDetail_w07_tabDescription")}
      className={cn(
        "inline-flex items-center gap-1 rounded-full border border-border bg-card p-1",
      )}
    >
      <PillTab
        label={t("jobDetail_w07_tabDescription")}
        isActive={active === "details"}
        onClick={() => onSelect("details")}
      />
      <PillTab
        label={`${t("jobDetail_w07_tabCandidates")} (${candidatesCount})`}
        isActive={active === "candidates"}
        onClick={() => onSelect("candidates")}
      />
    </div>
  )
}

interface PillTabProps {
  label: string
  isActive: boolean
  onClick: () => void
}

function PillTab({ label, isActive, onClick }: PillTabProps) {
  return (
    <Button
      variant="ghost"
      size="auto"
      type="button"
      role="tab"
      aria-selected={isActive}
      onClick={onClick}
      className={cn(
        "rounded-full px-4 py-2 text-[13px] font-semibold transition-colors duration-150",
        isActive
          ? "bg-primary-soft text-primary-deep"
          : "text-muted-foreground hover:text-foreground",
      )}
    >
      {label}
    </Button>
  )
}

// ─── Description tab layout (W-07) ──────────────────────────────────

function DescriptionLayout({
  job,
  candidatesCount,
}: {
  job: JobResponse
  candidatesCount: number
}) {
  return (
    <div className="grid gap-6 lg:grid-cols-[minmax(0,1.7fr)_minmax(0,1fr)]">
      <DescriptionMain job={job} />
      <SummarySidebar job={job} candidatesCount={candidatesCount} />
    </div>
  )
}

function DescriptionMain({ job }: { job: JobResponse }) {
  const t = useTranslations("job")
  return (
    <div className="space-y-5">
      {job.video_url && (
        <SectionCard heading={t("jobDetail_w07_videoSection")}>
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
        </SectionCard>
      )}

      <SectionCard heading={t("jobDetail_w07_descSection")}>
        <p className="whitespace-pre-wrap break-words text-[15px] leading-[1.7] text-foreground overflow-wrap-anywhere">
          {job.description}
        </p>
      </SectionCard>

      {job.skills.length > 0 && (
        <SectionCard heading={t("jobDetail_w07_skillsSection")}>
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
        </SectionCard>
      )}

      <BudgetCard job={job} />
    </div>
  )
}

interface SectionCardProps {
  heading: string
  children: React.ReactNode
}

function SectionCard({ heading, children }: SectionCardProps) {
  return (
    <section
      className="rounded-2xl border border-border bg-card p-5 sm:p-6"
      style={{ boxShadow: "var(--shadow-card)" }}
    >
      <h2 className="mb-4 font-serif text-[20px] font-medium tracking-[-0.01em] text-foreground">
        {heading}
      </h2>
      {children}
    </section>
  )
}

function BudgetCard({ job }: { job: JobResponse }) {
  const t = useTranslations("job")
  const kindLabel =
    job.budget_type === "long_term" ? t("longTerm") : t("oneShot")
  const frequencyLabel =
    job.payment_frequency === "weekly"
      ? t("jobDetail_w07_paymentWeekly")
      : job.payment_frequency === "monthly"
        ? t("jobDetail_w07_paymentMonthly")
        : null
  return (
    <section
      className="rounded-2xl border border-border bg-card p-5 sm:p-6"
      style={{ boxShadow: "var(--shadow-card)" }}
    >
      <div className="flex items-start gap-4">
        <div className="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-primary-soft text-primary">
          <Briefcase className="h-5 w-5" strokeWidth={1.6} />
        </div>
        <div className="min-w-0 flex-1">
          <p className="font-mono text-[11px] font-semibold uppercase tracking-[0.08em] text-muted-foreground">
            {kindLabel}
          </p>
          <p className="mt-1 font-serif text-[24px] font-medium leading-tight tracking-[-0.015em] text-foreground sm:text-[28px]">
            {job.min_budget.toLocaleString("fr-FR")} €
            <span className="mx-1 text-muted-foreground">—</span>
            {job.max_budget.toLocaleString("fr-FR")} €
            {frequencyLabel && (
              <span className="ml-2 font-sans text-[13px] font-normal text-muted-foreground">
                {frequencyLabel}
              </span>
            )}
          </p>
          {(job.duration_weeks || job.is_indefinite) && (
            <p className="mt-3 inline-flex items-center gap-1.5 text-[13px] text-muted-foreground">
              <Clock className="h-4 w-4" strokeWidth={1.6} />
              {job.is_indefinite
                ? t("jobDetail_w07_durationIndefinite")
                : t("jobDetail_w07_durationWeeks", {
                    count: job.duration_weeks ?? 0,
                  })}
            </p>
          )}
        </div>
      </div>
    </section>
  )
}

// ─── Summary sidebar ────────────────────────────────────────────────

function SummarySidebar({
  job,
  candidatesCount,
}: {
  job: JobResponse
  candidatesCount: number
}) {
  const t = useTranslations("job")
  const applicantTypeLabel =
    job.applicant_type === "freelancers"
      ? t("applicantFreelancers")
      : job.applicant_type === "agencies"
        ? t("applicantAgencies")
        : t("applicantAll")
  const kindLabel =
    job.budget_type === "long_term"
      ? t("longTermShort")
      : t("oneShotShort")

  return (
    <aside className="lg:sticky lg:top-6 lg:h-fit lg:self-start">
      <div
        className="relative overflow-hidden rounded-2xl border border-border p-5 sm:p-6"
        style={{
          background: "var(--gradient-warm)",
          boxShadow: "var(--shadow-card)",
        }}
      >
        <p className="font-mono text-[11px] font-bold uppercase tracking-[0.1em] text-primary-deep">
          <Sparkles
            className="mr-1 inline h-3 w-3"
            strokeWidth={1.7}
            aria-hidden="true"
          />
          {t("jobDetail_w07_sidebarHeading")}
        </p>
        <dl className="mt-4 space-y-3.5">
          <SummaryRow
            label={t("jobDetail_w07_statApplicants")}
            value={t("jobDetail_w07_statApplicantsValue", {
              count: candidatesCount,
            })}
            icon={<Users className="h-3.5 w-3.5" strokeWidth={1.7} />}
          />
          <SummaryRow
            label={t("jobDetail_w07_statBudget")}
            value={`${job.min_budget.toLocaleString("fr-FR")} € — ${job.max_budget.toLocaleString("fr-FR")} €`}
            icon={<Briefcase className="h-3.5 w-3.5" strokeWidth={1.7} />}
          />
          <SummaryRow
            label={t("jobDetail_w07_statKind")}
            value={kindLabel}
            icon={<Clock className="h-3.5 w-3.5" strokeWidth={1.7} />}
          />
          <SummaryRow
            label={t("jobDetail_w07_statApplicantType")}
            value={applicantTypeLabel}
            icon={<Users className="h-3.5 w-3.5" strokeWidth={1.7} />}
          />
          <SummaryRow
            label={t("jobDetail_w07_statPublished")}
            value={new Date(job.created_at).toLocaleDateString("fr-FR", {
              day: "numeric",
              month: "long",
              year: "numeric",
            })}
            icon={<Calendar className="h-3.5 w-3.5" strokeWidth={1.7} />}
          />
        </dl>
      </div>
    </aside>
  )
}

interface SummaryRowProps {
  label: string
  value: string
  icon: React.ReactNode
}

function SummaryRow({ label, value, icon }: SummaryRowProps) {
  return (
    <div className="flex items-start justify-between gap-3 border-b border-dashed border-border-strong/40 pb-3 last:border-b-0 last:pb-0">
      <dt className="inline-flex items-center gap-1.5 text-[12px] font-medium uppercase tracking-[0.06em] text-muted-foreground">
        <span aria-hidden="true">{icon}</span>
        {label}
      </dt>
      <dd className="text-right text-[13px] font-semibold text-foreground">
        {value}
      </dd>
    </div>
  )
}

// ─── Candidates tab layout (W-08) ───────────────────────────────────

function CandidatesLayout({ jobId }: { jobId: string }) {
  return (
    <div className="space-y-4">
      <CandidatesList jobId={jobId} />
    </div>
  )
}
