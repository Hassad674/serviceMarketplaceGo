"use client"

import { useParams } from "next/navigation"
import { useRouter } from "@i18n/navigation"
import { useQuery } from "@tanstack/react-query"
import { useHasPermission } from "@/shared/hooks/use-permissions"
import { getJob } from "@/features/job/api/job-api"
import { EditJobForm } from "@/features/job/components/edit-job-form"

export default function EditJobPage() {
  const params = useParams<{ id: string }>()
  const router = useRouter()
  const jobId = params.id
  const canEdit = useHasPermission("jobs.edit")

  const { data: job, isLoading } = useQuery({
    queryKey: ["jobs", jobId],
    queryFn: () => getJob(jobId),
  })

  if (isLoading) {
    return (
      <div className="mx-auto max-w-[680px] space-y-4 animate-shimmer">
        <div className="h-8 w-1/3 rounded-lg bg-slate-100 dark:bg-slate-800" />
        <div className="h-64 rounded-2xl bg-slate-100 dark:bg-slate-800" />
        <div className="h-48 rounded-2xl bg-slate-100 dark:bg-slate-800" />
      </div>
    )
  }

  if (!canEdit) {
    router.replace("/jobs")
    return null
  }

  if (!job) return null

  return <EditJobForm job={job} />
}
