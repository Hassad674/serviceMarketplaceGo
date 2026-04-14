"use client"

import { Link } from "@i18n/navigation"
import { useLocale } from "next-intl"
import {
  AvailabilityPill,
} from "@/shared/components/ui/availability-pill"
import { LocationRow } from "@/shared/components/ui/location-row"
import { LanguagesStrip } from "@/shared/components/ui/languages-strip"
import {
  formatPricing,
  type PricingLocale,
} from "@/shared/lib/profile/pricing-format"
import type { FreelanceProfile } from "../api/freelance-profile-api"

interface FreelanceProfileCardProps {
  profile: FreelanceProfile
  displayName: string
}

// FreelanceProfileCard is the compact directory tile used in the
// /freelancers listing page. Links to the public profile route via
// the canonical organization id. Composes the shared UI primitives
// so the hero strip and the card stay visually in sync.
export function FreelanceProfileCard({
  profile,
  displayName,
}: FreelanceProfileCardProps) {
  const locale = (useLocale() === "fr" ? "fr" : "en") satisfies PricingLocale

  return (
    <article className="bg-card border border-border rounded-2xl p-5 shadow-sm transition-all duration-200 hover:shadow-md hover:border-rose-200 hover:-translate-y-0.5">
      <Link
        href={`/freelancers/${profile.organization_id}`}
        className="flex flex-col gap-4 focus-visible:outline-2 focus-visible:outline-ring focus-visible:outline-offset-2 rounded-2xl"
      >
        <header className="flex items-start gap-3">
          <Avatar photoUrl={profile.photo_url} alt={displayName} />
          <div className="min-w-0 flex-1">
            <h3 className="truncate text-base font-semibold text-foreground">
              {displayName}
            </h3>
            {profile.title ? (
              <p className="truncate text-sm text-muted-foreground">
                {profile.title}
              </p>
            ) : null}
          </div>
          <AvailabilityPill status={profile.availability_status} />
        </header>

        <LocationRow
          city={profile.city}
          countryCode={profile.country_code}
          workMode={profile.work_mode}
        />

        <LanguagesStrip
          professional={profile.languages_professional}
          conversational={profile.languages_conversational}
        />

        {profile.pricing ? (
          <p className="text-sm font-semibold text-foreground">
            {formatPricing(profile.pricing, locale)}
          </p>
        ) : null}
      </Link>
    </article>
  )
}

interface AvatarProps {
  photoUrl: string
  alt: string
}

function Avatar({ photoUrl, alt }: AvatarProps) {
  if (!photoUrl) {
    return (
      <div className="h-12 w-12 rounded-full bg-muted border border-border" />
    )
  }
  return (
    <img
      src={photoUrl}
      alt={alt}
      width={48}
      height={48}
      className="h-12 w-12 rounded-full border border-border object-cover"
    />
  )
}
