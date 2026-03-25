import { getTranslations } from "next-intl/server"

export default async function AgenciesDirectoryPage() {
  const t = await getTranslations("pages")

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
          {t("agencyDirectory")}
        </h1>
        <p className="mt-2 text-sm text-gray-500 dark:text-gray-400">
          {t("agencyDirectoryDesc")}
        </p>
      </div>
      <div className="rounded-xl border border-dashed border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-900 p-12 text-center text-sm text-gray-400 dark:text-gray-500">
        {t("agencyDirectoryPlaceholder")}
      </div>
    </div>
  )
}
