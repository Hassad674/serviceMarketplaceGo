"use client"

import { Globe, MapPin, Sparkles, Wallet } from "lucide-react"
import { useLocale, useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import type {
  AvailabilityStatus,
  Pricing,
  Profile,
  WorkMode,
} from "../api/profile-api"
import {
  getCountryLabel,
  getFlagEmoji,
} from "../lib/country-options"
import {
  getLanguageFlagCountry,
  getLanguageLabel,
} from "../lib/language-options"
import { formatPricing, type PricingLocale } from "../lib/pricing-format"

const AVAILABILITY_STYLES: Record<AvailabilityStatus, string> = {
  available_now:
    "bg-emerald-50 text-emerald-700 border-emerald-200 dark:bg-emerald-500/10 dark:text-emerald-300 dark:border-emerald-500/30",
  available_soon:
    "bg-amber-50 text-amber-700 border-amber-200 dark:bg-amber-500/10 dark:text-amber-300 dark:border-amber-500/30",
  not_available:
    "bg-rose-50 text-rose-700 border-rose-200 dark:bg-rose-500/10 dark:text-rose-300 dark:border-rose-500/30",
}

const MAX_LANGUAGE_FLAGS = 5

interface ProfileIdentityStripProps {
  profile: Profile
}

// Dense horizontal card shown on the public profile right below the
// header. Four blocks: availability, pricing, location, languages.
// Stacks vertically on mobile; horizontal grid on desktop.
export function ProfileIdentityStrip({ profile }: ProfileIdentityStripProps) {
  const locale = useLocale() === "fr" ? "fr" : "en"
  const hasAvailability = Boolean(profile.availability_status)
  const hasPricing = (profile.pricing?.length ?? 0) > 0
  const hasLocation = Boolean(profile.city || profile.country_code)
  const hasLanguages = (profile.languages_professional?.length ?? 0) > 0

  if (!hasAvailability && !hasPricing && !hasLocation && !hasLanguages) {
    return null
  }

  return (
    <section
      aria-labelledby="profile-identity-strip-title"
      className="bg-card border border-border rounded-xl p-4 sm:p-5 shadow-sm"
    >
      <h2 id="profile-identity-strip-title" className="sr-only">
        Identity
      </h2>
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {hasAvailability ? (
          <AvailabilityBlock
            direct={profile.availability_status as AvailabilityStatus}
            referrer={profile.referrer_availability_status ?? null}
          />
        ) : null}
        {hasPricing ? (
          <PricingBlock rows={profile.pricing ?? []} locale={locale} />
        ) : null}
        {hasLocation ? (
          <LocationBlock
            city={profile.city ?? ""}
            countryCode={profile.country_code ?? ""}
            workMode={profile.work_mode ?? []}
            locale={locale}
          />
        ) : null}
        {hasLanguages ? (
          <LanguagesBlock
            professional={profile.languages_professional ?? []}
            conversational={profile.languages_conversational ?? []}
            locale={locale}
          />
        ) : null}
      </div>
    </section>
  )
}

// ----- Blocks -----------------------------------------------------------

interface AvailabilityBlockProps {
  direct: AvailabilityStatus
  referrer: AvailabilityStatus | null
}

function AvailabilityBlock({ direct, referrer }: AvailabilityBlockProps) {
  const t = useTranslations("profile.availability")
  return (
    <BlockShell icon={<Sparkles className="h-4 w-4" aria-hidden="true" />}>
      <div className="flex flex-col gap-1.5">
        <AvailabilityBadge
          label={referrer ? t("directShort") : ""}
          status={direct}
        />
        {referrer ? (
          <AvailabilityBadge label={t("referrerShort")} status={referrer} />
        ) : null}
      </div>
    </BlockShell>
  )
}

interface AvailabilityBadgeProps {
  label: string
  status: AvailabilityStatus
}

function AvailabilityBadge({ label, status }: AvailabilityBadgeProps) {
  const t = useTranslations("profile.availability")
  return (
    <div className="inline-flex items-center gap-2">
      {label ? (
        <span className="text-[10px] font-medium uppercase tracking-wider text-muted-foreground">
          {label}
        </span>
      ) : null}
      <span
        className={cn(
          "inline-flex items-center gap-1 rounded-full border px-2 py-0.5 text-xs font-medium",
          AVAILABILITY_STYLES[status],
        )}
      >
        <span
          aria-hidden="true"
          className={cn(
            "h-1.5 w-1.5 rounded-full",
            status === "available_now"
              ? "bg-emerald-500"
              : status === "available_soon"
                ? "bg-amber-500"
                : "bg-rose-500",
          )}
        />
        {t(
          status === "available_now"
            ? "statusAvailableNow"
            : status === "available_soon"
              ? "statusAvailableSoon"
              : "statusNotAvailable",
        )}
      </span>
    </div>
  )
}

interface PricingBlockProps {
  rows: Pricing[]
  locale: PricingLocale
}

function PricingBlock({ rows, locale }: PricingBlockProps) {
  const t = useTranslations("profile.pricing")
  return (
    <BlockShell icon={<Wallet className="h-4 w-4" aria-hidden="true" />}>
      <ul className="flex flex-col gap-1">
        {rows.map((row) => (
          <li key={row.kind} className="text-sm">
            <span className="block text-[10px] font-medium uppercase tracking-wider text-muted-foreground">
              {row.kind === "direct" ? t("kindDirect") : t("kindReferral")}
            </span>
            <span className="font-semibold text-foreground">
              {formatPricing(row, locale)}
            </span>
          </li>
        ))}
      </ul>
    </BlockShell>
  )
}

interface LocationBlockProps {
  city: string
  countryCode: string
  workMode: WorkMode[]
  locale: PricingLocale
}

function LocationBlock({
  city,
  countryCode,
  workMode,
  locale,
}: LocationBlockProps) {
  const t = useTranslations("profile.location")
  const flag = countryCode ? getFlagEmoji(countryCode) : ""
  const countryLabel = countryCode ? getCountryLabel(countryCode, locale) : ""
  return (
    <BlockShell icon={<MapPin className="h-4 w-4" aria-hidden="true" />}>
      <div className="flex flex-col gap-1">
        <p className="text-sm font-semibold text-foreground">
          {flag ? <span className="mr-1.5">{flag}</span> : null}
          {[city, countryLabel].filter(Boolean).join(", ")}
        </p>
        {workMode.length > 0 ? (
          <p className="flex flex-wrap gap-1">
            {workMode.map((mode) => (
              <span
                key={mode}
                className="inline-flex items-center rounded-full bg-muted px-2 py-0.5 text-[10px] font-medium text-muted-foreground border border-border"
              >
                {t(
                  mode === "remote"
                    ? "workModeRemote"
                    : mode === "on_site"
                      ? "workModeOnSite"
                      : "workModeHybrid",
                )}
              </span>
            ))}
          </p>
        ) : null}
      </div>
    </BlockShell>
  )
}

interface LanguagesBlockProps {
  professional: string[]
  conversational: string[]
  locale: PricingLocale
}

function LanguagesBlock({
  professional,
  conversational,
  locale,
}: LanguagesBlockProps) {
  const t = useTranslations("profile.languages")
  const visible = professional.slice(0, MAX_LANGUAGE_FLAGS)
  const overflow = professional.length - visible.length
  return (
    <BlockShell icon={<Globe className="h-4 w-4" aria-hidden="true" />}>
      <div className="flex flex-col gap-1">
        <div className="flex flex-wrap items-center gap-1.5">
          {visible.map((code) => {
            const flag = getFlagEmoji(getLanguageFlagCountry(code))
            return (
              <span
                key={code}
                title={getLanguageLabel(code, locale)}
                className="inline-flex items-center gap-1 rounded-full bg-muted px-2 py-0.5 text-xs font-medium text-foreground border border-border"
              >
                {flag ? <span aria-hidden="true">{flag}</span> : null}
                {code.toUpperCase()}
              </span>
            )
          })}
          {overflow > 0 ? (
            <span className="inline-flex items-center rounded-full bg-muted px-2 py-0.5 text-xs font-medium text-muted-foreground border border-border">
              +{overflow}
            </span>
          ) : null}
        </div>
        {conversational.length > 0 ? (
          <span className="text-[11px] text-muted-foreground">
            {t("conversationalShort", { count: conversational.length })}
          </span>
        ) : null}
      </div>
    </BlockShell>
  )
}

interface BlockShellProps {
  icon: React.ReactNode
  children: React.ReactNode
}

function BlockShell({ icon, children }: BlockShellProps) {
  return (
    <div className="flex items-start gap-3 rounded-lg border border-border bg-muted/20 px-3 py-2.5">
      <span className="mt-0.5 inline-flex h-8 w-8 items-center justify-center rounded-full bg-primary/10 text-primary">
        {icon}
      </span>
      <div className="min-w-0 flex-1">{children}</div>
    </div>
  )
}
