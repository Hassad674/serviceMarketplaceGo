import { getTranslations } from "next-intl/server"
import { ResetPasswordForm } from "@/features/auth/components/reset-password-form"

export default async function ResetPasswordPage({
  searchParams,
}: {
  searchParams: Promise<{ token?: string }>
}) {
  const { token } = await searchParams
  const t = await getTranslations("auth")

  return (
    <div className="w-full max-w-md space-y-6">
      <div className="text-center">
        <h1 className="text-2xl font-bold tracking-tight text-gray-900">
          {t("resetTitle")}
        </h1>
        <p className="mt-2 text-sm text-gray-500">
          {t("resetSubtitle")}
        </p>
      </div>
      <ResetPasswordForm token={token || ""} />
    </div>
  )
}
