"use client"

import { useState } from "react"
import { ArrowRight, Loader2 } from "lucide-react"
import { useTranslations } from "next-intl"

import { CountrySelector } from "./country-selector"
import { TrustSignals } from "./trust-signals"

import { Button } from "@/shared/components/ui/button"
type OnboardingWizardProps = {
  loading: boolean
  onSubmit: (country: string) => void
}

export function OnboardingWizard({ loading, onSubmit }: OnboardingWizardProps) {
  const t = useTranslations("paymentInfo")
  const [country, setCountry] = useState<string | null>(null)

  const handleSubmit = () => {
    if (!country) return
    onSubmit(country)
  }

  return (
    <div className="mx-auto max-w-2xl">
      <div className="mb-8 text-center">
        <h2 className="text-3xl font-extrabold tracking-tight text-slate-900 sm:text-4xl">
          {t("title")}
        </h2>
        <p className="mx-auto mt-3 max-w-md text-[15px] leading-relaxed text-slate-600">
          {t("subtitle")}
        </p>
      </div>

      <div className="rounded-3xl border border-slate-100 bg-white p-6 shadow-sm sm:p-8">
        <div className="flex flex-col gap-6">
          <div>
            <label className="mb-2 block text-[13px] font-semibold text-slate-900">
              {t("countryLabel")}
            </label>
            <CountrySelector value={country} onChange={setCountry} disabled={loading} />
            <p className="mt-2 text-[12px] text-slate-500">
              {t("countryHint")}
            </p>
          </div>

          <Button variant="ghost" size="auto"
            onClick={handleSubmit}
            disabled={!country || loading}
            className={`flex h-12 items-center justify-center gap-2 rounded-xl text-[15px] font-semibold transition-all ${
              !country || loading
                ? "cursor-not-allowed bg-slate-100 text-slate-400"
                : "bg-gradient-to-r from-rose-500 to-rose-600 text-white shadow-md hover:shadow-lg hover:shadow-rose-500/25 active:scale-[0.98]"
            }`}
          >
            {loading ? (
              <>
                <Loader2 className="h-4 w-4 animate-spin" aria-hidden />
                {t("loading")}
              </>
            ) : (
              <>
                {t("continue")}
                <ArrowRight className="h-4 w-4" aria-hidden />
              </>
            )}
          </Button>
        </div>
      </div>

      <div className="mt-6">
        <TrustSignals />
      </div>
    </div>
  )
}
