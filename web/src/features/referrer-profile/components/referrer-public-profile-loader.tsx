"use client"

import { ArrowLeft } from "lucide-react"
import { useTranslations } from "next-intl"
import { useRouter } from "@i18n/navigation"
import { useReferrerReputation } from "../hooks/use-referrer-reputation"
import { usePublicReferrerProfile } from "../hooks/use-referrer-profile"
import { ReferrerPublicProfile } from "./referrer-public-profile"
import { PublicReferrerSocialLinks } from "./referrer-social-links-section"

interface ReferrerPublicProfileLoaderProps {
  orgId: string
}

// ReferrerPublicProfileLoader wires the referrer profile read path to
// the dedicated apporteur reputation aggregate (not the freelance
// rating) — the apporteur profile has its OWN rating, computed from
// client reviews on the providers introduced through this user's
// referrals. Using useProfileRating here would leak the user's
// freelance rating into the apporteur profile, which is the exact
// bug this feature fixes.
export function ReferrerPublicProfileLoader({
  orgId,
}: ReferrerPublicProfileLoaderProps) {
  const t = useTranslations("profile.referrer")
  const router = useRouter()
  const { data: profile, isLoading, error } = usePublicReferrerProfile(orgId)
  const reputation = useReferrerReputation(orgId)
  const firstPage = reputation.data?.pages[0]

  if (isLoading) return <LoadingShell />

  if (error || !profile) {
    return (
      <div className="rounded-xl border border-destructive/30 bg-destructive/5 p-8 text-center">
        <p className="text-sm font-semibold text-destructive">
          {t("notFoundTitle")}
        </p>
        <p className="mt-1 text-xs text-destructive/80">
          {t("notFoundDescription")}
        </p>
        <button
          onClick={() => router.back()}
          className="mt-3 inline-flex items-center gap-1.5 text-sm text-primary hover:opacity-80 transition-opacity"
        >
          <ArrowLeft className="h-4 w-4" />
          {t("loadError")}
        </button>
      </div>
    )
  }

  // displayName falls back to a localized "Apporteur d'affaires" label
  // when the referrer has not yet set a title — surfacing the raw
  // organization id (a UUID) to a public viewer is both ugly and a
  // privacy leak. The `t("publicTitleSuffix")` already covers the
  // metadata path; this mirror in the header keeps the two surfaces
  // consistent.
  const displayName = profile.title || t("displayNameFallback")
  // The header rating uses the apporteur reputation, not the freelance
  // rating. Undefined until the reputation query settles so the
  // ProfileIdentityHeader hides the rating block during loading.
  const headerRating = firstPage
    ? { average: firstPage.rating_avg, count: firstPage.review_count }
    : undefined

  return (
    <div className="space-y-6">
      <ReferrerPublicProfile
        profile={profile}
        displayName={displayName}
        rating={headerRating}
      />
      <PublicReferrerSocialLinks orgId={orgId} />
    </div>
  )
}

function LoadingShell() {
  return (
    <div className="space-y-6" role="status" aria-live="polite">
      <div className="h-32 rounded-xl border border-border bg-muted/40 animate-shimmer" />
      <div className="h-40 rounded-xl border border-border bg-muted/40 animate-shimmer" />
      <div className="h-64 rounded-xl border border-border bg-muted/40 animate-shimmer" />
    </div>
  )
}
