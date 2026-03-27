"use client"

import { Briefcase, Plus, X, Clock } from "lucide-react"
import { useTranslations } from "next-intl"
import { Link } from "@i18n/navigation"
import { cn } from "@/shared/lib/utils"
import { useMyJobs, useCloseJob } from "../hooks/use-jobs"
import type { JobResponse } from "../types"

export function JobList() {
  const t = useTranslations("job")
  const { data, isLoading, error } = useMyJobs()
  const closeJob = useCloseJob()

  if (isLoading) {
    return <JobListSkeleton />
  }

  if (error) {
    return (
      <div className="rounded-xl border border-red-200 dark:border-red-800 bg-red-50 dark:bg-red-900/20 p-6 text-sm text-red-700 dark:text-red-300">
        {t("errorLoading")}
      </div>
    )
  }

  const jobs = data?.data ?? []

  if (jobs.length === 0) {
    return <EmptyState />
  }

  return (
    <div className="space-y-3">
      {jobs.map((job) => (
        <JobCard
          key={job.id}
          job={job}
          onClose={(id) => closeJob.mutate(id)}
          isClosing={closeJob.isPending}
        />
      ))}
    </div>
  )
}

/* -------------------------------------------------- */
/* Job card                                           */
/* -------------------------------------------------- */

type JobCardProps = {
  job: JobResponse
  onClose: (id: string) => void
  isClosing: boolean
}

function JobCard({ job, onClose, isClosing }: JobCardProps) {
  const t = useTranslations("job")
  const isOpen = job.status === "open"

  return (
    <div
      className={cn(
        "rounded-2xl border bg-white dark:bg-gray-900 p-5",
        "border-gray-200 dark:border-gray-700",
        "transition-all duration-200 hover:shadow-md hover:border-rose-200 dark:hover:border-rose-800",
      )}
    >
      <div className="flex items-start justify-between gap-4">
        <div className="min-w-0 flex-1">
          <h3 className="truncate text-base font-semibold text-gray-900 dark:text-white">
            {job.title}
          </h3>
          <p className="mt-1 line-clamp-2 text-sm text-gray-500 dark:text-gray-400">
            {job.description}
          </p>
        </div>

        {/* Status badge */}
        <span
          className={cn(
            "shrink-0 rounded-full px-3 py-1 text-xs font-semibold",
            isOpen
              ? "bg-green-100 dark:bg-green-500/20 text-green-700 dark:text-green-300"
              : "bg-gray-100 dark:bg-gray-800 text-gray-500 dark:text-gray-400",
          )}
        >
          {isOpen ? t("statusOpen") : t("statusClosed")}
        </span>
      </div>

      {/* Skills */}
      {job.skills.length > 0 && (
        <div className="mt-3 flex flex-wrap gap-1.5">
          {job.skills.map((skill) => (
            <span
              key={skill}
              className="rounded-lg bg-rose-50 dark:bg-rose-500/10 px-2.5 py-0.5 text-xs font-medium text-rose-700 dark:text-rose-300"
            >
              {skill}
            </span>
          ))}
        </div>
      )}

      {/* Footer */}
      <div className="mt-4 flex items-center justify-between">
        <div className="flex items-center gap-4 text-xs text-gray-400 dark:text-gray-500">
          <span>
            {job.budget_type === "one_shot" ? t("oneShot") : t("longTerm")}
          </span>
          <span>
            {job.min_budget.toLocaleString()}&euro; &ndash; {job.max_budget.toLocaleString()}&euro;
          </span>
          <span className="flex items-center gap-1">
            <Clock className="h-3 w-3" />
            {new Date(job.created_at).toLocaleDateString()}
          </span>
        </div>

        {isOpen && (
          <button
            type="button"
            onClick={() => onClose(job.id)}
            disabled={isClosing}
            className={cn(
              "flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-xs font-medium",
              "text-gray-500 dark:text-gray-400 transition-all duration-200",
              "hover:bg-red-50 dark:hover:bg-red-900/20 hover:text-red-600 dark:hover:text-red-400",
              "disabled:opacity-50 disabled:cursor-not-allowed",
            )}
          >
            <X className="h-3 w-3" />
            {t("closeJob")}
          </button>
        )}
      </div>
    </div>
  )
}

/* -------------------------------------------------- */
/* Empty state                                        */
/* -------------------------------------------------- */

function EmptyState() {
  const t = useTranslations("job")

  return (
    <div className="rounded-xl border border-dashed border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-900 p-12 text-center">
      <Briefcase className="mx-auto h-10 w-10 text-gray-300 dark:text-gray-600" />
      <p className="mt-4 text-sm font-medium text-gray-500 dark:text-gray-400">
        {t("noJobs")}
      </p>
      <Link
        href="/jobs/create"
        className="mt-4 inline-flex items-center gap-2 rounded-xl px-5 py-2.5 text-sm font-semibold text-white gradient-primary transition-all duration-200 hover:shadow-glow active:scale-[0.98]"
      >
        <Plus className="h-4 w-4" strokeWidth={2} />
        {t("createJob")}
      </Link>
    </div>
  )
}

/* -------------------------------------------------- */
/* Skeleton loader                                    */
/* -------------------------------------------------- */

function JobListSkeleton() {
  return (
    <div className="space-y-3">
      {Array.from({ length: 3 }).map((_, i) => (
        <div
          key={i}
          className="rounded-2xl border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 p-5"
        >
          <div className="flex items-start justify-between gap-4">
            <div className="flex-1 space-y-2">
              <div className="h-5 w-3/4 rounded-lg bg-gray-200 dark:bg-gray-700 animate-shimmer" />
              <div className="h-4 w-full rounded-lg bg-gray-100 dark:bg-gray-800 animate-shimmer" />
            </div>
            <div className="h-6 w-16 rounded-full bg-gray-100 dark:bg-gray-800 animate-shimmer" />
          </div>
          <div className="mt-3 flex gap-1.5">
            <div className="h-5 w-16 rounded-lg bg-gray-100 dark:bg-gray-800 animate-shimmer" />
            <div className="h-5 w-20 rounded-lg bg-gray-100 dark:bg-gray-800 animate-shimmer" />
          </div>
        </div>
      ))}
    </div>
  )
}
