import { ProviderRegisterForm } from "@/features/auth/components/provider-register-form"

export default function RegisterProviderPage() {
  return (
    <div className="mx-auto w-full max-w-md space-y-6">
      <div className="text-center">
        <h1 className="text-2xl font-bold text-gray-900">Inscription Freelance</h1>
        <p className="mt-2 text-sm text-muted-foreground">
          Créez votre compte freelance
        </p>
      </div>
      <ProviderRegisterForm />
    </div>
  )
}
