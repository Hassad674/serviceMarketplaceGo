import { getTranslations } from "next-intl/server"

export default async function EnterpriseProjectsPage() {
  const t = await getTranslations("pages")

  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-bold text-gray-900">{t("myProjects")}</h1>
      <p className="text-sm text-gray-500">
        {t("projectsDesc")}
      </p>
    </div>
  )
}
