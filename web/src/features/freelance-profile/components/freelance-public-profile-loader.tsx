"use client"

import { ArrowLeft } from "lucide-react"
import { useTranslations } from "next-intl"
import { useRouter } from "@i18n/navigation"
import { useProfileRating } from "@/shared/hooks/profile/use-profile-rating"
import { usePublicFreelanceProfile } from "../hooks/use-freelance-profile"
import { FreelancePublicProfile } from "./freelance-public-profile"

interface FreelancePublicProfileLoaderProps {
  orgId: string
}

// FreelancePublicProfileLoader is the client-side composition of the
// public viewer: data fetching + the read-only profile card. Kept
// separate from FreelancePublicProfile so that the same component
// can be reused under /profile with an already-loaded payload
// (owner-side) and under /freelancers/[id] with a network fetch
// (public viewer side).
export function FreelancePublicProfileLoader({
  orgId,
}: FreelancePublicProfileLoaderProps) {
  const t = useTranslations("profile.freelance")
  const router = useRouter()
  const { data: profile, isLoading, error } = usePublicFreelanceProfile(orgId)
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
    <FreelancePublicProfile
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
