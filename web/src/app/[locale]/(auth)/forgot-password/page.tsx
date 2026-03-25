import { useTranslations } from "next-intl"
import { ForgotPasswordForm } from "@/features/auth/components/forgot-password-form"

export default function ForgotPasswordPage() {
  const t = useTranslations("auth")

  return (
    <div className="w-full max-w-md space-y-6">
      <div className="text-center">
        <h1 className="text-2xl font-bold tracking-tight text-gray-900">
          {t("forgotTitle")}
        </h1>
        <p className="mt-2 text-sm text-gray-500">
          {t("forgotSubtitle")}
        </p>
      </div>
      <ForgotPasswordForm />
    </div>
  )
}
