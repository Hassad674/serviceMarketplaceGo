"use client"

import { useState, useMemo } from "react"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import { useHasPermission } from "@/shared/hooks/use-permissions"
import { useMyJobs, useCloseJob, useReopenJob, useDeleteJob } from "../hooks/use-jobs"
import { JobListHeader } from "./job-list-header"
import { JobListFilterPills, type JobStatusFilter } from "./job-list-filter-pills"
import { JobListCard } from "./job-list-card"
import { JobListEmptyState, JobListSkeleton } from "./job-list-states"

// W-06 — Mes annonces (entreprise listing). Soleil v2 visual port.
//
// Thin orchestrator: composes the editorial header, the status filter
// pills, and the job cards. Loading / error / empty branches use the
// dedicated state components. Sub-components (header, pills, card,
// states) live in sibling files to keep every file under the 600-line
// cap mandated by web/CLAUDE.md.
//
// Data shape note: only `open` and `closed` statuses are exposed by
// the backend. The JSX source's En pause / Brouillon / Archivée
// filters are SKIPPED + FLAGGED in the batch report — they would
// require a backend status enum extension.

export function JobList() {
  const t = useTranslations("job")
  const { data, isLoading, error } = useMyJobs()
  const closeJob = useCloseJob()
  const reopenJob = useReopenJob()
  const deleteJob = useDeleteJob()
  const canCreate = useHasPermission("jobs.create")
  const canEdit = useHasPermission("jobs.edit")
  const canDelete = useHasPermission("jobs.delete")
  const [filter, setFilter] = useState<JobStatusFilter>("all")

  const jobs = useMemo(() => data?.data ?? [], [data])
  const counts = useMemo(() => {
    const open = jobs.filter((j) => j.status === "open").length
    return { all: jobs.length, open, closed: jobs.length - open }
  }, [jobs])
  const filtered = useMemo(() => {
    if (filter === "all") return jobs
    return jobs.filter((j) => j.status === filter)
  }, [jobs, filter])

  if (isLoading) {
    return (
      <div className="mx-auto w-full max-w-[1100px] space-y-7 px-4 py-8 sm:px-6">
        <JobListHeader canCreate={canCreate} />
        <JobListSkeleton />
      </div>
    )
  }

  if (error) {
    return (
      <div className="mx-auto w-full max-w-[1100px] space-y-7 px-4 py-8 sm:px-6">
        <JobListHeader canCreate={canCreate} />
        <div
          className={cn(
            "rounded-2xl border border-destructive/30 bg-destructive/5 p-6",
            "text-sm font-medium text-destructive",
          )}
          role="alert"
        >
          {t("errorLoading")}
        </div>
      </div>
    )
  }

  if (jobs.length === 0) {
    return (
      <div className="mx-auto w-full max-w-[1100px] space-y-7 px-4 py-8 sm:px-6">
        <JobListHeader canCreate={canCreate} />
        <JobListEmptyState canCreate={canCreate} />
      </div>
    )
  }

  return (
    <div className="mx-auto w-full max-w-[1100px] space-y-7 px-4 py-8 sm:px-6">
      <JobListHeader canCreate={canCreate} />
      <JobListFilterPills filter={filter} onChange={setFilter} counts={counts} />
      {filtered.length === 0 ? (
        <p className="rounded-2xl border border-dashed border-border bg-card p-10 text-center text-sm text-muted-foreground">
          {t("noJobs")}
        </p>
      ) : (
        <div className="space-y-4">
          {filtered.map((job) => (
            <JobListCard
              key={job.id}
              job={job}
              canEdit={canEdit}
              canDelete={canDelete}
              onClose={(id) => closeJob.mutate(id)}
              onReopen={(id) => reopenJob.mutate(id)}
              onDelete={(id) => {
                if (confirm(t("deleteConfirmJob"))) deleteJob.mutate(id)
              }}
              isActing={
                closeJob.isPending || reopenJob.isPending || deleteJob.isPending
              }
            />
          ))}
        </div>
      )}
    </div>
  )
}
