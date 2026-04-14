"use client"

import { MapPin, Star } from "lucide-react"
import { useLocale, useTranslations } from "next-intl"
import { Link } from "@i18n/navigation"
import { cn } from "@/shared/lib/utils"
import type { PublicProfileSummary, SearchType } from "../api/search-api"
import { SkillsDisplay } from "./skills-display"
import { getFlagEmoji } from "../lib/country-options"
import { formatPricing, type PricingLocale } from "../lib/pricing-format"

// Keeps the card compact — the directory grid must stay scannable even
// for providers with many skills. Anything beyond this count collapses
// into a "+N" overflow chip rendered by SkillsDisplay.
const CARD_MAX_VISIBLE_SKILLS = 4

// Badge styling is keyed on search type (the "what directory am I in"
// dimension), not the raw org type, because provider_personal orgs
// surface in both the freelancer and referrer directories with
// different labels.
const BADGE_STYLES: Record<SearchType, string> = {
  agency: "bg-blue-50 text-blue-700 dark:bg-blue-500/20 dark:text-blue-400",
  enterprise: "bg-purple-50 text-purple-700 dark:bg-purple-500/20 dark:text-purple-400",
  freelancer: "bg-rose-50 text-rose-700 dark:bg-rose-500/20 dark:text-rose-400",
  referrer: "bg-amber-50 text-amber-700 dark:bg-amber-500/20 dark:text-amber-400",
}

const BADGE_LABELS: Record<SearchType, string> = {
  agency: "Agency",
  enterprise: "Enterprise",
  freelancer: "Freelancer",
  referrer: "Referrer",
}

function getProfileHref(profile: PublicProfileSummary, type: SearchType): string {
  switch (type) {
    case "agency":
      return `/agencies/${profile.organization_id}`
    case "enterprise":
      return `/enterprises/${profile.organization_id}`
    case "referrer":
      return `/referrers/${profile.organization_id}`
    case "freelancer":
    default:
      return `/freelancers/${profile.organization_id}`
  }
}

function getInitials(profile: PublicProfileSummary): string {
  const trimmed = profile.name.trim()
  if (!trimmed) return "?"
  const parts = trimmed.split(/\s+/)
  if (parts.length >= 2) {
    return `${parts[0].charAt(0)}${parts[1].charAt(0)}`.toUpperCase()
  }
  return trimmed.charAt(0).toUpperCase()
}

interface ProviderCardProps {
  profile: PublicProfileSummary
  type: SearchType
}

