"use client"

import { useState } from "react"
import { Search, Briefcase, Loader2, Ticket, HelpCircle, AlertTriangle } from "lucide-react"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import { useOpenJobs, useMyApplications } from "../hooks/use-job-applications"
import { useCredits } from "../hooks/use-jobs"
import { useUser } from "@/shared/hooks/use-user"
import { OpportunityCard } from "./opportunity-card"
import { CreditsInfoModal } from "./credits-info-modal"
import type { OpenJobListFilters, JobResponse } from "../types"
import { Button } from "@/shared/components/ui/button"

import { Input } from "@/shared/components/ui/input"

// W-12 · Opportunities feed list — Soleil v2.
// Search pill + corail credits chip + filter pills + responsive 1/2/3
// column grid + Soleil-style empty state. The filtering / pagination
// behaviour mirrors the original implementation exactly so all existing
// hooks/api calls remain untouched.
export function OpportunityList() {
  const t = useTranslations("opportunity")
  const [filters, setFilters] = useState<OpenJobListFilters>({})
  const [search, setSearch] = useState("")
  const [cursor, setCursor] = useState<string | undefined>()
  const [showCreditsInfo, setShowCreditsInfo] = useState(false)

  const { data: user } = useUser()
  const { data: creditsData } = useCredits()
  const credits = creditsData?.credits
  const { data: myAppsData } = useMyApplications()
  const appliedJobIds = new Set(myAppsData?.data.map((a) => a.application.job_id) ?? [])
  const activeFilters = { ...filters, search: search || undefined }
  const { data, isLoading, error } = useOpenJobs(activeFilters, cursor)

  // Filter out own jobs + jobs incompatible with user's role
  function filterJobs(jobs: JobResponse[]): JobResponse[] {
    return jobs.filter((job) => {
      if (user?.id && job.creator_id === user.id) return false
      if (user?.role === "provider") return job.applicant_type === "freelancers" || job.applicant_type === "all"
      if (user?.role === "agency") return job.applicant_type === "agencies" || job.applicant_type === "all"
      return true
    })
  }

  return (
    <div className="space-y-6">
      {/* Search bar + credits */}
      <div className="flex flex-wrap items-center gap-3">
        <div className="relative min-w-[240px] flex-1">
          <Search className="pointer-events-none absolute left-4 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            type="text"
            placeholder={t("searchPlaceholder")}
            value={search}
            onChange={(e) => {
              setSearch(e.target.value)
              setCursor(undefined)
            }}
            className={cn(
              "h-11 w-full rounded-full border border-border bg-card pl-11 pr-4 text-sm text-foreground",
              "placeholder:text-muted-foreground",
              "focus:border-border-strong focus:ring-2 focus:ring-primary/20 outline-none transition-all",
            )}
          />
        </div>
        {credits !== undefined && (
          <div className="flex shrink-0 items-center gap-2 rounded-full border border-primary/20 bg-primary-soft px-3.5 py-1.5">
            <Ticket className="h-4 w-4 text-primary-deep" strokeWidth={1.6} />
            <span className="text-[13px] font-semibold text-primary-deep">
              {t("creditsRemaining", { count: credits })}
            </span>
            <Button
              variant="ghost"
              size="auto"
              type="button"
              onClick={() => setShowCreditsInfo(true)}
              className="ml-0.5 rounded-full p-0.5 text-primary-deep/70 hover:bg-card hover:text-primary-deep"
              aria-label={t("creditsHowItWorks")}
            >
              <HelpCircle className="h-3.5 w-3.5" />
            </Button>
          </div>
        )}
      </div>

      {/* No credits warning — Soleil amber-soft */}
      {credits === 0 && (
        <div className="flex items-center gap-2 rounded-2xl border border-border-strong bg-amber-soft px-4 py-3">
          <AlertTriangle className="h-4 w-4 shrink-0 text-warning" strokeWidth={1.6} />
          <p className="text-sm text-foreground">{t("noCreditsLeft")}</p>
        </div>
      )}

      {/* Filter pills — Soleil signature: full radius, encre when active */}
      <div className="flex flex-wrap gap-2">
        {(["all", "freelancers", "agencies"] as const).map((type) => (
          <Button
            key={type}
            variant="ghost"
            size="auto"
            type="button"
            onClick={() => {
              setFilters((f) => ({ ...f, applicant_type: f.applicant_type === type ? undefined : type }))
              setCursor(undefined)
            }}
            className={cn(
              "rounded-full px-3.5 py-1.5 text-[12.5px] font-semibold transition-all",
              filters.applicant_type === type
                ? "bg-foreground text-background"
                : "border border-border bg-card text-foreground hover:border-border-strong",
            )}
          >
            {type === "all"
              ? t("allTypes")
              : type === "freelancers"
                ? t("freelancersOnly")
                : t("agenciesOnly")}
          </Button>
        ))}
        {(["one_shot", "long_term"] as const).map((type) => (
          <Button
            key={type}
            variant="ghost"
            size="auto"
            type="button"
            onClick={() => {
              setFilters((f) => ({ ...f, budget_type: f.budget_type === type ? undefined : type }))
              setCursor(undefined)
            }}
            className={cn(
              "rounded-full px-3.5 py-1.5 text-[12.5px] font-semibold transition-all",
              filters.budget_type === type
                ? "bg-foreground text-background"
                : "border border-border bg-card text-foreground hover:border-border-strong",
            )}
          >
            {type === "one_shot" ? t("oneShot") : t("longTerm")}
          </Button>
        ))}
      </div>

      {/* Loading skeleton */}
      {isLoading && (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {Array.from({ length: 6 }).map((_, i) => (
            <div
              key={i}
              className="h-48 rounded-2xl border border-border bg-card animate-shimmer"
              style={{ boxShadow: "var(--shadow-card)" }}
            />
          ))}
        </div>
      )}

      {/* Error */}
      {error && (
        <div className="rounded-2xl border border-border bg-card px-4 py-3 text-center text-sm text-primary-deep">
          {error.message}
        </div>
      )}

      {/* Empty state — Soleil-style */}
      {data && filterJobs(data.data).length === 0 && (
        <div
          className="flex flex-col items-center gap-3 rounded-2xl border border-border bg-card px-6 py-16 text-center"
          style={{ boxShadow: "var(--shadow-card)" }}
        >
          <div className="flex h-12 w-12 items-center justify-center rounded-full bg-primary-soft text-primary">
            <Briefcase className="h-5 w-5" strokeWidth={1.6} />
          </div>
          <p className="font-serif text-[18px] font-medium text-foreground">
            {t("noOpportunities")}
          </p>
        </div>
      )}

      {/* Results grid — 1/2/3 col responsive */}
      {data && filterJobs(data.data).length > 0 && (
        <>
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {filterJobs(data.data).map((job) => (
              <OpportunityCard
                key={job.id}
                job={job}
                hasApplied={appliedJobIds.has(job.id)}
              />
            ))}
          </div>

          {data.has_more && (
            <div className="flex justify-center pt-2">
              <Button
                variant="ghost"
                size="auto"
                type="button"
                onClick={() => setCursor(data.next_cursor)}
                className="inline-flex items-center gap-2 rounded-full border border-border-strong bg-card px-5 py-2 text-sm font-semibold text-foreground transition-colors hover:border-primary"
              >
                <Loader2 className="h-4 w-4" />
                {t("loadMore")}
              </Button>
            </div>
          )}
        </>
      )}

      <CreditsInfoModal open={showCreditsInfo} onClose={() => setShowCreditsInfo(false)} />
    </div>
  )
}
