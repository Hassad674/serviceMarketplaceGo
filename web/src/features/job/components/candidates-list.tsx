"use client"

import { Users, Loader2 } from "lucide-react"
import { useTranslations } from "next-intl"
import { useJobApplications } from "../hooks/use-job-applications"
import { CandidateCard } from "./candidate-card"

interface CandidatesListProps {
  jobId: string
}

export function CandidatesList({ jobId }: CandidatesListProps) {
  const t = useTranslations("opportunity")
  const { data, isLoading } = useJobApplications(jobId)

  if (isLoading) {
    return (
      <div className="space-y-4">
        {Array.from({ length: 3 }).map((_, i) => (
          <div key={i} className="h-24 rounded-2xl bg-slate-100 animate-shimmer dark:bg-slate-800" />
        ))}
      </div>
    )
  }

  if (!data || data.data.length === 0) {
    return (
      <div className="text-center py-8">
        <Users className="mx-auto h-10 w-10 text-slate-300 mb-3" />
        <p className="text-sm text-slate-500 dark:text-slate-400">{t("noCandidates")}</p>
      </div>
    )
  }

  return (
    <div className="space-y-4">
      <p className="text-sm text-slate-500 dark:text-slate-400">
        {t("applicantCount", { count: data.data.length })}
      </p>
      {data.data.map((item) => (
        <CandidateCard key={item.application.id} item={item} jobId={jobId} />
      ))}
    </div>
  )
}
