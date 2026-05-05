"use client"

import { useLocale, useTranslations } from "next-intl"
import {
  getCountryLabel,
  getFlagEmoji,
} from "@/shared/lib/profile/country-options"
import type { WorkMode } from "@/shared/components/ui/location-row"

interface LocationDisplayCardProps {
  city: string
  countryCode: string
  workMode: WorkMode[]
  travelRadiusKm: number | null
}

const WORK_MODE_KEYS: Record<WorkMode, string> = {
  remote: "workModeRemote",
  on_site: "workModeOnSite",
  hybrid: "workModeHybrid",
}

// LocationDisplayCard renders the shared location block in read-only
// mode for public profile pages. Collapses to null when every field
// is empty so the public viewer never sees an empty card.
export function LocationDisplayCard({
  city,
  countryCode,
  workMode,
  travelRadiusKm,
}: LocationDisplayCardProps) {
  const t = useTranslations("profile.location")
  const locale = useLocale() === "fr" ? "fr" : "en"

  const hasLocation = city.trim() !== "" || countryCode.trim() !== ""
  const hasWorkMode = workMode.length > 0
  if (!hasLocation && !hasWorkMode) return null

  const countryLabel = countryCode ? getCountryLabel(countryCode, locale) : ""
  const flag = countryCode ? getFlagEmoji(countryCode) : ""

  return (
    <section
      aria-labelledby="public-location-title"
      className="bg-card border border-border rounded-2xl p-7 shadow-[var(--shadow-card)]"
    >
      <header className="mb-4">
        <h2
          id="public-location-title"
          className="font-serif text-xl font-medium tracking-[-0.005em] text-foreground"
        >
          {t("sectionTitle")}
        </h2>
      </header>

      {hasLocation ? (
        <p className="text-[15px] text-foreground">
          {flag ? <span className="mr-1.5">{flag}</span> : null}
          <strong className="font-semibold">
            {[city, countryLabel].filter(Boolean).join(", ")}
          </strong>
        </p>
      ) : null}

      {hasWorkMode ? (
        <ul
          aria-label={t("workModeLabel")}
          className="mt-4 flex flex-wrap gap-1.5"
        >
          {workMode.map((mode) => (
            <li
              key={mode}
              className="inline-flex items-center rounded-full border border-border bg-muted px-3 py-1 text-xs font-medium text-foreground"
            >
              {t(WORK_MODE_KEYS[mode])}
            </li>
          ))}
          {travelRadiusKm !== null ? (
            <li className="inline-flex items-center rounded-full border border-border bg-muted px-3 py-1 text-xs font-medium text-muted-foreground">
              {t("travelRadiusShort", { km: travelRadiusKm })}
            </li>
          ) : null}
        </ul>
      ) : null}
    </section>
  )
}
