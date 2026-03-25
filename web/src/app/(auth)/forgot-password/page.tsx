import { ForgotPasswordForm } from "@/features/auth/components/forgot-password-form"

export default function ForgotPasswordPage() {
  return (
    <div className="mx-auto w-full max-w-md space-y-6">
      <div className="text-center">
        <h1 className="text-2xl font-bold text-gray-900">Forgot password</h1>
        <p className="mt-2 text-sm text-gray-500">
          Enter your email to receive a reset link
        </p>
      </div>
      <ForgotPasswordForm />
    </div>
  )
}
