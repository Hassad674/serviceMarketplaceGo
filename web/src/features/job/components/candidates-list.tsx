"use client"

import { useCallback, useMemo } from "react"
import { Users } from "lucide-react"
import { useTranslations } from "next-intl"
import { useSearchParams } from "next/navigation"
import { Link, useRouter, usePathname } from "@i18n/navigation"
import { cn } from "@/shared/lib/utils"
import { useJobApplications } from "../hooks/use-job-applications"
import { CandidateCard } from "./candidate-card"
import { CandidateDetailPanel } from "./candidate-detail-panel"
import type { ApplicationWithProfile } from "../types"

interface CandidatesListProps {
  jobId: string
}

// W-08 candidates list — Soleil v2 wrapper around CandidateCard.
// Loading state uses ivoire skeleton tiles, empty state renders an
// editorial corail-soft Soleil card with a CTA back to the listings.
// Selecting a card opens CandidateDetailPanel (already Soleil-styled
// in the panel file). Underlying API hook is unchanged.
export function CandidatesList({ jobId }: CandidatesListProps) {
  const t = useTranslations("opportunity")
  const tJob = useTranslations("job")
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
      <div className="space-y-3">
        {Array.from({ length: 3 }).map((_, i) => (
          <div
            key={i}
            className="h-28 animate-shimmer rounded-2xl border border-border bg-card"
          />
        ))}
      </div>
    )
  }

  if (!data || data.data.length === 0) {
    return <CandidatesEmpty />
  }

  return (
    <>
      <div className="space-y-4">
        <div className="flex items-baseline justify-between gap-3">
          <p className="font-mono text-[11px] font-bold uppercase tracking-[0.1em] text-primary-deep">
            {tJob("jobDetail_w08_eyebrow")}
          </p>
          <p className="text-[12.5px] font-medium text-muted-foreground">
            {tJob("jobDetail_w08_count", { count: data.data.length })}
          </p>
        </div>
        <div className="space-y-3">
          {data.data.map((item) => (
            <CandidateCard
              key={item.application.id}
              item={item}
              isSelected={item.application.id === selectedId}
              onClick={() => handleSelect(item.application.id)}
            />
          ))}
        </div>
        {/* Keep `t` imported above used at runtime for parity with the
            earlier translation namespace (e.g. screen-reader fallbacks
            in CandidateDetailPanel below). */}
        <span className="sr-only">{t("candidates")}</span>
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

function CandidatesEmpty() {
  const tJob = useTranslations("job")
  return (
    <section
      className={cn(
        "relative overflow-hidden rounded-2xl border border-border p-8 text-center sm:p-10",
      )}
      style={{
        background: "var(--gradient-warm)",
        boxShadow: "var(--shadow-card)",
      }}
    >
      <div className="mx-auto flex h-12 w-12 items-center justify-center rounded-full bg-primary-soft text-primary-deep">
        <Users className="h-5 w-5" strokeWidth={1.7} />
      </div>
      <p className="mt-4 font-mono text-[11px] font-bold uppercase tracking-[0.1em] text-primary-deep">
        {tJob("jobDetail_w08_eyebrow")}
      </p>
      <h3 className="mt-2 font-serif text-[22px] font-medium tracking-[-0.01em] text-foreground sm:text-[26px]">
        {tJob("jobDetail_w08_emptyTitle")}
      </h3>
      <p className="mx-auto mt-2 max-w-md text-[14px] leading-relaxed text-muted-foreground">
        {tJob("jobDetail_w08_emptyBody")}
      </p>
      <Link
        href="/jobs"
        className={cn(
          "mt-5 inline-flex items-center gap-1.5 rounded-full px-4 py-2",
          "bg-primary text-[13px] font-semibold text-white",
          "transition-colors duration-150 hover:bg-primary-deep",
        )}
        style={{ boxShadow: "var(--shadow-message)" }}
      >
        {tJob("jobDetail_w08_emptyCta")}
      </Link>
    </section>
  )
}
