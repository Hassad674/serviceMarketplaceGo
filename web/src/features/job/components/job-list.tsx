"use client"

import { useState, useRef, useEffect } from "react"
import { Briefcase, Plus, Clock, Users, MoreVertical, Trash2, XCircle, Pencil, RotateCcw } from "lucide-react"
import { useTranslations } from "next-intl"
import { Link, useRouter } from "@i18n/navigation"
import { cn } from "@/shared/lib/utils"
import { useHasPermission } from "@/shared/hooks/use-permissions"
import { useMyJobs, useCloseJob, useReopenJob, useDeleteJob } from "../hooks/use-jobs"
import type { JobWithCountsResponse } from "../types"
import { Button } from "@/shared/components/ui/button"

export function JobList() {
  const t = useTranslations("job")
  const { data, isLoading, error } = useMyJobs()
  const closeJob = useCloseJob()
  const reopenJob = useReopenJob()
  const deleteJob = useDeleteJob()
  const canCreate = useHasPermission("jobs.create")
  const canEdit = useHasPermission("jobs.edit")
  const canDelete = useHasPermission("jobs.delete")

  if (isLoading) {
    return (
      <div className="space-y-6">
        <PageHeader canCreate={canCreate} />
        <JobListSkeleton />
      </div>
    )
  }
  if (error) {
    return (
      <div className="space-y-6">
        <PageHeader canCreate={canCreate} />
        <div className="rounded-xl border border-red-200 dark:border-red-800 bg-red-50 dark:bg-red-900/20 p-6 text-sm text-red-700 dark:text-red-300">
          {t("errorLoading")}
        </div>
      </div>
    )
  }

  const jobs = data?.data ?? []
  if (jobs.length === 0) {
    return (
      <div className="space-y-6">
        <PageHeader canCreate={canCreate} />
        <EmptyState canCreate={canCreate} />
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <PageHeader canCreate={canCreate} />
      <div className="space-y-3">
        {jobs.map((job) => (
          <JobCard
            key={job.id}
            job={job}
            canEdit={canEdit}
            canDelete={canDelete}
            onClose={(id) => closeJob.mutate(id)}
            onReopen={(id) => reopenJob.mutate(id)}
            onDelete={(id) => { if (confirm(t("deleteConfirmJob"))) deleteJob.mutate(id) }}
            isActing={closeJob.isPending || reopenJob.isPending || deleteJob.isPending}
          />
        ))}
      </div>
    </div>
  )
}

type JobCardProps = {
  job: JobWithCountsResponse
  canEdit: boolean
  canDelete: boolean
  onClose: (id: string) => void
  onReopen: (id: string) => void
  onDelete: (id: string) => void
  isActing: boolean
}

function JobCard({ job, canEdit, canDelete, onClose, onReopen, onDelete, isActing }: JobCardProps) {
  const t = useTranslations("job")
  const router = useRouter()
  const isOpen = job.status === "open"
  const [menuOpen, setMenuOpen] = useState(false)
  const menuRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) setMenuOpen(false)
    }
    if (menuOpen) document.addEventListener("mousedown", handleClickOutside)
    return () => document.removeEventListener("mousedown", handleClickOutside)
  }, [menuOpen])

  return (
    <div
      onClick={() => router.push(`/jobs/${job.id}`)}
      className={cn(
        "rounded-2xl border bg-white dark:bg-gray-900 p-5 cursor-pointer",
        "border-gray-200 dark:border-gray-700",
        "transition-all duration-200 hover:shadow-md hover:border-rose-200 dark:hover:border-rose-800",
      )}
    >
      <div className="flex items-start justify-between gap-4">
        <div className="min-w-0 flex-1">
          <h3 className="truncate text-base font-semibold text-gray-900 dark:text-white">{job.title}</h3>
          <p className="mt-1 line-clamp-2 text-sm text-gray-500 dark:text-gray-400">{job.description}</p>
        </div>

        <div className="flex items-center gap-2 shrink-0">
          {/* Application count badges */}
          {job.total_applicants > 0 && (
            <div className="flex items-center gap-1.5">
              <span className="flex items-center gap-1 text-xs text-slate-500">
                <Users className="h-3.5 w-3.5" />
                {job.total_applicants}
              </span>
              {job.new_applicants > 0 && (
                <span className="rounded-full bg-rose-500 px-1.5 py-0.5 text-[10px] font-bold text-white leading-none">
                  {job.new_applicants}
                </span>
              )}
            </div>
          )}

          {/* Status badge */}
          <span className={cn(
            "rounded-full px-3 py-1 text-xs font-semibold",
            isOpen
              ? "bg-green-100 dark:bg-green-500/20 text-green-700 dark:text-green-300"
              : "bg-gray-100 dark:bg-gray-800 text-gray-500 dark:text-gray-400",
          )}>
            {isOpen ? t("statusOpen") : t("statusClosed")}
          </span>

          {/* 3-dot menu — hidden when the user has neither edit nor delete permission */}
          {(canEdit || canDelete) && (
          <div className="relative" ref={menuRef}>
            <Button variant="ghost" size="auto"
              type="button"
              onClick={(e) => { e.stopPropagation(); setMenuOpen(!menuOpen) }}
              className="rounded-lg p-1 hover:bg-slate-100 dark:hover:bg-slate-700 transition-colors"
            >
              <MoreVertical className="h-4 w-4 text-slate-400" />
            </Button>
            {menuOpen && (
              <div
                className="absolute right-0 top-full mt-1 z-10 w-44 rounded-xl border border-slate-200 bg-white shadow-lg dark:border-slate-600 dark:bg-slate-800 py-1"
                onClick={(e) => e.stopPropagation()}
              >
                {canEdit && (
                <Button variant="ghost" size="auto"
                  type="button"
                  onClick={() => { setMenuOpen(false); router.push(`/jobs/${job.id}/edit`) }}
                  className="w-full flex items-center gap-2 px-3 py-2 text-sm text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-slate-700"
                >
                  <Pencil className="h-4 w-4" />
                  {t("editJob")}
                </Button>
                )}
                {canEdit && (isOpen ? (
                  <Button variant="ghost" size="auto"
                    type="button"
                    onClick={() => { setMenuOpen(false); onClose(job.id) }}
                    disabled={isActing}
                    className="w-full flex items-center gap-2 px-3 py-2 text-sm text-amber-600 hover:bg-amber-50 dark:hover:bg-amber-500/10"
                  >
                    <XCircle className="h-4 w-4" />
                    {t("closeJob")}
                  </Button>
                ) : (
                  <Button variant="ghost" size="auto"
                    type="button"
                    onClick={() => { setMenuOpen(false); onReopen(job.id) }}
                    disabled={isActing}
                    className="w-full flex items-center gap-2 px-3 py-2 text-sm text-green-600 hover:bg-green-50 dark:hover:bg-green-500/10"
                  >
                    <RotateCcw className="h-4 w-4" />
                    {t("reopenJob")}
                  </Button>
                ))}
                {canDelete && (
                <Button variant="ghost" size="auto"
                  type="button"
                  onClick={() => { setMenuOpen(false); onDelete(job.id) }}
                  disabled={isActing}
                  className="w-full flex items-center gap-2 px-3 py-2 text-sm text-red-600 hover:bg-red-50 dark:hover:bg-red-500/10"
                >
                  <Trash2 className="h-4 w-4" />
                  {t("deleteJob")}
                </Button>
                )}
              </div>
            )}
          </div>
          )}
        </div>
      </div>

      {/* Skills */}
      {job.skills.length > 0 && (
        <div className="mt-3 flex flex-wrap gap-1.5">
          {job.skills.map((skill) => (
            <span key={skill} className="rounded-lg bg-rose-50 dark:bg-rose-500/10 px-2.5 py-0.5 text-xs font-medium text-rose-700 dark:text-rose-300">
              {skill}
            </span>
          ))}
        </div>
      )}

      {/* Footer */}
      <div className="mt-4 flex items-center gap-4 text-xs text-gray-400 dark:text-gray-500">
        <span>{job.budget_type === "one_shot" ? t("oneShot") : t("longTerm")}</span>
        <span>{job.min_budget.toLocaleString()}&euro; &ndash; {job.max_budget.toLocaleString()}&euro;</span>
        <span className="flex items-center gap-1">
          <Clock className="h-3 w-3" />
          {new Date(job.created_at).toLocaleDateString("fr-FR")}
        </span>
      </div>
    </div>
  )
}

