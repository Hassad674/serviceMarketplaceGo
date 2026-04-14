"use client"

import { useCallback, useState } from "react"
import { Check, Loader2 } from "lucide-react"
import { useLocale, useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import type { WorkMode } from "../api/profile-api"
import {
  COUNTRY_OPTIONS,
  getCountryLabel,
} from "../lib/country-options"
import { useProfile } from "../hooks/use-profile"
import { useUpdateLocation } from "../hooks/use-update-location"

const ALL_WORK_MODES: WorkMode[] = ["remote", "on_site", "hybrid"]

interface LocationSectionProps {
  orgType: string | undefined
  readOnly?: boolean
}

export function LocationSection({
  orgType,
  readOnly = false,
}: LocationSectionProps) {
  const t = useTranslations("profile.location")
  const locale = useLocale() === "fr" ? "fr" : "en"
  const { data: profile } = useProfile()
  const mutation = useUpdateLocation()

  // Derive persisted values from primitive profile fields so they stay
  // referentially stable across renders. Sync local draft to persisted
  // state during render (React's recommended pattern) instead of in
  // effects.
  const persistedCity = profile?.city ?? ""
  const persistedCountry = profile?.country_code ?? ""
  const persistedWorkModeKey = (profile?.work_mode ?? []).join(",")
  const persistedRadius = profile?.travel_radius_km ?? null

  const [prevKey, setPrevKey] = useState(
    `${persistedCity}|${persistedCountry}|${persistedWorkModeKey}|${persistedRadius ?? ""}`,
  )
  const [city, setCity] = useState(persistedCity)
  const [country, setCountry] = useState(persistedCountry)
  const [workMode, setWorkMode] = useState<WorkMode[]>(
    profile?.work_mode ?? [],
  )
  const [radius, setRadius] = useState<string>(
    persistedRadius === null ? "" : String(persistedRadius),
  )

  const currentKey = `${persistedCity}|${persistedCountry}|${persistedWorkModeKey}|${persistedRadius ?? ""}`
  if (prevKey !== currentKey) {
    setPrevKey(currentKey)
    setCity(persistedCity)
    setCountry(persistedCountry)
    setWorkMode(persistedWorkModeKey ? (persistedWorkModeKey.split(",") as WorkMode[]) : [])
    setRadius(persistedRadius === null ? "" : String(persistedRadius))
  }

  const isDirty =
    city !== persistedCity ||
    country !== persistedCountry ||
    workMode.join(",") !== persistedWorkModeKey ||
    (radius === "" ? persistedRadius !== null : Number(radius) !== persistedRadius)

  const toggleWorkMode = (mode: WorkMode) => {
    setWorkMode((current) =>
      current.includes(mode)
        ? current.filter((m) => m !== mode)
        : [...current, mode],
    )
  }

  const handleSave = useCallback(() => {
    const radiusValue = radius.trim() === "" ? null : Number(radius)
    const sanitizedRadius =
      radiusValue !== null && Number.isFinite(radiusValue) && radiusValue >= 0
        ? Math.round(radiusValue)
        : null
    mutation.mutate({
      city: city.trim(),
      country_code: country,
      work_mode: workMode,
      travel_radius_km: sanitizedRadius,
    })
  }, [city, country, mutation, radius, workMode])

  if (orgType === "enterprise") return null
  if (readOnly) return null

  const needsRadius = workMode.includes("on_site") || workMode.includes("hybrid")

  return (
    <section
      aria-labelledby="location-section-title"
      className="bg-card border border-border rounded-xl p-6 shadow-sm"
    >
      <header className="mb-4 flex flex-col gap-1">
        <h2
          id="location-section-title"
          className="text-lg font-semibold text-foreground"
        >
          {t("sectionTitle")}
        </h2>
        <p className="text-sm text-muted-foreground">{t("sectionSubtitle")}</p>
      </header>

      <div className="grid gap-4 sm:grid-cols-2">
        <div>
          <label
            htmlFor="location-city"
            className="block text-sm font-medium text-foreground mb-1.5"
          >
            {t("cityLabel")}
          </label>
          <input
            id="location-city"
            type="text"
            value={city}
            onChange={(e) => setCity(e.target.value)}
            placeholder={t("cityPlaceholder")}
            className="w-full h-10 rounded-lg border border-border bg-background px-3 text-sm shadow-xs focus:border-primary focus:ring-4 focus:ring-primary/10 focus:outline-none"
          />
        </div>

        <div>
          <label
            htmlFor="location-country"
            className="block text-sm font-medium text-foreground mb-1.5"
          >
            {t("countryLabel")}
          </label>
          <select
            id="location-country"
            value={country}
            onChange={(e) => setCountry(e.target.value)}
            className="w-full h-10 rounded-lg border border-border bg-background px-3 text-sm shadow-xs focus:border-primary focus:ring-4 focus:ring-primary/10 focus:outline-none"
          >
            <option value="">{t("countryPlaceholder")}</option>
            {COUNTRY_OPTIONS.map((option) => (
              <option key={option.code} value={option.code}>
                {getCountryLabel(option.code, locale)}
              </option>
            ))}
          </select>
        </div>
      </div>

      <div className="mt-4">
        <p className="block text-sm font-medium text-foreground mb-2">
          {t("workModeLabel")}
        </p>
        <div role="group" aria-label={t("workModeLabel")} className="flex flex-wrap gap-2">
          {ALL_WORK_MODES.map((mode) => {
            const isSelected = workMode.includes(mode)
            return (
              <button
                key={mode}
                type="button"
                onClick={() => toggleWorkMode(mode)}
                aria-pressed={isSelected}
                className={cn(
                  "inline-flex items-center gap-1.5 rounded-full px-3 py-1.5 text-sm font-medium border transition-all duration-150",
                  "focus-visible:outline-2 focus-visible:outline-ring focus-visible:outline-offset-2",
                  isSelected
                    ? "bg-primary text-primary-foreground border-primary"
                    : "bg-background text-foreground border-border hover:border-primary/60 hover:bg-muted",
                )}
              >
                {isSelected ? (
                  <Check className="w-3.5 h-3.5" aria-hidden="true" />
                ) : null}
                {t(workModeKey(mode))}
              </button>
            )
          })}
        </div>
      </div>

      {needsRadius ? (
        <div className="mt-4 max-w-xs">
          <label
            htmlFor="location-radius"
            className="block text-sm font-medium text-foreground mb-1.5"
          >
            {t("travelRadiusLabel")}
          </label>
          <input
            id="location-radius"
            type="number"
            min={0}
            max={5000}
            value={radius}
            onChange={(e) => setRadius(e.target.value)}
            className="w-full h-10 rounded-lg border border-border bg-background px-3 text-sm shadow-xs focus:border-primary focus:ring-4 focus:ring-primary/10 focus:outline-none"
          />
        </div>
      ) : null}

      <div className="mt-5 flex items-center justify-end">
        <button
          type="button"
          onClick={handleSave}
          disabled={!isDirty || mutation.isPending}
          className="bg-primary text-primary-foreground rounded-md h-9 px-4 text-sm font-medium hover:opacity-90 transition-opacity duration-150 focus-visible:outline-2 focus-visible:outline-ring focus-visible:outline-offset-2 disabled:opacity-50 disabled:cursor-not-allowed inline-flex items-center gap-2"
        >
          {mutation.isPending ? (
            <Loader2 className="w-4 h-4 animate-spin" aria-hidden="true" />
          ) : (
            <Check className="w-4 h-4" aria-hidden="true" />
          )}
          {mutation.isPending ? t("saving") : t("save")}
        </button>
      </div>
    </section>
  )
}

function workModeKey(mode: WorkMode): string {
  switch (mode) {
    case "remote":
      return "workModeRemote"
    case "on_site":
      return "workModeOnSite"
    case "hybrid":
      return "workModeHybrid"
  }
}
