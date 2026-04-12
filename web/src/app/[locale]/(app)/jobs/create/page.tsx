"use client"

import { useRouter } from "@i18n/navigation"
import { useHasPermission } from "@/shared/hooks/use-permissions"
import { CreateJobForm } from "@/features/job/components/create-job-form"

export default function CreateJobPage() {
  const router = useRouter()
  const canCreate = useHasPermission("jobs.create")

  if (!canCreate) {
    router.replace("/jobs")
    return null
  }

  return <CreateJobForm />
}
