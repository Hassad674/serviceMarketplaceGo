"use client"

import { useState } from "react"
import { ArrowRight, Loader2 } from "lucide-react"
import { useTranslations } from "next-intl"

import { cn } from "@/shared/lib/utils"

import { CountrySelector } from "./country-selector"
import { TrustSignals } from "./trust-signals"

import { Button } from "@/shared/components/ui/button"
type OnboardingWizardProps = {
  loading: boolean
  onSubmit: (country: string) => void
}

export function OnboardingWizard({ loading, onSubmit }: OnboardingWizardProps) {
  const t = useTranslations("paymentInfo")
  const tW05 = useTranslations("kyc_w05")
  const [country, setCountry] = useState<string | null>(null)

  const handleSubmit = () => {
    if (!country) return
    onSubmit(country)
  }

  return (
    <div className="mx-auto max-w-2xl">
      <div className="mb-8 text-center">
        <h2 className="font-serif text-[28px] font-medium leading-[1.1] tracking-[-0.02em] text-foreground sm:text-[34px]">
          {tW05("wizardTitlePart1")}{" "}
          <span className="italic text-primary">{tW05("wizardTitleAccent")}</span>
        </h2>
        <p className="mx-auto mt-3 max-w-md text-[14px] leading-relaxed text-muted-foreground">
          {t("subtitle")}
        </p>
      </div>

      <div className="rounded-2xl border border-border bg-card p-6 shadow-card sm:p-8">
        <div className="flex flex-col gap-6">
          <div>
            <label className="mb-2 block text-[13px] font-semibold text-foreground">
              {t("countryLabel")}
            </label>
            <CountrySelector value={country} onChange={setCountry} disabled={loading} />
            <p className="mt-2 text-[12px] text-muted-foreground">
              {t("countryHint")}
            </p>
          </div>

          <Button
            variant="ghost"
            size="auto"
            onClick={handleSubmit}
            disabled={!country || loading}
            className={cn(
              "flex h-12 items-center justify-center gap-2 rounded-full text-[15px] font-semibold transition-all",
              !country || loading
                ? "cursor-not-allowed bg-border text-subtle-foreground"
                : "bg-primary text-primary-foreground hover:bg-primary-deep hover:shadow-[0_4px_14px_rgba(232,93,74,0.28)] active:scale-[0.98]",
            )}
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
