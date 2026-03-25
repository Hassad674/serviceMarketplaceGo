import { AgencyRegisterForm } from "@/features/auth/components/agency-register-form"

export default function RegisterAgencyPage() {
  return (
    <div className="w-full max-w-md space-y-6">
      <div className="text-center">
        <h1 className="text-2xl font-bold tracking-tight text-gray-900">
          Agency Sign Up
        </h1>
        <p className="mt-2 text-sm text-gray-500">
          Create your agency account
        </p>
      </div>
      <AgencyRegisterForm />
    </div>
  )
}
