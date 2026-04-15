"use client"

import { useLocale, useTranslations } from "next-intl"
import { getFlagEmoji } from "@/shared/lib/profile/country-options"
import {
  getLanguageFlagCountry,
  getLanguageLabel,
} from "@/shared/lib/profile/language-options"

interface LanguagesDisplayCardProps {
  professional: string[]
  conversational: string[]
}

// LanguagesDisplayCard renders the shared language lists in read-only
// mode for public profile pages. Collapses to null when both arrays
// are empty so the public viewer never sees an empty card.
export function LanguagesDisplayCard({
  professional,
  conversational,
}: LanguagesDisplayCardProps) {
  const t = useTranslations("profile.languages")
  const locale = useLocale() === "fr" ? "fr" : "en"
  if (professional.length === 0 && conversational.length === 0) return null

  return (
    <section
      aria-labelledby="public-languages-title"
      className="bg-card border border-border rounded-xl p-6 shadow-sm"
    >
      <header className="mb-4">
        <h2
          id="public-languages-title"
          className="text-lg font-semibold text-foreground"
        >
          {t("sectionTitle")}
        </h2>
      </header>

      {professional.length > 0 ? (
        <LanguageGroup
          label={t("professionalLabel")}
          codes={professional}
          locale={locale}
          primary
        />
      ) : null}

      {conversational.length > 0 ? (
        <LanguageGroup
          label={t("conversationalLabel")}
          codes={conversational}
          locale={locale}
          primary={false}
        />
      ) : null}
    </section>
  )
}

interface LanguageGroupProps {
  label: string
  codes: string[]
  locale: "fr" | "en"
  primary: boolean
}

function LanguageGroup({ label, codes, locale, primary }: LanguageGroupProps) {
  return (
    <div className={primary ? "" : "mt-4"}>
      <p className="text-xs font-medium uppercase tracking-wider text-muted-foreground mb-2">
        {label}
      </p>
      <ul className="flex flex-wrap gap-1.5" aria-label={label}>
        {codes.map((code) => {
          const flag = getFlagEmoji(getLanguageFlagCountry(code))
          return (
            <li
              key={code}
              className="inline-flex items-center gap-1.5 rounded-full border border-border bg-muted/60 px-2.5 py-0.5 text-xs font-medium text-foreground"
            >
              {flag ? <span aria-hidden="true">{flag}</span> : null}
              {getLanguageLabel(code, locale)}
            </li>
          )
        })}
      </ul>
    </div>
  )
}
