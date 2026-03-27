import { Plus } from "lucide-react"
import { getTranslations } from "next-intl/server"
import { Link } from "@i18n/navigation"
import { JobList } from "@/features/job/components/job-list"

export default async function JobsListPage() {
  const t = await getTranslations("job")

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold tracking-tight text-gray-900 dark:text-white">
          {t("title")}
        </h1>
        <Link
          href="/jobs/create"
          className="inline-flex items-center gap-2 rounded-xl px-5 py-2.5 text-sm font-semibold text-white gradient-primary transition-all duration-200 hover:shadow-glow active:scale-[0.98]"
        >
          <Plus className="h-4 w-4" strokeWidth={2} />
          {t("createJob")}
        </Link>
      </div>

      {/* Job list (client component with TanStack Query) */}
      <JobList />
    </div>
  )
}
