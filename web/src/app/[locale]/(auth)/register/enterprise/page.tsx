import { useTranslations } from "next-intl"
import { EnterpriseRegisterForm } from "@/features/auth/components/enterprise-register-form"

export default function RegisterEnterprisePage() {
  const t = useTranslations("auth")

  return (
    <div className="w-full max-w-md space-y-6">
      <div className="text-center">
        <h1 className="text-2xl font-bold tracking-tight text-gray-900">
          {t("roleEnterprise")}
        </h1>
        <p className="mt-2 text-sm text-gray-500">
          {t("createEnterpriseAccount")}
        </p>
      </div>
      <EnterpriseRegisterForm />
    </div>
  )
}