export function ProviderCard({ profile, type }: ProviderCardProps) {
  const t = useTranslations("search")
  const locale: PricingLocale = useLocale() === "fr" ? "fr" : "en"
  const badgeStyle = BADGE_STYLES[type] ?? BADGE_STYLES.freelancer
  const badgeLabel = BADGE_LABELS[type] ?? type
  const primaryPricing = pickPrimaryPricing(profile)
  const primaryPricingLabel = primaryPricing
    ? formatPricing(primaryPricing, locale)
    : ""

  return (
    <Link
      href={getProfileHref(profile, type)}
      className={cn(
        "group block rounded-xl border border-gray-100 dark:border-gray-800",
        "bg-white dark:bg-gray-900 p-4 shadow-sm",
        "transition-all duration-200 hover:shadow-md hover:-translate-y-0.5",
      )}
    >
      <div className="flex items-start gap-4">
        {/* Avatar */}
        <div className="shrink-0">
          {profile.photo_url ? (
            <img
              src={profile.photo_url}
              alt={profile.name}
              width={48}
              height={48}
              className="h-12 w-12 rounded-full object-cover"
            />
          ) : (
            <div className="flex h-12 w-12 items-center justify-center rounded-full bg-gradient-to-br from-rose-500 to-purple-600 text-sm font-semibold text-white">
              {getInitials(profile)}
            </div>
          )}
        </div>

        {/* Info */}
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2 flex-wrap">
            <h3 className="truncate text-sm font-semibold text-gray-900 dark:text-white group-hover:text-rose-600 dark:group-hover:text-rose-400 transition-colors">
              {profile.name || t("noTitle")}
            </h3>
            <span
              className={cn(
                "inline-block rounded-md px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-wider",
                badgeStyle,
              )}
            >
              {badgeLabel}
            </span>
          </div>
          <p className="mt-0.5 truncate text-sm text-gray-500 dark:text-gray-400">
            {profile.title || t("noTitle")}
          </p>
          {profile.review_count > 0 && (
            <div className="mt-1.5 flex items-center gap-1 text-xs">
              <Star
                className="h-3 w-3 fill-amber-400 text-amber-400"
                strokeWidth={1.5}
                aria-hidden="true"
              />
              <span className="font-semibold text-gray-900 dark:text-white">
                {profile.average_rating.toFixed(1)}
              </span>
              <span className="text-gray-500 dark:text-gray-400">
                ({profile.review_count})
              </span>
            </div>
          )}
          <ProviderCardSignals
            profile={profile}
            primaryPricingLabel={primaryPricingLabel}
          />
          {profile.skills && profile.skills.length > 0 && (
            <SkillsDisplay
              skills={profile.skills}
              maxVisible={CARD_MAX_VISIBLE_SKILLS}
              className="mt-2"
            />
          )}
        </div>
      </div>
    </Link>
  )
}

// ----- Inline signals row ------------------------------------------------

const AVAILABILITY_DOT_STYLES = {
  available_now: "bg-emerald-500",
  available_soon: "bg-amber-500",
  not_available: "bg-rose-500",
} as const

interface ProviderCardSignalsProps {
  profile: PublicProfileSummary
  primaryPricingLabel: string
}

function ProviderCardSignals({
  profile,
  primaryPricingLabel,
}: ProviderCardSignalsProps) {
  const languages = profile.languages_professional ?? []
  const hasLocation = Boolean(profile.city || profile.country_code)
  const hasAvailability = Boolean(profile.availability_status)
  const hasLanguages = languages.length > 0
  const hasPricing = primaryPricingLabel !== ""

  if (!hasLocation && !hasAvailability && !hasLanguages && !hasPricing) {
    return null
  }

  return (
    <div className="mt-1.5 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-gray-500 dark:text-gray-400">
      {hasLocation ? (
        <span className="inline-flex items-center gap-1 truncate max-w-[160px]">
          <MapPin className="h-3 w-3" aria-hidden="true" strokeWidth={1.5} />
          {profile.country_code ? (
            <span aria-hidden="true">{getFlagEmoji(profile.country_code)}</span>
          ) : null}
          <span className="truncate">{profile.city ?? profile.country_code}</span>
        </span>
      ) : null}
      {hasAvailability ? (
        <span className="inline-flex items-center gap-1">
          <span
            aria-hidden="true"
            className={cn(
              "h-1.5 w-1.5 rounded-full",
              AVAILABILITY_DOT_STYLES[profile.availability_status ?? "not_available"],
            )}
          />
        </span>
      ) : null}
      {hasLanguages ? (
        <span className="inline-flex items-center gap-0.5 uppercase font-medium">
          {languages.slice(0, 3).map((code, index) => (
            <span key={code}>
              {index > 0 ? <span aria-hidden="true"> · </span> : null}
              {code}
            </span>
          ))}
        </span>
      ) : null}
      {hasPricing ? (
        <span className="inline-flex items-center font-semibold text-gray-900 dark:text-white">
          {primaryPricingLabel}
        </span>
      ) : null}
    </div>
  )
}

function pickPrimaryPricing(profile: PublicProfileSummary) {
  const rows = profile.pricing ?? []
  if (rows.length === 0) return null
  return rows.find((row) => row.kind === "direct") ?? rows[0]
}
