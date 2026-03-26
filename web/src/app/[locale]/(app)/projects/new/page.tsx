import { getTranslations } from "next-intl/server"
import { CreateProjectForm } from "@/features/project/components/create-project-form"

export default async function NewProjectPage() {
  const t = await getTranslations("projects")

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold tracking-tight text-gray-900 dark:text-white">
        {t("createProject")}
      </h1>
      <CreateProjectForm />
    </div>
  )
}
