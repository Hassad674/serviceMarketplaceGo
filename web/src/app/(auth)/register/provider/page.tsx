import { ProviderRegisterForm } from "@/features/auth/components/provider-register-form"

export default function RegisterProviderPage() {
  return (
    <div className="w-full max-w-md space-y-6">
      <div className="text-center">
        <h1 className="text-2xl font-bold tracking-tight text-gray-900">
          Freelance Sign Up
        </h1>
        <p className="mt-2 text-sm text-gray-500">
          Create your freelance account
        </p>
      </div>
      <ProviderRegisterForm />
    </div>
  )
}
