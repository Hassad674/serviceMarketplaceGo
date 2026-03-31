"use client"

import { useParams } from "next/navigation"
import { useQuery } from "@tanstack/react-query"
import { ArrowLeft, Briefcase, Calendar, Users } from "lucide-react"
import { useTranslations } from "next-intl"
import { useRouter } from "@i18n/navigation"
import { getJob } from "@/features/job/api/job-api"
import { CandidatesList } from "@/features/job/components/candidates-list"

export default function JobDetailPage() {
  const params = useParams<{ id: string }>()
  const t = useTranslations("opportunity")
  const router = useRouter()
  const jobId = params.id

  const { data: job, isLoading } = useQuery({
    queryKey: ["jobs", jobId],
    queryFn: () => getJob(jobId),
  })

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
      <button type="button" onClick={() => router.back()} className="flex items-center gap-1.5 text-sm text-slate-500 hover:text-slate-700">
        <ArrowLeft className="h-4 w-4" />
        {t("back")}
      </button>

      <div>
        <h1 className="text-2xl font-bold text-slate-900 dark:text-white">{job.title}</h1>
        <div className="flex items-center gap-3 mt-2 text-sm text-slate-500">
          <span className="flex items-center gap-1"><Briefcase className="h-4 w-4" />{job.min_budget.toLocaleString("fr-FR")}€ - {job.max_budget.toLocaleString("fr-FR")}€</span>
          <span className="flex items-center gap-1"><Users className="h-4 w-4" />{job.applicant_type}</span>
          <span className="flex items-center gap-1"><Calendar className="h-4 w-4" />{new Date(job.created_at).toLocaleDateString("fr-FR")}</span>
        </div>
      </div>

      <div className="rounded-2xl border border-slate-100 bg-white p-5 shadow-sm dark:border-slate-700 dark:bg-slate-800/80">
        <h2 className="text-base font-semibold text-slate-900 dark:text-white mb-3">{t("candidates")}</h2>
        <CandidatesList jobId={jobId} />
      </div>
    </div>
  )
}
