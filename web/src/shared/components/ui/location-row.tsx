"use client"

import { MapPin } from "lucide-react"
import { useLocale, useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import {
  getCountryLabel,
  getFlagEmoji,
} from "@/shared/lib/profile/country-options"

export type WorkMode = "remote" | "on_site" | "hybrid"

interface LocationRowProps {
  city: string
  countryCode: string
  workMode: WorkMode[]
  className?: string
}

// LocationRow is the inline "MapPin + flag + city, country + work mode
// badges" block rendered on profile headers and public profile strips.
// No routing, no data fetching — purely reads from props so each
// persona feature can compose it with its own data source.
export function LocationRow({
  city,
  countryCode,
  workMode,
  className,
}: LocationRowProps) {
  const locale = useLocale() === "fr" ? "fr" : "en"
  const flag = countryCode ? getFlagEmoji(countryCode) : ""
  const countryLabel = countryCode ? getCountryLabel(countryCode, locale) : ""
  const label = [city, countryLabel].filter(Boolean).join(", ")

  if (!label && workMode.length === 0) return null

  return (
    <div className={cn("flex flex-col gap-1.5", className)}>
      {label ? (
        <p className="flex items-center gap-2 text-sm font-semibold text-foreground">
          <MapPin className="h-4 w-4 text-muted-foreground" aria-hidden="true" />
          {flag ? <span aria-hidden="true">{flag}</span> : null}
          <span>{label}</span>
        </p>
      ) : null}
      {workMode.length > 0 ? <WorkModeBadges modes={workMode} /> : null}
    </div>
  )
}

interface WorkModeBadgesProps {
  modes: WorkMode[]
}

function WorkModeBadges({ modes }: WorkModeBadgesProps) {
  const t = useTranslations("profile.location")
  return (
    <ul className="flex flex-wrap gap-1">
      {modes.map((mode) => (
        <li key={mode}>
          <span className="inline-flex items-center rounded-full border border-border bg-muted px-2 py-0.5 text-[10px] font-medium text-muted-foreground">
            {t(workModeLabelKey(mode))}
          </span>
        </li>
      ))}
    </ul>
  )
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
