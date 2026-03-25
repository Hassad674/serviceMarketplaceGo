import { getTranslations } from "next-intl/server"

export default async function FreelancesDirectoryPage() {
  const t = await getTranslations("pages")

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-gray-900">
          {t("freelancerDirectory")}
        </h1>
        <p className="mt-2 text-sm text-gray-500">
          {t("freelancerDirectoryDesc")}
        </p>
      </div>
      <div className="rounded-xl border border-dashed border-gray-300 bg-white p-12 text-center text-sm text-gray-400">
        {t("freelancerDirectoryPlaceholder")}
      </div>
    </div>
  )
}
