import { ResetPasswordForm } from "@/features/auth/components/reset-password-form"

export default async function ResetPasswordPage({
  searchParams,
}: {
  searchParams: Promise<{ token?: string }>
}) {
  const { token } = await searchParams

  return (
    <div className="mx-auto w-full max-w-md space-y-6">
      <div className="text-center">
        <h1 className="text-2xl font-bold text-gray-900">New password</h1>
        <p className="mt-2 text-sm text-gray-500">
          Choose a new password
        </p>
      </div>
      <ResetPasswordForm token={token || ""} />
    </div>
  )
}
