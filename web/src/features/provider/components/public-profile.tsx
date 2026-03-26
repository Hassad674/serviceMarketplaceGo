"use client"

import { useQuery } from "@tanstack/react-query"
import { useTranslations } from "next-intl"
import { ArrowLeft } from "lucide-react"
import { Link } from "@i18n/navigation"
import { apiClient } from "@/shared/lib/api-client"
import type { Profile } from "../api/profile-api"
import { ProfileHeader } from "./profile-header"
import { ProfileVideo } from "./profile-video"
import { ProfileAbout } from "./profile-about"
import { ProfileHistory } from "./profile-history"
import { ProfileSkeleton } from "./profile-skeleton"

type ProfileType = "agency" | "freelancer" | "referrer"

const TYPE_BACK_LINKS: Record<ProfileType, string> = {
  agency: "/agencies",
  freelancer: "/freelancers",
  referrer: "/freelancers",
}

const TYPE_BACK_LABELS: Record<ProfileType, string> = {
  agency: "backToAgencies",
  freelancer: "backToFreelancers",
  referrer: "backToFreelancers",
}

interface PublicProfileProps {
  userId: string
  type: ProfileType
}

export function PublicProfile({ userId, type }: PublicProfileProps) {
  const t = useTranslations("publicProfile")

  const { data: profile, isLoading, error } = useQuery({
    queryKey: ["public-profile", userId],
    queryFn: () => apiClient<Profile>(`/api/v1/profiles/${userId}`),
    staleTime: 5 * 60 * 1000,
  })

  if (isLoading) return <ProfileSkeleton />

  if (error || !profile) {
    return (
      <div className="rounded-xl border border-destructive/30 bg-destructive/5 p-8 text-center">
        <p className="text-sm text-destructive">
          {t("profileNotFound")}
        </p>
        <Link
          href={TYPE_BACK_LINKS[type]}
          className="mt-3 inline-flex items-center gap-1.5 text-sm text-primary hover:opacity-80 transition-opacity"
        >
          <ArrowLeft className="h-4 w-4" />
          {t(TYPE_BACK_LABELS[type])}
        </Link>
      </div>
    )
  }

  const roleContext = type === "agency" ? "agency" : type === "referrer" ? "referrer" : "provider"
  const displayName = profile.title || t("untitledProfile")
  const videoUrl = roleContext === "referrer" ? profile.referrer_video_url : profile.presentation_video_url
  const aboutText = roleContext === "referrer" ? profile.referrer_about : profile.about

  return (
    <div className="space-y-6">
      <Link
        href={TYPE_BACK_LINKS[type]}
        className="inline-flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground transition-colors"
      >
        <ArrowLeft className="h-4 w-4" />
        {t(TYPE_BACK_LABELS[type])}
      </Link>

      <div className="space-y-4">
        <ProfileHeader
          profile={profile}
          displayName={displayName}
          roleContext={roleContext}
          readOnly
        />

        <ProfileVideo
          videoUrl={videoUrl}
          readOnly
        />

        <ProfileAbout
          content={aboutText || ""}
          readOnly
        />

        <ProfileHistory readOnly />
      </div>
    </div>
  )
}
