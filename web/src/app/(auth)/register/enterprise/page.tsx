import { EnterpriseRegisterForm } from "@/features/auth/components/enterprise-register-form"

export default function RegisterEnterprisePage() {
  return (
    <div className="mx-auto w-full max-w-md space-y-6">
      <div className="text-center">
        <h1 className="text-2xl font-bold text-gray-900">Inscription Entreprise</h1>
        <p className="mt-2 text-sm text-muted-foreground">
          Créez votre compte entreprise
        </p>
      </div>
      <EnterpriseRegisterForm />
    </div>
  )
}
