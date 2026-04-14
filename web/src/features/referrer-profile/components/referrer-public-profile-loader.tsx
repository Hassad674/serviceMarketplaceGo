"use client"

import { ArrowLeft } from "lucide-react"
import { useTranslations } from "next-intl"
import { useRouter } from "@i18n/navigation"
import { useProfileRating } from "@/shared/hooks/profile/use-profile-rating"
import { usePublicReferrerProfile } from "../hooks/use-referrer-profile"
import { ReferrerPublicProfile } from "./referrer-public-profile"

interface ReferrerPublicProfileLoaderProps {
  orgId: string
}

// ReferrerPublicProfileLoader mirrors the freelance loader and only
// differs by the persona API endpoint + the "referrer" namespace for
// error copy.
export function ReferrerPublicProfileLoader({
  orgId,
}: ReferrerPublicProfileLoaderProps) {
  const t = useTranslations("profile.referrer")
  const router = useRouter()
  const { data: profile, isLoading, error } = usePublicReferrerProfile(orgId)
  const { data: rating } = useProfileRating(orgId)

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

  const displayName = profile.title || profile.organization_id

  return (
    <ReferrerPublicProfile
      profile={profile}
      displayName={displayName}
      rating={
        rating ? { average: rating.average, count: rating.count } : undefined
      }
    />
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
