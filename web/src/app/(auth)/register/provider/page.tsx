import { ProviderRegisterForm } from "@/features/auth/components/provider-register-form"

export default function RegisterProviderPage() {
  return (
    <div className="mx-auto w-full max-w-md space-y-6">
      <div className="text-center">
        <h1 className="text-2xl font-bold text-gray-900">Freelance Sign Up</h1>
        <p className="mt-2 text-sm text-muted-foreground">
          Create your freelance account
        </p>
      </div>
      <ProviderRegisterForm />
    </div>
  )
}
