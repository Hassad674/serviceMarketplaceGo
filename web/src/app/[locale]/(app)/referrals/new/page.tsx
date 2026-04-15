import { ReferralCreationForm } from "@/features/referral/components/referral-creation-form"

// /referrals/new — apporteur creates a new business referral. The form
// posts to POST /api/v1/referrals and redirects to the detail page on
// success. Pre-fill via ?provider_id=… is supported in V2.
export default function NewReferralPage() {
  return (
    <main className="mx-auto max-w-3xl px-4 py-8">
      <header className="mb-6">
        <h1 className="text-2xl font-bold text-slate-900">
          Nouvelle mise en relation
        </h1>
        <p className="mt-1 text-sm text-slate-500">
          Présentez un prestataire à un client. Vous négociez la commission
          avec le prestataire en privé ; le client accepte ou refuse la mise
          en relation sans voir le taux.
        </p>
      </header>
      <ReferralCreationForm />
    </main>
  )
}
