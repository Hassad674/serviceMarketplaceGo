import { getTranslations } from "next-intl/server"
import { OpportunityList } from "@/features/job/components/opportunity-list"

export default async function OpportunitiesPage() {
  const t = await getTranslations("opportunity")

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-slate-900 dark:text-white">
          {t("browse")}
        </h1>
        <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">
          {t("browseDesc")}
        </p>
      </div>
      <OpportunityList />
    </div>
  )
}
