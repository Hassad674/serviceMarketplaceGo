"use client"

import { ArrowLeft } from "lucide-react"
import { useTranslations } from "next-intl"
import { useRouter } from "@i18n/navigation"
import { useProfileRating } from "@/shared/hooks/profile/use-profile-rating"
import { usePublicFreelanceProfile } from "../hooks/use-freelance-profile"
import { FreelancePublicProfile } from "./freelance-public-profile"
import { PublicFreelanceSocialLinks } from "./freelance-social-links-section"
import { ProjectHistorySection } from "@/shared/components/profile/project-history-section"
import { Button } from "@/shared/components/ui/button"

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
        <Button variant="ghost" size="auto"
          onClick={() => router.back()}
          className="mt-3 inline-flex items-center gap-1.5 text-sm text-primary hover:opacity-80 transition-opacity"
        >
          <ArrowLeft className="h-4 w-4" />
          {t("loadError")}
        </Button>
      </div>
    )
  }

  const displayName = profile.title || profile.organization_id

  // Soleil v2 W-16 v3 — history pinned LAST (brief #2). The aerated
  // max-w-4xl wrapper (~896px) is set on the inner profile component
  // to keep the editorial column tight; the trailing social links and
  // project history adopt the same wrapper here for visual cohesion.
  return (
    <div className="mx-auto w-full max-w-4xl space-y-6">
      <FreelancePublicProfile
        profile={profile}
        displayName={displayName}
        rating={
          rating ? { average: rating.average, count: rating.count } : undefined
        }
      />
      <PublicFreelanceSocialLinks orgId={orgId} />
      <ProjectHistorySection orgId={orgId} readOnly />
    </div>
  )
}

function LoadingShell() {
  return (
    <div
      className="mx-auto w-full max-w-4xl space-y-5"
      role="status"
      aria-live="polite"
    >
      <div className="gradient-warm h-40 rounded-2xl" aria-hidden="true" />
      <div className="-mt-16 mx-4 h-40 rounded-2xl border border-border bg-card shadow-[0_4px_24px_rgba(42,31,21,0.04)] sm:mx-6" />
      <div className="h-40 rounded-xl border border-border bg-card shadow-[0_4px_24px_rgba(42,31,21,0.04)] animate-shimmer" />
      <div className="h-64 rounded-xl border border-border bg-card shadow-[0_4px_24px_rgba(42,31,21,0.04)] animate-shimmer" />
    </div>
  )
}
