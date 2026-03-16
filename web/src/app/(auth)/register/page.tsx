import { RegisterForm } from "@/features/auth/components/register-form"

export default function RegisterPage() {
  return (
    <div className="mx-auto w-full max-w-md space-y-6">
      <div className="text-center">
        <h1 className="text-2xl font-bold text-gray-900">Inscription</h1>
        <p className="mt-2 text-sm text-gray-500">
          Creez votre compte pour acceder a la plateforme
        </p>
      </div>
      <RegisterForm />
    </div>
  )
}
