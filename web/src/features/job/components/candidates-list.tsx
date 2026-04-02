"use client"

import { useCallback, useMemo } from "react"
import { Users } from "lucide-react"
import { useTranslations } from "next-intl"
import { useSearchParams } from "next/navigation"
import { useRouter, usePathname } from "@i18n/navigation"
import { useJobApplications } from "../hooks/use-job-applications"
import { CandidateCard } from "./candidate-card"
import { CandidateDetailPanel } from "./candidate-detail-panel"
import type { ApplicationWithProfile } from "../types"

interface CandidatesListProps {
  jobId: string
}

export function CandidatesList({ jobId }: CandidatesListProps) {
  const t = useTranslations("opportunity")
  const { data, isLoading } = useJobApplications(jobId)
  const searchParams = useSearchParams()
  const router = useRouter()
  const pathname = usePathname()

  const selectedId = searchParams.get("candidate")
  const candidates = useMemo(() => data?.data ?? [], [data?.data])

  const selectedCandidate = useMemo<ApplicationWithProfile | null>(() => {
    if (!selectedId || candidates.length === 0) return null
    return candidates.find((c) => c.application.id === selectedId) ?? null
  }, [selectedId, candidates])

  const handleSelect = useCallback(
    (applicationId: string) => {
      const params = new URLSearchParams(searchParams.toString())
      params.set("candidate", applicationId)
      router.replace(`${pathname}?${params.toString()}`, { scroll: false })
    },
    [searchParams, router, pathname],
  )

  const handleClose = useCallback(() => {
    const params = new URLSearchParams(searchParams.toString())
    params.delete("candidate")
    const qs = params.toString()
    router.replace(qs ? `${pathname}?${qs}` : pathname, { scroll: false })
  }, [searchParams, router, pathname])

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
    <>
      <div className="space-y-3">
        <p className="text-sm text-slate-500 dark:text-slate-400">
          {t("applicantCount", { count: data.data.length })}
        </p>
        {data.data.map((item) => (
          <CandidateCard
            key={item.application.id}
            item={item}
            isSelected={item.application.id === selectedId}
            onClick={() => handleSelect(item.application.id)}
          />
        ))}
      </div>

      <CandidateDetailPanel
        candidate={selectedCandidate}
        candidates={candidates}
        onClose={handleClose}
        onNavigate={handleSelect}
        jobId={jobId}
      />
    </>
  )
}
