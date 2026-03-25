import { AgencyRegisterForm } from "@/features/auth/components/agency-register-form"

export default function RegisterAgencyPage() {
  return (
    <div className="mx-auto w-full max-w-md space-y-6">
      <div className="text-center">
        <h1 className="text-2xl font-bold text-gray-900">Agency Sign Up</h1>
        <p className="mt-2 text-sm text-muted-foreground">
          Create your agency account
        </p>
      </div>
      <AgencyRegisterForm />
    </div>
  )
}
