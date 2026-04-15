"use client"

import Image from "next/image"
import { Star, MapPin } from "lucide-react"
import { useLocale, useTranslations } from "next-intl"
import { Link } from "@i18n/navigation"
import { cn } from "@/shared/lib/utils"
import { AvailabilityPill } from "@/shared/components/ui/availability-pill"
import { formatPricing } from "@/shared/lib/profile/pricing-format"
import { getFlagEmoji } from "@/shared/lib/profile/country-options"
import {
  currencyForPricing,
  formatTotalEarned,
  type FormatLocale,
} from "@/shared/lib/search/format-total-earned"
import type {
  SearchDocument,
  SearchDocumentPersona,
} from "@/shared/lib/search/search-document"

// SearchResultCard is the Malt-inspired card used across the three
// public listing pages (freelancers / agencies / referrers). It reads
// only from the frozen SearchDocument contract so the Typesense swap
// is a one-file adapter change; the card itself never touches the API.
//
// Visual hierarchy (top to bottom):
//
//   1. Photo cover (4:5 aspect). Availability pill top-left.
//      Rating badge top-right (hidden until at least one review).
//   2. Display name + title.
//   3. Metadata row: city + flag icons for professional languages.
//   4. Total-earned line (Upwork-inspired, hidden at zero).
//   5. Pricing line with optional negotiable pill (hidden when null).
//   6. Skills chips (top three + overflow).
//
// Extracted sub-components keep the card body under 50 lines and JSX
// nesting at 3 levels — the project's quality bar.

interface SearchResultCardProps {
  document: SearchDocument
}

const PERSONA_TO_PATH: Record<SearchDocumentPersona, string> = {
  freelance: "/freelancers",
  agency: "/agencies",
  referrer: "/referrers",
}

export function SearchResultCard({ document }: SearchResultCardProps) {
  const locale: FormatLocale = useLocale() === "fr" ? "fr" : "en"
  const t = useTranslations("search")
  const headingId = `search-card-${document.id}`
  const href = `${PERSONA_TO_PATH[document.persona]}/${document.id}`

  return (
    <article aria-labelledby={headingId} className="h-full">
      <Link
        href={href}
        className={cn(
          "group flex h-full flex-col overflow-hidden rounded-2xl border border-border bg-card",
          "shadow-sm transition-all duration-200 ease-out",
          "hover:-translate-y-0.5 hover:border-rose-200 hover:shadow-md",
          "focus-visible:outline-none focus-visible:ring-4 focus-visible:ring-rose-500/20",
        )}
      >
        <PhotoCover document={document} />
        <div className="flex flex-1 flex-col gap-2 p-4">
          <HeaderBlock document={document} headingId={headingId} />
          <MetadataRow document={document} />
          <TotalEarnedLine
            amount={document.total_earned}
            currency={currencyForPricing(document.pricing)}
            locale={locale}
          />
          <PricingLine document={document} locale={locale} />
          <SkillChips skills={document.skills} />
        </div>
      </Link>
    </article>
  )
}

// ---------------------------------------------------------------------------
// Sub-components
// ---------------------------------------------------------------------------

function PhotoCover({ document }: { document: SearchDocument }) {
  const hasPhoto = document.photo_url.trim().length > 0
  return (
    <div className="relative aspect-[4/5] w-full overflow-hidden bg-muted">
      {hasPhoto ? (
        <Image
          src={document.photo_url}
          alt={document.display_name || "Profile photo"}
          fill
          sizes="(max-width: 768px) 100vw, (max-width: 1024px) 50vw, 33vw"
          className="object-cover transition-transform duration-300 group-hover:scale-[1.02]"
        />
      ) : (
        <InitialsBackdrop name={document.display_name} />
      )}
      <div className="absolute left-3 top-3">
        <AvailabilityPill status={document.availability_status} />
      </div>
      {document.rating.count > 0 ? (
        <div className="absolute right-3 top-3 flex items-center gap-1 rounded-full bg-black/70 px-2 py-0.5 text-xs font-medium text-white backdrop-blur-sm">
          <Star className="h-3 w-3 fill-amber-300 text-amber-300" aria-hidden />
          <span>{document.rating.average.toFixed(1)}</span>
        </div>
      ) : null}
    </div>
  )
}

