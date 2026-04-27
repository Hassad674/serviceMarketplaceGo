"use client"

import { useMemo, useState } from "react"
import { ArrowRight, Loader2 } from "lucide-react"
import { useTranslations } from "next-intl"
import { AddressElement, Elements } from "@stripe/react-stripe-js"
import type { StripeAddressElementChangeEvent } from "@stripe/stripe-js"

import { stripePromise } from "@/shared/lib/stripe-client"

import { CountrySelector } from "./country-selector"
import { TrustSignals } from "./trust-signals"

type WizardAddress = {
  line1: string
  line2: string
  city: string
  postal_code: string
  country: string
}

type OnboardingWizardProps = {
  loading: boolean
  onSubmit: (country: string) => void
  /**
   * Optional — fired every time the user edits the AddressElement.
   * Lets the parent persist street/city/postal/country alongside the
   * country choice (Stripe Connect's embedded onboarding will collect
   * the same fields again, but we mirror them locally so the parent
   * page is free to forward them in a single round-trip later).
   */
  onAddressChange?: (address: WizardAddress) => void
}

const EMPTY_ADDRESS: WizardAddress = {
  line1: "",
  line2: "",
  city: "",
  postal_code: "",
  country: "",
}

export function OnboardingWizard({ loading, onSubmit, onAddressChange }: OnboardingWizardProps) {
  const t = useTranslations("paymentInfo")
  const [country, setCountry] = useState<string | null>(null)
  const [address, setAddress] = useState<WizardAddress>(EMPTY_ADDRESS)

  const handleSubmit = () => {
    if (!country) return
    onSubmit(country)
  }

  const handleAddressChange = (event: StripeAddressElementChangeEvent) => {
    const next: WizardAddress = {
      line1: event.value.address.line1 ?? "",
      line2: event.value.address.line2 ?? "",
      city: event.value.address.city ?? "",
      postal_code: event.value.address.postal_code ?? "",
      country: event.value.address.country ?? "",
    }
    setAddress(next)
    onAddressChange?.(next)
  }

  // Setup-mode Elements does not require a clientSecret — it is the
  // right mode for "collect info that won't be paid right now". The
  // currency is required by the type but unused for AddressElement.
  const elementsOptions = useMemo(
    () => ({ mode: "setup" as const, currency: "eur" }),
    [],
  )

  // AddressElement options — locked to the country the user picked
  // above so the dropdown is consistent with the wizard's country
  // selector. When no country has been chosen yet we leave the list
  // open so Stripe's autocomplete still works on first focus.
  const addressOptions = useMemo(
    () => ({
      mode: "billing" as const,
      autocomplete: { mode: "automatic" as const },
      ...(country ? { allowedCountries: [country] } : {}),
      defaultValues: {
        address: {
          line1: address.line1,
          line2: address.line2 || undefined,
          city: address.city,
          postal_code: address.postal_code,
          state: "",
          country: country ?? "",
        },
      },
    }),
    [country, address.line1, address.line2, address.city, address.postal_code],
  )

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

          {country ? (
            <div data-testid="address-section">
              <label className="mb-2 block text-[13px] font-semibold text-slate-900">
                {t("addressLabel")}
              </label>
              <Elements stripe={stripePromise} options={elementsOptions}>
                <AddressElement
                  options={addressOptions}
                  onChange={handleAddressChange}
                />
              </Elements>
              <p className="mt-2 text-[12px] text-slate-500">
                {t("addressHint")}
              </p>
            </div>
          ) : null}

          <button
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
          </button>
        </div>
      </div>

      <div className="mt-6">
        <TrustSignals />
      </div>
    </div>
  )
}
