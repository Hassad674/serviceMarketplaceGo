"use client"

import { useCallback, useState } from "react"
import { Check, Loader2 } from "lucide-react"
import { useLocale, useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import {
  COUNTRY_OPTIONS,
  getCountryLabel,
} from "@/shared/lib/profile/country-options"
import {
  CityAutocomplete,
  type CitySelection,
} from "@/shared/components/location/city-autocomplete"
import { useOrganizationShared } from "../hooks/use-organization-shared"
import { useUpdateOrganizationLocation } from "../hooks/use-update-organization-location"
import type { WorkMode } from "../api/organization-shared-api"

import { Button } from "@/shared/components/ui/button"
import { Input } from "@/shared/components/ui/input"
import { Select } from "@/shared/components/ui/select"
const ALL_WORK_MODES: WorkMode[] = ["remote", "on_site", "hybrid"]

// SharedLocationSection renders the editable "Where you work" card on
// /profile. Writes go to the organizations row via
// /api/v1/organization/location, so both the freelance and the
// referrer persona read the canonical value through their own
// profile endpoint after the mutation settles (the hook invalidates
// both persona caches for us).
export function SharedLocationSection() {
  const t = useTranslations("profile.location")
  const locale = useLocale() === "fr" ? "fr" : "en"
  const { data: shared } = useOrganizationShared()
  const mutation = useUpdateOrganizationLocation()

  const persisted = buildPersistedSnapshot(shared)
  const draft = useLocationDraft(persisted)

  const handleSave = useCallback(() => {
    const radiusValue = draft.radius.trim() === "" ? null : Number(draft.radius)
    const sanitized =
      radiusValue !== null && Number.isFinite(radiusValue) && radiusValue >= 0
        ? Math.round(radiusValue)
        : null
    mutation.mutate({
      city: draft.selection?.city ?? "",
      country_code: draft.selection?.countryCode ?? draft.country,
      latitude: draft.selection?.latitude ?? null,
      longitude: draft.selection?.longitude ?? null,
      work_mode: draft.workMode,
      travel_radius_km: sanitized,
    })
  }, [draft, mutation])

  const needsRadius =
    draft.workMode.includes("on_site") || draft.workMode.includes("hybrid")

  return (
    <section
      aria-labelledby="shared-location-section-title"
      className="bg-card border border-border rounded-xl p-6 shadow-sm"
    >
      <header className="mb-4 flex flex-col gap-1">
        <h2
          id="shared-location-section-title"
          className="text-lg font-semibold text-foreground"
        >
          {t("sectionTitle")}
        </h2>
        <p className="text-sm text-muted-foreground">{t("sectionSubtitle")}</p>
      </header>

      <div className="grid gap-4 sm:grid-cols-2">
        <div>
          <label
            htmlFor="shared-location-city"
            className="block text-sm font-medium text-foreground mb-1.5"
          >
            {t("cityLabel")}
          </label>
          <div id="shared-location-city">
            <CityAutocomplete
              value={draft.selection}
              countryCode={draft.country}
              onChange={draft.setSelection}
            />
          </div>
        </div>
        <CountryField
          value={draft.country}
          locale={locale}
          onChange={(next) => draft.setCountry(next)}
        />
      </div>

      <WorkModeField
        selected={draft.workMode}
        onToggle={draft.toggleWorkMode}
      />

      {needsRadius ? (
        <RadiusField
          value={draft.radius}
          onChange={(next) => draft.setRadius(next)}
        />
      ) : null}

      <SaveRow
        isDirty={draft.isDirty}
        isSaving={mutation.isPending}
        onSave={handleSave}
      />
    </section>
  )
}

// ----- Field sub-components keep the main function under 50 lines ----

interface CountryFieldProps {
  value: string
  locale: "fr" | "en"
  onChange: (next: string) => void
}

function CountryField({ value, locale, onChange }: CountryFieldProps) {
  const t = useTranslations("profile.location")
  return (
    <div>
      <label
        htmlFor="shared-location-country"
        className="block text-sm font-medium text-foreground mb-1.5"
      >
        {t("countryLabel")}
      </label>
      <Select
        id="shared-location-country"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="w-full h-10 rounded-lg border border-border bg-background px-3 text-sm shadow-xs focus:border-primary focus:ring-4 focus:ring-primary/10 focus:outline-none"
      >
        <option value="">{t("countryPlaceholder")}</option>
        {COUNTRY_OPTIONS.map((option) => (
          <option key={option.code} value={option.code}>
            {getCountryLabel(option.code, locale)}
          </option>
        ))}
      </Select>
    </div>
  )
}

interface WorkModeFieldProps {
  selected: WorkMode[]
  onToggle: (mode: WorkMode) => void
}

function WorkModeField({ selected, onToggle }: WorkModeFieldProps) {
  const t = useTranslations("profile.location")
  return (
    <div className="mt-4">
      <p className="block text-sm font-medium text-foreground mb-2">
        {t("workModeLabel")}
      </p>
      <div
        role="group"
        aria-label={t("workModeLabel")}
        className="flex flex-wrap gap-2"
      >
        {ALL_WORK_MODES.map((mode) => {
          const isSelected = selected.includes(mode)
          return (
            <Button variant="ghost" size="auto"
              key={mode}
              type="button"
              onClick={() => onToggle(mode)}
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
              {t(workModeLabelKey(mode))}
            </Button>
          )
        })}
      </div>
    </div>
  )
}

interface RadiusFieldProps {
  value: string
  onChange: (next: string) => void
}

function RadiusField({ value, onChange }: RadiusFieldProps) {
  const t = useTranslations("profile.location")
  return (
    <div className="mt-4 max-w-xs">
      <label
        htmlFor="shared-location-radius"
        className="block text-sm font-medium text-foreground mb-1.5"
      >
        {t("travelRadiusLabel")}
      </label>
      <Input
        id="shared-location-radius"
        type="number"
        min={0}
        max={5000}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="w-full h-10 rounded-lg border border-border bg-background px-3 text-sm shadow-xs focus:border-primary focus:ring-4 focus:ring-primary/10 focus:outline-none"
      />
    </div>
  )
}

interface SaveRowProps {
  isDirty: boolean
  isSaving: boolean
  onSave: () => void
}

function SaveRow({ isDirty, isSaving, onSave }: SaveRowProps) {
  const t = useTranslations("profile.location")
  return (
    <div className="mt-5 flex items-center justify-end">
      <Button variant="ghost" size="auto"
        type="button"
        onClick={onSave}
        disabled={!isDirty || isSaving}
        className="bg-primary text-primary-foreground rounded-md h-9 px-4 text-sm font-medium hover:opacity-90 transition-opacity duration-150 focus-visible:outline-2 focus-visible:outline-ring focus-visible:outline-offset-2 disabled:opacity-50 disabled:cursor-not-allowed inline-flex items-center gap-2"
      >
        {isSaving ? (
          <Loader2 className="w-4 h-4 animate-spin" aria-hidden="true" />
        ) : (
          <Check className="w-4 h-4" aria-hidden="true" />
        )}
        {isSaving ? t("saving") : t("save")}
      </Button>
    </div>
  )
}

// ----- Draft hook: mirrors the react-recommended derive-during-render pattern

interface PersistedLocation {
  selection: CitySelection | null
  country: string
  workMode: WorkMode[]
  radius: string
}

type SharedLocationFields = {
  city?: string
  country_code?: string
  latitude?: number | null
  longitude?: number | null
  work_mode?: WorkMode[]
  travel_radius_km?: number | null
}

function buildPersistedSnapshot(
  shared: SharedLocationFields | undefined,
): PersistedLocation {
  const hasCoords =
    shared?.latitude !== null &&
    shared?.latitude !== undefined &&
    shared?.longitude !== null &&
    shared?.longitude !== undefined
  return {
    selection:
      shared?.city && shared?.country_code && hasCoords
        ? {
            city: shared.city,
            countryCode: shared.country_code,
            latitude: shared.latitude as number,
            longitude: shared.longitude as number,
          }
        : null,
    country: shared?.country_code ?? "",
    workMode: shared?.work_mode ?? [],
    radius:
      shared?.travel_radius_km === null || shared?.travel_radius_km === undefined
        ? ""
        : String(shared.travel_radius_km),
  }
}

interface LocationDraft extends PersistedLocation {
  setSelection: (next: CitySelection | null) => void
  setCountry: (next: string) => void
  setRadius: (next: string) => void
  toggleWorkMode: (mode: WorkMode) => void
  isDirty: boolean
}

function useLocationDraft(persisted: PersistedLocation): LocationDraft {
  const persistedKey = `${persisted.selection?.city ?? ""}|${persisted.selection?.countryCode ?? ""}|${persisted.selection?.latitude ?? ""}|${persisted.selection?.longitude ?? ""}|${persisted.country}|${persisted.workMode.join(",")}|${persisted.radius}`
  const [prevKey, setPrevKey] = useState(persistedKey)
  const [selection, setSelection] = useState<CitySelection | null>(persisted.selection)
  const [country, setCountry] = useState(persisted.country)
  const [workMode, setWorkMode] = useState<WorkMode[]>(persisted.workMode)
  const [radius, setRadius] = useState(persisted.radius)

  if (prevKey !== persistedKey) {
    setPrevKey(persistedKey)
    setSelection(persisted.selection)
    setCountry(persisted.country)
    setWorkMode(persisted.workMode)
    setRadius(persisted.radius)
  }

  const toggleWorkMode = (mode: WorkMode) => {
    setWorkMode((current) =>
      current.includes(mode)
        ? current.filter((m) => m !== mode)
        : [...current, mode],
    )
  }

  // Changing the country wipes the city selection so the user is
  // forced to re-pick a canonical municipality in the new country —
  // keeps lat/lng consistent with the ISO code we persist.
  const setCountryAndClearCity = (next: string) => {
    setCountry(next)
    if (selection && selection.countryCode !== next) {
      setSelection(null)
    }
  }

  const selectionKey = `${selection?.city ?? ""}|${selection?.countryCode ?? ""}|${selection?.latitude ?? ""}|${selection?.longitude ?? ""}`
  const persistedSelectionKey = `${persisted.selection?.city ?? ""}|${persisted.selection?.countryCode ?? ""}|${persisted.selection?.latitude ?? ""}|${persisted.selection?.longitude ?? ""}`

  const isDirty =
    selectionKey !== persistedSelectionKey ||
    country !== persisted.country ||
    workMode.join(",") !== persisted.workMode.join(",") ||
    radius !== persisted.radius

  return {
    selection,
    country,
    workMode,
    radius,
    setSelection,
    setCountry: setCountryAndClearCity,
    setRadius,
    toggleWorkMode,
    isDirty,
  }
}

function workModeLabelKey(mode: WorkMode): string {
  switch (mode) {
    case "remote":
      return "workModeRemote"
    case "on_site":
      return "workModeOnSite"
    case "hybrid":
      return "workModeHybrid"
  }
}
