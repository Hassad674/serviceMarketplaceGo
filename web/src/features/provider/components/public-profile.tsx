"use client"

import { useQuery } from "@tanstack/react-query"
import { useTranslations } from "next-intl"
import { ArrowLeft } from "lucide-react"
import { useRouter } from "@i18n/navigation"
import { apiClient } from "@/shared/lib/api-client"
import type { Profile } from "../api/profile-api"
import { ProfileSkeleton } from "./profile-skeleton"
import { useProfileRating } from "../hooks/use-profile-rating"
import { AgencyPublicProfile } from "./agency-public-profile"

type ProfileType = "agency" | "freelancer" | "referrer"

interface PublicProfileProps {
  orgId: string
  type: ProfileType
}

// PublicProfile is the data boundary for the public /agencies/[id]
// route. It fetches the legacy /api/v1/profiles/{id} aggregate and
// hands it to the harmonized AgencyPublicProfile shell, which mirrors
// the card order, shells and spacing of FreelancePublicProfile. The
// `type` prop is kept for backward compatibility but only "agency"
// is wired — freelance and referrer routes now use their dedicated
// split-profile loaders.
export function PublicProfile({ orgId, type: _type }: PublicProfileProps) {
  const t = useTranslations("publicProfile")
  const router = useRouter()

  const { data: profile, isLoading, error } = useQuery({
    queryKey: ["public-profile", orgId],
    queryFn: () => apiClient<Profile>(`/api/v1/profiles/${orgId}`),
    staleTime: 5 * 60 * 1000,
  })
  const { data: rating } = useProfileRating(orgId)

  if (isLoading) return <ProfileSkeleton />

  if (error || !profile) {
    return (
      <div className="rounded-xl border border-destructive/30 bg-destructive/5 p-8 text-center">
        <p className="text-sm text-destructive">{t("profileNotFound")}</p>
        <button
          onClick={() => router.back()}
          className="mt-3 inline-flex items-center gap-1.5 text-sm text-primary hover:opacity-80 transition-opacity"
        >
          <ArrowLeft className="h-4 w-4" />
          {t("back")}
        </button>
      </div>
    )
  }

  const displayName = profile.title || t("untitledProfile")

  return (
    <div className="space-y-6">
      <button
        onClick={() => router.back()}
        className="inline-flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground transition-colors"
      >
        <ArrowLeft className="h-4 w-4" />
        {t("back")}
      </button>

      <AgencyPublicProfile
        profile={profile}
        orgId={orgId}
        displayName={displayName}
        rating={rating ? { average: rating.average, count: rating.count } : undefined}
      />
    </div>
  )
}
