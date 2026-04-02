"use client"

import { useParams } from "next/navigation"
import { useQuery } from "@tanstack/react-query"
import { getJob } from "@/features/job/api/job-api"
import { EditJobForm } from "@/features/job/components/edit-job-form"

export default function EditJobPage() {
  const params = useParams<{ id: string }>()
  const jobId = params.id

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

  if (!job) return null

  return <EditJobForm job={job} />
}
