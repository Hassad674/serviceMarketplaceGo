import { Briefcase, Plus } from "lucide-react"
import { getTranslations } from "next-intl/server"
import { Link } from "@i18n/navigation"

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

      {/* Empty state */}
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
    </div>
  )
}
