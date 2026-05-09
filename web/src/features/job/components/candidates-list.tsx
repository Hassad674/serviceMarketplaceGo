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
import type { ApplicantKind, ApplicationWithProfile } from "../types"

interface CandidatesListProps {
  jobId: string
}

// CandidateKindFilter is the chip selection on the candidates tab. The
// "all" option is the implicit default; a selection is reflected in the
// `kind` query param so a refresh / share link reproduces the view.
type CandidateKindFilter = "all" | ApplicantKind

const KIND_FILTERS: ReadonlyArray<{ value: CandidateKindFilter; labelKey: string }> = [
  { value: "all", labelKey: "filterAllCandidates" },
  { value: "freelance", labelKey: "filterFreelances" },
  { value: "agency", labelKey: "filterAgencies" },
  { value: "referrer", labelKey: "filterReferrers" },
] as const

function parseKindParam(value: string | null): CandidateKindFilter {
  if (value === "freelance" || value === "agency" || value === "referrer") {
    return value
  }
  return "all"
}

// W-08 candidates list — Soleil v2 wrapper around CandidateCard.
// Loading state uses ivoire skeleton tiles, empty state renders an
// editorial corail-soft Soleil card with a CTA back to the listings.
// Selecting a card opens CandidateDetailPanel (already Soleil-styled
// in the panel file). Underlying API hook is unchanged.
//
// 2026-05-09 — Persona filter (Fix 3): segmented filter chips above the
// list narrow the rows to a single applicant_kind. The active filter
// is mirrored in the URL query (?kind=freelance|agency|referrer) so a
// refresh or share link preserves the view.
export function CandidatesList({ jobId }: CandidatesListProps) {
  const t = useTranslations("opportunity")
  const tJob = useTranslations("job")
  const searchParams = useSearchParams()
  const router = useRouter()
  const pathname = usePathname()

  const filter = parseKindParam(searchParams.get("kind"))
  const repoKind = filter === "all" ? undefined : filter

  const { data, isLoading } = useJobApplications(jobId, undefined, repoKind)

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

  const handleFilterChange = useCallback(
    (next: CandidateKindFilter) => {
      const params = new URLSearchParams(searchParams.toString())
      if (next === "all") {
        params.delete("kind")
      } else {
        params.set("kind", next)
      }
      // Reset the selected candidate when the filter changes —
      // the previously selected row may no longer be in the list.
      params.delete("candidate")
      const qs = params.toString()
      router.replace(qs ? `${pathname}?${qs}` : pathname, { scroll: false })
    },
    [searchParams, router, pathname],
  )

  if (isLoading) {
    return (
      <div className="space-y-3">
        <CandidateKindFilterBar active={filter} onChange={handleFilterChange} />
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
    return (
      <div className="space-y-4">
        <CandidateKindFilterBar active={filter} onChange={handleFilterChange} />
        <CandidatesEmpty />
      </div>
    )
  }

  return (
    <>
      <div className="space-y-4">
        <CandidateKindFilterBar active={filter} onChange={handleFilterChange} />
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

interface CandidateKindFilterBarProps {
  active: CandidateKindFilter
  onChange: (next: CandidateKindFilter) => void
}

function CandidateKindFilterBar({ active, onChange }: CandidateKindFilterBarProps) {
  const t = useTranslations("opportunity")
  return (
    <div
      role="tablist"
      aria-label={t("filterAllCandidates")}
      className="-mx-1 flex flex-wrap items-center gap-1 overflow-x-auto"
    >
      {KIND_FILTERS.map((opt) => {
        const isActive = active === opt.value
        return (
          <button
            key={opt.value}
            type="button"
            role="tab"
            aria-selected={isActive}
            onClick={() => onChange(opt.value)}
            className={cn(
              "rounded-full px-3.5 py-1.5 text-[12.5px] font-semibold transition-colors duration-150",
              isActive
                ? "bg-primary-soft text-primary-deep"
                : "border border-border bg-card text-muted-foreground hover:border-border-strong hover:text-foreground",
            )}
          >
            {t(opt.labelKey)}
          </button>
        )
      })}
    </div>
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
