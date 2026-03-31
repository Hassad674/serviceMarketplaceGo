"use client"

import { useState } from "react"
import { ArrowLeft, Briefcase, Calendar, Clock, Users, Video } from "lucide-react"
import { useTranslations } from "next-intl"
import { useRouter } from "@i18n/navigation"
import { cn } from "@/shared/lib/utils"
import { useHasApplied } from "../hooks/use-job-applications"
import { ApplyModal } from "./apply-modal"
import type { JobResponse } from "../types"
import { getJob } from "../api/job-api"
import { useQuery } from "@tanstack/react-query"

interface OpportunityDetailProps {
  jobId: string
}

export function OpportunityDetail({ jobId }: OpportunityDetailProps) {
  const t = useTranslations("opportunity")
  const router = useRouter()
  const [showApplyModal, setShowApplyModal] = useState(false)

  const { data: job, isLoading } = useQuery({
    queryKey: ["jobs", jobId],
    queryFn: () => getJob(jobId),
  })
  const { data: appliedData } = useHasApplied(jobId)
  const hasAlreadyApplied = appliedData?.has_applied ?? false

  if (isLoading) {
    return (
      <div className="space-y-4 animate-shimmer">
        <div className="h-8 w-2/3 rounded-lg bg-slate-100 dark:bg-slate-800" />
        <div className="h-4 w-1/3 rounded bg-slate-100 dark:bg-slate-800" />
        <div className="h-40 rounded-2xl bg-slate-100 dark:bg-slate-800" />
      </div>
    )
  }

  if (!job) return null

  return (
    <div className="space-y-6">
      {/* Back */}
      <button type="button" onClick={() => router.back()} className="flex items-center gap-1.5 text-sm text-slate-500 hover:text-slate-700 dark:hover:text-slate-300">
        <ArrowLeft className="h-4 w-4" />
        {t("back")}
      </button>

      {/* Header */}
      <div className="flex items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold text-slate-900 dark:text-white">{job.title}</h1>
          <div className="flex items-center gap-3 mt-2 text-sm text-slate-500 dark:text-slate-400">
            <span className="flex items-center gap-1"><Calendar className="h-4 w-4" />{new Date(job.created_at).toLocaleDateString("fr-FR", { day: "numeric", month: "long", year: "numeric" })}</span>
            <span className="flex items-center gap-1"><Users className="h-4 w-4" />{job.applicant_type === "all" ? t("allTypes") : job.applicant_type === "freelancers" ? t("freelancersOnly") : t("agenciesOnly")}</span>
          </div>
        </div>
        <button
          type="button"
          onClick={() => setShowApplyModal(true)}
          disabled={hasAlreadyApplied}
          className={cn(
            "shrink-0 rounded-xl px-6 py-2.5 text-sm font-medium transition-all",
            hasAlreadyApplied
              ? "bg-slate-100 text-slate-400 cursor-not-allowed dark:bg-slate-700"
              : "gradient-primary text-white hover:shadow-glow active:scale-[0.98]",
          )}
        >
          {hasAlreadyApplied ? t("alreadyApplied") : t("apply")}
        </button>
      </div>

      {/* Budget card */}
      <div className="rounded-2xl border border-slate-100 bg-white p-5 shadow-sm dark:border-slate-700 dark:bg-slate-800/80">
        <div className="flex items-center gap-3 mb-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-rose-100 dark:bg-rose-500/20">
            <Briefcase className="h-5 w-5 text-rose-600 dark:text-rose-400" />
          </div>
          <div>
            <p className="text-sm font-medium text-slate-900 dark:text-white">{job.budget_type === "one_shot" ? t("oneShot") : t("longTerm")}</p>
            <p className="text-lg font-bold text-rose-600 dark:text-rose-400">
              {job.min_budget.toLocaleString("fr-FR")}€ - {job.max_budget.toLocaleString("fr-FR")}€
              {job.payment_frequency && <span className="text-sm font-normal text-slate-500"> / {job.payment_frequency === "weekly" ? t("weekly") : t("monthly")}</span>}
            </p>
          </div>
        </div>
        {job.duration_weeks && !job.is_indefinite && (
          <p className="flex items-center gap-1 text-sm text-slate-500"><Clock className="h-4 w-4" />{job.duration_weeks} {t("weeks")}</p>
        )}
        {job.is_indefinite && (
          <p className="flex items-center gap-1 text-sm text-slate-500"><Clock className="h-4 w-4" />{t("indefinite")}</p>
        )}
      </div>

      {/* Description */}
      <div className="rounded-2xl border border-slate-100 bg-white p-5 shadow-sm dark:border-slate-700 dark:bg-slate-800/80">
        <h2 className="text-base font-semibold text-slate-900 dark:text-white mb-3">{t("description")}</h2>
        {job.video_url && (
          <div className="mb-4 flex items-center gap-2 text-sm text-rose-600"><Video className="h-4 w-4" /><a href={job.video_url} target="_blank" rel="noopener noreferrer" className="underline">{t("watchVideo")}</a></div>
        )}
        <p className="text-sm text-slate-600 dark:text-slate-300 whitespace-pre-wrap">{job.description}</p>
      </div>

      {/* Skills */}
      {job.skills.length > 0 && (
        <div className="rounded-2xl border border-slate-100 bg-white p-5 shadow-sm dark:border-slate-700 dark:bg-slate-800/80">
          <h2 className="text-base font-semibold text-slate-900 dark:text-white mb-3">{t("requiredSkills")}</h2>
          <div className="flex flex-wrap gap-2">
            {job.skills.map((skill) => (
              <span key={skill} className="rounded-full bg-rose-50 px-3 py-1 text-sm font-medium text-rose-700 dark:bg-rose-500/10 dark:text-rose-400">{skill}</span>
            ))}
          </div>
        </div>
      )}

      <ApplyModal open={showApplyModal} onClose={() => setShowApplyModal(false)} jobId={jobId} />
    </div>
  )
}