function InitialsBackdrop({ name }: { name: string }) {
  const initials = getInitials(name)
  return (
    <div className="flex h-full w-full items-center justify-center bg-gradient-to-br from-rose-100 to-rose-50 text-4xl font-semibold text-rose-500 dark:from-rose-500/20 dark:to-slate-900">
      {initials}
    </div>
  )
}

function HeaderBlock({
  document,
  headingId,
}: {
  document: SearchDocument
  headingId: string
}) {
  const t = useTranslations("search")
  return (
    <div className="flex flex-col gap-0.5">
      <h3
        id={headingId}
        className="truncate text-base font-semibold text-foreground transition-colors group-hover:text-rose-600 dark:group-hover:text-rose-400"
      >
        {document.display_name || t("noTitle")}
      </h3>
      <p className="truncate text-sm text-muted-foreground">
        {document.title || t("noTitle")}
      </p>
    </div>
  )
}

function MetadataRow({ document }: { document: SearchDocument }) {
  const hasLocation =
    document.city.length > 0 || document.country_code.length > 0
  const languages = document.languages_professional.slice(0, 3)
  if (!hasLocation && languages.length === 0) return null

  return (
    <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-muted-foreground">
      {hasLocation ? (
        <span className="inline-flex items-center gap-1 truncate">
          <MapPin className="h-3 w-3" aria-hidden strokeWidth={1.75} />
          {document.country_code ? (
            <span aria-hidden>{getFlagEmoji(document.country_code)}</span>
          ) : null}
          <span className="truncate">
            {document.city || document.country_code}
          </span>
        </span>
      ) : null}
      {languages.length > 0 ? (
        <span className="inline-flex items-center gap-1" aria-label="languages">
          {languages.map((code) => (
            <span
              key={code}
              className="rounded-sm bg-muted px-1 py-0.5 text-[10px] font-semibold uppercase tracking-wide"
            >
              {code}
            </span>
          ))}
        </span>
      ) : null}
    </div>
  )
}

interface TotalEarnedLineProps {
  amount: number
  currency: string
  locale: FormatLocale
}

function TotalEarnedLine({
  amount,
  currency,
  locale,
}: TotalEarnedLineProps) {
  const t = useTranslations("search")
  const formatted = formatTotalEarned(amount, currency, locale)
  if (!formatted) return null
  return (
    <p className="text-[13px] font-semibold text-rose-600 dark:text-rose-400">
      {t("totalEarned", { amount: formatted })}
    </p>
  )
}

function PricingLine({
  document,
  locale,
}: {
  document: SearchDocument
  locale: FormatLocale
}) {
  const t = useTranslations("search")
  const pricing = document.pricing
  if (!pricing) return null
  const formatted = formatPricing(
    {
      type: pricing.type,
      min_amount: pricing.min_amount,
      max_amount: pricing.max_amount,
      currency: pricing.currency,
    },
    locale,
  )
  return (
    <div className="flex flex-wrap items-center gap-2">
      <span className="text-sm font-semibold text-foreground">{formatted}</span>
      {pricing.negotiable ? (
        <span className="inline-flex items-center rounded-full bg-rose-100 px-2 py-0.5 text-[11px] font-medium text-rose-700 dark:bg-rose-500/15 dark:text-rose-300">
          {t("negotiable")}
        </span>
      ) : null}
    </div>
  )
}

function SkillChips({ skills }: { skills: string[] }) {
  const t = useTranslations("search")
  if (skills.length === 0) return null
  const visible = skills.slice(0, 3)
  const overflow = skills.length - visible.length
  return (
    <ul className="mt-auto flex flex-wrap gap-1.5 pt-1" aria-label="skills">
      {visible.map((skill) => (
        <li key={skill}>
          <span className="inline-flex items-center rounded-full border border-border bg-muted/50 px-2 py-0.5 text-[11px] font-medium text-foreground">
            {skill}
          </span>
        </li>
      ))}
      {overflow > 0 ? (
        <li>
          <span className="inline-flex items-center rounded-full bg-muted px-2 py-0.5 text-[11px] font-medium text-muted-foreground">
            {t("moreSkills", { count: overflow })}
          </span>
        </li>
      ) : null}
    </ul>
  )
}

function getInitials(name: string): string {
  const trimmed = name.trim()
  if (!trimmed) return "?"
  const parts = trimmed.split(/\s+/)
  if (parts.length >= 2) {
    return `${parts[0].charAt(0)}${parts[1].charAt(0)}`.toUpperCase()
  }
  return trimmed.charAt(0).toUpperCase()
}
