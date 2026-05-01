"use client"

import { useState } from "react"
import { Briefcase, Calendar, Users, MoreVertical } from "lucide-react"
import { useTranslations } from "next-intl"
import { Link } from "@i18n/navigation"
import { cn } from "@/shared/lib/utils"
import { ReportDialog } from "@/features/reporting/components/report-dialog"
import type { JobResponse } from "../types"

interface OpportunityCardProps {
  job: JobResponse
  hasApplied?: boolean
}

export function OpportunityCard({ job, hasApplied = false }: OpportunityCardProps) {
  const t = useTranslations("opportunity")
  const tReport = useTranslations("reporting")
  const [showReport, setShowReport] = useState(false)

  const budgetLabel =
    job.budget_type === "long_term"
      ? `${job.min_budget.toLocaleString("fr-FR")}€ - ${job.max_budget.toLocaleString("fr-FR")}€ / ${job.payment_frequency === "weekly" ? t("weekly") : t("monthly")}`
      : `${job.min_budget.toLocaleString("fr-FR")}€ - ${job.max_budget.toLocaleString("fr-FR")}€`

  return (
    <>
    <Link href={`/opportunities/${job.id}`}>
      <div
        className={cn(
          "rounded-2xl border border-slate-100 bg-white p-5 shadow-sm transition-all dark:border-slate-700 dark:bg-slate-800/80",
          "hover:shadow-md hover:border-rose-200 hover:-translate-y-0.5 dark:hover:border-rose-500/30",
        )}
      >
        <div className="flex items-start justify-between gap-3 mb-3">
          <h3 className="text-base font-semibold text-slate-900 dark:text-white line-clamp-2">
            {job.title}
          </h3>
          <div className="flex items-center gap-1 shrink-0">
            {hasApplied && (
              <span className="rounded-full bg-slate-100 px-2.5 py-0.5 text-xs font-medium text-slate-500 dark:bg-slate-700 dark:text-slate-400">
                {t("alreadyApplied")}
              </span>
            )}
            <button
              type="button"
              onClick={(e) => {
                e.preventDefault()
                e.stopPropagation()
                setShowReport(true)
              }}
              aria-label={tReport("report")}
              className="flex h-7 w-7 items-center justify-center rounded-lg text-slate-300 hover:bg-slate-100 hover:text-slate-500 dark:text-slate-600 dark:hover:bg-slate-700 dark:hover:text-slate-400 transition-all"
            >
              <MoreVertical className="h-4 w-4" />
            </button>
          </div>
        </div>

        {job.description && (
          <p className="text-sm text-slate-500 dark:text-slate-400 line-clamp-2 mb-3 break-words overflow-wrap-anywhere">
            {job.description}
          </p>
        )}

        {job.skills.length > 0 && (
          <div className="flex flex-wrap gap-1.5 mb-3">
            {job.skills.slice(0, 4).map((skill) => (
              <span
                key={skill}
                className="rounded-full bg-rose-50 px-2.5 py-0.5 text-xs font-medium text-rose-700 dark:bg-rose-500/10 dark:text-rose-400"
              >
                {skill}
              </span>
            ))}
            {job.skills.length > 4 && (
              <span className="text-xs text-slate-400">+{job.skills.length - 4}</span>
            )}
          </div>
        )}

        <div className="flex items-center gap-4 text-xs text-slate-500 dark:text-slate-400">
          <span className="flex items-center gap-1">
            <Briefcase className="h-3.5 w-3.5" />
            {budgetLabel}
          </span>
          <span className="flex items-center gap-1">
            <Users className="h-3.5 w-3.5" />
            {job.applicant_type === "all" ? t("allTypes") : job.applicant_type === "freelancers" ? t("freelancersOnly") : t("agenciesOnly")}
          </span>
          <span className="flex items-center gap-1 ml-auto">
            <Calendar className="h-3.5 w-3.5" />
            {new Date(job.created_at).toLocaleDateString("fr-FR")}
          </span>
        </div>
      </div>
    </Link>
    <ReportDialog
      open={showReport}
      onClose={() => setShowReport(false)}
      targetType="job"
      targetId={job.id}
    />
    </>
  )
}
