import type { Metadata } from "next"
import { BillingProfileForm } from "@/features/invoicing/components/billing-profile-form"

export const metadata: Metadata = {
  title: "Profil de facturation",
}

export default function BillingProfilePage() {
  return (
    <div className="mx-auto max-w-3xl px-4 py-8">
      <header className="mb-6">
        <h1 className="text-2xl font-bold text-slate-900 dark:text-white">
          Profil de facturation
        </h1>
        <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">
          Ces informations apparaissent sur les factures que la plateforme
          émet à ton organisation. Elles doivent être complètes pour pouvoir
          retirer ton solde et souscrire à un abonnement Premium.
        </p>
      </header>
      <BillingProfileForm />
    </div>
  )
}
