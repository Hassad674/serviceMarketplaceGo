import { getTranslations } from "next-intl/server"
import { ApplicationList } from "@/features/job/components/application-list"

export default async function MyApplicationsPage() {
  const t = await getTranslations("opportunity")

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold text-slate-900 dark:text-white">
        {t("myApplications")}
      </h1>
      <ApplicationList />
    </div>
  )
}