function PageHeader({ canCreate }: { canCreate: boolean }) {
  const t = useTranslations("job")
  return (
    <div className="flex items-center justify-between">
      <h1 className="text-2xl font-bold tracking-tight text-gray-900 dark:text-white">
        {t("title")}
      </h1>
      {canCreate && (
        <Link
          href="/jobs/create"
          className="inline-flex items-center gap-2 rounded-xl px-5 py-2.5 text-sm font-semibold text-white gradient-primary transition-all duration-200 hover:shadow-glow active:scale-[0.98]"
        >
          <Plus className="h-4 w-4" strokeWidth={2} />
          {t("createJob")}
        </Link>
      )}
    </div>
  )
}

function EmptyState({ canCreate }: { canCreate: boolean }) {
  const t = useTranslations("job")
  return (
    <div className="rounded-xl border border-dashed border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-900 p-12 text-center">
      <Briefcase className="mx-auto h-10 w-10 text-gray-300 dark:text-gray-600" />
      <p className="mt-4 text-sm font-medium text-gray-500 dark:text-gray-400">{t("noJobs")}</p>
      {canCreate && (
        <Link
          href="/jobs/create"
          className="mt-4 inline-flex items-center gap-2 rounded-xl px-5 py-2.5 text-sm font-semibold text-white gradient-primary transition-all duration-200 hover:shadow-glow active:scale-[0.98]"
        >
          <Plus className="h-4 w-4" strokeWidth={2} />
          {t("createJob")}
        </Link>
      )}
    </div>
  )
}

function JobListSkeleton() {
  return (
    <div className="space-y-3">
      {Array.from({ length: 3 }).map((_, i) => (
        <div key={i} className="rounded-2xl border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 p-5">
          <div className="flex items-start justify-between gap-4">
            <div className="flex-1 space-y-2">
              <div className="h-5 w-3/4 rounded-lg bg-gray-200 dark:bg-gray-700 animate-shimmer" />
              <div className="h-4 w-full rounded-lg bg-gray-100 dark:bg-gray-800 animate-shimmer" />
            </div>
            <div className="h-6 w-16 rounded-full bg-gray-100 dark:bg-gray-800 animate-shimmer" />
          </div>
        </div>
      ))}
    </div>
  )
}
