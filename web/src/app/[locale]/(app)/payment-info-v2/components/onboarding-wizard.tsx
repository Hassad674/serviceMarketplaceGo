"use client"

import { useState } from "react"
import { ArrowRight, Loader2 } from "lucide-react"

import { BusinessTypeCard } from "../../test-embedded/components/business-type-card"
import { CountrySelector } from "../../test-embedded/components/country-selector"
import { TrustSignals } from "../../test-embedded/components/trust-signals"

type BusinessType = "individual" | "company"

type OnboardingWizardProps = {
  loading: boolean
  onSubmit: (country: string, businessType: BusinessType) => void
}

export function OnboardingWizard({ loading, onSubmit }: OnboardingWizardProps) {
  const [country, setCountry] = useState<string | null>(null)
  const [businessType, setBusinessType] = useState<BusinessType | null>(null)

  const handleSubmit = () => {
    if (!country || !businessType) return
    onSubmit(country, businessType)
  }

  return (
    <div className="mx-auto max-w-2xl">
      <div className="mb-8 text-center">
        <h2 className="text-3xl font-extrabold tracking-tight text-slate-900 sm:text-4xl">
          Configurons vos paiements
        </h2>
        <p className="mx-auto mt-3 max-w-md text-[15px] leading-relaxed text-slate-600">
          Quelques informations pour adapter le processus de vérification à votre situation.
        </p>
      </div>

      <div className="rounded-3xl border border-slate-100 bg-white p-6 shadow-sm sm:p-8">
        <div className="flex flex-col gap-6">
          <div>
            <label className="mb-2 block text-[13px] font-semibold text-slate-900">
              Pays de résidence fiscale
            </label>
            <CountrySelector value={country} onChange={setCountry} disabled={loading} />
            <p className="mt-2 text-[12px] text-slate-500">
              Le pays où vous déclarez vos revenus, indépendamment de votre nationalité.
            </p>
          </div>

          <div>
            <label className="mb-3 block text-[13px] font-semibold text-slate-900">
              Type de compte
            </label>
            <BusinessTypeCard
              value={businessType}
              onChange={setBusinessType}
              disabled={loading}
            />
          </div>

          <button
            onClick={handleSubmit}
            disabled={!country || !businessType || loading}
            className={`flex h-12 items-center justify-center gap-2 rounded-xl text-[15px] font-semibold transition-all ${
              !country || !businessType || loading
                ? "cursor-not-allowed bg-slate-100 text-slate-400"
                : "bg-gradient-to-r from-rose-500 to-rose-600 text-white shadow-md hover:shadow-lg hover:shadow-rose-500/25 active:scale-[0.98]"
            }`}
          >
            {loading ? (
              <>
                <Loader2 className="h-4 w-4 animate-spin" aria-hidden />
                Préparation...
              </>
            ) : (
              <>
                Continuer
                <ArrowRight className="h-4 w-4" aria-hidden />
              </>
            )}
          </button>
        </div>
      </div>

      <div className="mt-6">
        <TrustSignals />
      </div>
    </div>
  )
}
