import { useTranslations } from "next-intl"
import { AgencyRegisterForm } from "@/features/auth/components/agency-register-form"

export default function RegisterAgencyPage() {
  const t = useTranslations("auth")

  return (
    <div className="w-full max-w-md space-y-6">
      <div className="text-center">
        <h1 className="text-2xl font-bold tracking-tight text-gray-900 dark:text-white">
          {t("roleAgency")}
        </h1>
        <p className="mt-2 text-sm text-gray-500 dark:text-gray-400">
          {t("createAgencyAccount")}
        </p>
      </div>
      <AgencyRegisterForm />
    </div>
  )
}
