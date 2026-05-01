"use client"

import { useState, useEffect } from "react"
import { useParams } from "next/navigation"
import { useQuery } from "@tanstack/react-query"
import { ArrowLeft, Briefcase, Calendar, Users, Clock, Pencil } from "lucide-react"
import { useTranslations } from "next-intl"
import { useRouter, Link } from "@i18n/navigation"
import { cn } from "@/shared/lib/utils"
import { getJob } from "@/features/job/api/job-api"
import { CandidatesList } from "@/features/job/components/candidates-list"
import { useMarkApplicationsViewed } from "@/features/job/hooks/use-jobs"

import { Button } from "@/shared/components/ui/button"
type Tab = "details" | "candidates"

export default function JobDetailPage() {
  const params = useParams<{ id: string }>()
  const t = useTranslations("job")
  const tOpp = useTranslations("opportunity")
  const router = useRouter()
  const jobId = params.id
  const [activeTab, setActiveTab] = useState<Tab>("details")
  const markViewed = useMarkApplicationsViewed()

  const { data: job, isLoading } = useQuery({
    queryKey: ["jobs", jobId],
    queryFn: () => getJob(jobId),
  })

  // Mark as viewed when candidates tab is displayed
  useEffect(() => {
    if (activeTab === "candidates" && jobId) {
      markViewed.mutate(jobId)
    }
  }, [activeTab, jobId]) // eslint-disable-line react-hooks/exhaustive-deps

  if (isLoading) {
    return (
      <div className="space-y-4 animate-shimmer">
        <div className="h-8 w-2/3 rounded-lg bg-slate-100 dark:bg-slate-800" />
        <div className="h-40 rounded-2xl bg-slate-100 dark:bg-slate-800" />
      </div>
    )
  }

  if (!job) return null

  return (
    <div className="space-y-6">
      <Button variant="ghost" size="auto" type="button" onClick={() => router.push("/jobs")} className="flex items-center gap-1.5 text-sm text-slate-500 hover:text-slate-700 dark:hover:text-slate-300">
        <ArrowLeft className="h-4 w-4" />
        {t("title")}
      </Button>

      {/* Header */}
      <div className="flex items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold text-slate-900 dark:text-white">{job.title}</h1>
          <div className="flex items-center gap-3 mt-2 text-sm text-slate-500">
            <span className="flex items-center gap-1"><Briefcase className="h-4 w-4" />{job.min_budget.toLocaleString("fr-FR")}€ - {job.max_budget.toLocaleString("fr-FR")}€</span>
            <span className="flex items-center gap-1"><Users className="h-4 w-4" />{job.applicant_type}</span>
            <span className="flex items-center gap-1"><Calendar className="h-4 w-4" />{new Date(job.created_at).toLocaleDateString("fr-FR")}</span>
          </div>
        </div>
        <div className="flex items-center gap-2 shrink-0">
          <Link
            href={`/jobs/${jobId}/edit`}
            className={cn(
              "flex items-center gap-1.5 rounded-xl px-4 py-2 text-sm font-medium transition-all",
              "border border-slate-200 text-slate-700 hover:bg-slate-50 dark:border-slate-600 dark:text-slate-300 dark:hover:bg-slate-700",
            )}
          >
            <Pencil className="h-4 w-4" />
            {t("editJob")}
          </Link>
          <span className={cn(
            "rounded-full px-3 py-1 text-xs font-semibold",
            job.status === "open"
              ? "bg-green-100 text-green-700 dark:bg-green-500/20 dark:text-green-300"
              : "bg-gray-100 text-gray-500",
          )}>
            {job.status === "open" ? t("statusOpen") : t("statusClosed")}
          </span>
        </div>
      </div>

      {/* Tabs */}
      <div className="flex gap-1 border-b border-slate-200 dark:border-slate-700">
        {(["details", "candidates"] as const).map((tab) => (
          <Button variant="ghost" size="auto"
            key={tab}
            type="button"
            onClick={() => setActiveTab(tab)}
            className={cn(
              "px-4 py-2.5 text-sm font-medium transition-colors border-b-2 -mb-px",
              activeTab === tab
                ? "border-rose-500 text-rose-600 dark:text-rose-400"
                : "border-transparent text-slate-500 hover:text-slate-700 dark:hover:text-slate-300",
            )}
          >
            {tab === "details" ? t("jobDetails") : tOpp("candidates")}
          </Button>
        ))}
      </div>

      {/* Tab content */}
      {activeTab === "details" && (
        <div className="space-y-5">
          {/* Video */}
          {job.video_url && (
            <div className="rounded-2xl border border-slate-100 bg-white p-5 shadow-sm dark:border-slate-700 dark:bg-slate-800/80">
              <h2 className="text-base font-semibold text-slate-900 dark:text-white mb-3">{tOpp("watchVideo")}</h2>
              <div className="aspect-video max-h-[360px] overflow-hidden rounded-xl bg-black">
                <video
                  src={job.video_url}
                  controls
                  className="h-full w-full object-contain"
                  aria-label={job.title}
                >
                  <track kind="captions" />
                </video>
              </div>
            </div>
          )}

          {/* Description */}
          <div className="rounded-2xl border border-slate-100 bg-white p-5 shadow-sm dark:border-slate-700 dark:bg-slate-800/80">
            <h2 className="text-base font-semibold text-slate-900 dark:text-white mb-3">{tOpp("description")}</h2>
            <p className="text-sm text-slate-600 dark:text-slate-300 whitespace-pre-wrap break-words overflow-wrap-anywhere">{job.description}</p>
          </div>

          {/* Skills */}
          {job.skills.length > 0 && (
            <div className="rounded-2xl border border-slate-100 bg-white p-5 shadow-sm dark:border-slate-700 dark:bg-slate-800/80">
              <h2 className="text-base font-semibold text-slate-900 dark:text-white mb-3">{tOpp("requiredSkills")}</h2>
              <div className="flex flex-wrap gap-2">
                {job.skills.map((skill) => (
                  <span key={skill} className="rounded-full bg-rose-50 px-3 py-1 text-sm font-medium text-rose-700 dark:bg-rose-500/10 dark:text-rose-400">{skill}</span>
                ))}
              </div>
            </div>
          )}

          {/* Budget */}
          <div className="rounded-2xl border border-slate-100 bg-white p-5 shadow-sm dark:border-slate-700 dark:bg-slate-800/80">
            <div className="flex items-center gap-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-rose-100 dark:bg-rose-500/20">
                <Briefcase className="h-5 w-5 text-rose-600 dark:text-rose-400" />
              </div>
              <div>
                <p className="text-sm font-medium text-slate-900 dark:text-white">{job.budget_type === "one_shot" ? t("oneShot") : t("longTerm")}</p>
                <p className="text-lg font-bold text-rose-600 dark:text-rose-400">
                  {job.min_budget.toLocaleString("fr-FR")}€ - {job.max_budget.toLocaleString("fr-FR")}€
                </p>
              </div>
            </div>
            {job.duration_weeks && !job.is_indefinite && (
              <p className="flex items-center gap-1 mt-2 text-sm text-slate-500"><Clock className="h-4 w-4" />{job.duration_weeks} semaines</p>
            )}
          </div>
        </div>
      )}

      {activeTab === "candidates" && (
        <CandidatesList jobId={jobId} />
      )}
    </div>
  )
}
