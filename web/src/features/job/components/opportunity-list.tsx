"use client"

import { useState } from "react"
import { Search, Briefcase, Loader2 } from "lucide-react"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import { useOpenJobs } from "../hooks/use-job-applications"
import { OpportunityCard } from "./opportunity-card"
import type { OpenJobListFilters } from "../types"

export function OpportunityList() {
  const t = useTranslations("opportunity")
  const [filters, setFilters] = useState<OpenJobListFilters>({})
  const [search, setSearch] = useState("")
  const [cursor, setCursor] = useState<string | undefined>()

  const activeFilters = { ...filters, search: search || undefined }
  const { data, isLoading, error } = useOpenJobs(activeFilters, cursor)

  return (
    <div className="space-y-6">
      {/* Search bar */}
      <div className="relative">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-slate-400" />
        <input
          type="text"
          placeholder={t("searchPlaceholder")}
          value={search}
          onChange={(e) => { setSearch(e.target.value); setCursor(undefined) }}
          className={cn(
            "w-full h-10 pl-10 pr-4 rounded-lg border border-slate-200 bg-white text-sm",
            "focus:border-rose-500 focus:ring-4 focus:ring-rose-500/10 outline-none transition-all",
            "dark:border-slate-600 dark:bg-slate-800 dark:text-white",
          )}
        />
      </div>

      {/* Filter chips */}
      <div className="flex flex-wrap gap-2">
        {(["all", "freelancers", "agencies"] as const).map((type) => (
          <button
            key={type}
            type="button"
            onClick={() => { setFilters((f) => ({ ...f, applicant_type: f.applicant_type === type ? undefined : type })); setCursor(undefined) }}
            className={cn(
              "rounded-full px-3 py-1 text-xs font-medium transition-all",
              filters.applicant_type === type
                ? "bg-rose-500 text-white"
                : "bg-slate-100 text-slate-600 hover:bg-slate-200 dark:bg-slate-700 dark:text-slate-300",
            )}
          >
            {type === "all" ? t("allTypes") : type === "freelancers" ? t("freelancersOnly") : t("agenciesOnly")}
          </button>
        ))}
        {(["one_shot", "long_term"] as const).map((type) => (
          <button
            key={type}
            type="button"
            onClick={() => { setFilters((f) => ({ ...f, budget_type: f.budget_type === type ? undefined : type })); setCursor(undefined) }}
            className={cn(
              "rounded-full px-3 py-1 text-xs font-medium transition-all",
              filters.budget_type === type
                ? "bg-rose-500 text-white"
                : "bg-slate-100 text-slate-600 hover:bg-slate-200 dark:bg-slate-700 dark:text-slate-300",
            )}
          >
            {type === "one_shot" ? t("oneShot") : t("longTerm")}
          </button>
        ))}
      </div>

      {/* Loading */}
      {isLoading && (
        <div className="grid gap-4 sm:grid-cols-2">
          {Array.from({ length: 4 }).map((_, i) => (
            <div key={i} className="h-40 rounded-2xl bg-slate-100 animate-shimmer dark:bg-slate-800" />
          ))}
        </div>
      )}

      {/* Error */}
      {error && (
        <div className="text-center py-8 text-sm text-red-500">{error.message}</div>
      )}

      {/* Empty */}
      {data && data.data.length === 0 && (
        <div className="text-center py-12">
          <Briefcase className="mx-auto h-10 w-10 text-slate-300 mb-3" />
          <p className="text-sm text-slate-500 dark:text-slate-400">{t("noOpportunities")}</p>
        </div>
      )}

      {/* Results */}
      {data && data.data.length > 0 && (
        <>
          <div className="grid gap-4 sm:grid-cols-2">
            {data.data.map((job) => (
              <OpportunityCard key={job.id} job={job} />
            ))}
          </div>

          {data.has_more && (
            <div className="flex justify-center">
              <button
                type="button"
                onClick={() => setCursor(data.next_cursor)}
                className="flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-medium text-rose-600 hover:bg-rose-50 dark:hover:bg-rose-500/10 transition-colors"
              >
                <Loader2 className="h-4 w-4" />
                {t("loadMore")}
              </button>
            </div>
          )}
        </>
      )}
    </div>
  )
}
