import { useTranslations } from "next-intl"
import { ProviderRegisterForm } from "@/features/auth/components/provider-register-form"

export default function RegisterProviderPage() {
  const t = useTranslations("auth")

  return (
    <div className="w-full max-w-md space-y-6">
      <div className="text-center">
        <h1 className="text-2xl font-bold tracking-tight text-gray-900">
          {t("roleFreelance")}
        </h1>
        <p className="mt-2 text-sm text-gray-500">
          {t("createFreelanceAccount")}
        </p>
      </div>
      <ProviderRegisterForm />
    </div>
  )
}
