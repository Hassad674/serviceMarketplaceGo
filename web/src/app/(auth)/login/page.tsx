import { LoginForm } from "@/features/auth/components/login-form"

export default function LoginPage() {
  return (
    <div className="mx-auto w-full max-w-md space-y-6">
      <div className="text-center">
        <h1 className="text-2xl font-bold text-gray-900">Sign In</h1>
        <p className="mt-2 text-sm text-gray-500">
          Sign in to your account
        </p>
      </div>
      <LoginForm />
    </div>
  )
}
