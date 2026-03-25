import { useTranslations } from "next-intl"
import { LoginForm } from "@/features/auth/components/login-form"

export default function LoginPage() {
  const t = useTranslations("auth")

  return (
    <div className="w-full max-w-md space-y-6">
      <div className="text-center">
        <h1 className="text-2xl font-bold tracking-tight text-gray-900">
          {t("loginTitle")}
        </h1>
        <p className="mt-2 text-sm text-gray-500">
          {t("loginSubtitle")}
        </p>
      </div>
      <LoginForm />
    </div>
  )
}
