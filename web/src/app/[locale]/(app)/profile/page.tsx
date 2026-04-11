"use client"

import { useTranslations } from "next-intl"
import { useUser, useOrganization } from "@/shared/hooks/use-user"
import { useProfile, useUpdateProfile } from "@/features/provider/hooks/use-profile"
import { useProfileRating } from "@/features/provider/hooks/use-profile-rating"
import { useUploadPhoto, useUploadVideo, useDeleteVideo } from "@/features/provider/hooks/use-upload"
import { ProfileHeader } from "@/features/provider/components/profile-header"
import { ProfileVideo } from "@/features/provider/components/profile-video"
import { ProfileAbout } from "@/features/provider/components/profile-about"
import { ProfileHistory } from "@/features/provider/components/profile-history"
import { ProfileSkeleton } from "@/features/provider/components/profile-skeleton"
import { SocialLinksSection } from "@/features/provider/components/social-links-section"
import { PortfolioSection } from "@/features/provider/components/portfolio-grid"

function orgTypeToRoleContext(orgType: string | undefined): "agency" | "provider" | "referrer" {
  if (orgType === "agency") return "agency"
  return "provider"
}

export default function ProfilePage() {
  const { data: user } = useUser()
  const { data: org } = useOrganization()
  const { data: profile, isLoading, error } = useProfile()
  const { data: rating } = useProfileRating(org?.id)
  const updateProfile = useUpdateProfile()
  const photoUpload = useUploadPhoto()
  const videoUpload = useUploadVideo()
  const videoDelete = useDeleteVideo()
  const t = useTranslations("profile")

  if (isLoading) return <ProfileSkeleton />

  if (error || !profile) {
    return (
      <div className="rounded-xl border border-destructive/30 bg-destructive/5 p-8 text-center">
        <p className="text-sm text-destructive">
          {t("loadError")}
        </p>
      </div>
    )
  }

  const orgType = org?.type ?? "provider_personal"
  const displayName = user?.display_name ?? `${user?.first_name ?? ""} ${user?.last_name ?? ""}`.trim()
  const roleContext = orgTypeToRoleContext(orgType)
  const aboutLabel = orgType === "agency" ? t("aboutAgency") : t("about")
  const aboutPlaceholder = orgType === "agency" ? t("aboutAgencyPlaceholder") : t("aboutPlaceholder")
  const videoDesc = orgType === "agency" ? t("addVideoDescAgency") : undefined

  return (
    <div className="space-y-6">
      <ProfileHeader
        profile={profile}
        displayName={displayName}
        roleContext={roleContext}
        onUpdateTitle={(title) => updateProfile.mutate({ title })}
        onUploadPhoto={async (file) => { await photoUpload.mutateAsync(file) }}
        uploadingPhoto={photoUpload.isPending}
        averageRating={rating?.average}
        reviewCount={rating?.count}
      />
      <ProfileVideo
        videoUrl={profile?.presentation_video_url}
        emptyDescription={videoDesc}
        onUploadVideo={async (file) => { await videoUpload.mutateAsync(file) }}
        uploadingVideo={videoUpload.isPending}
        onDeleteVideo={() => videoDelete.mutate()}
        deletingVideo={videoDelete.isPending}
      />
      <ProfileAbout
        content={profile?.about || ""}
        onSave={async (text) => {
          await updateProfile.mutateAsync({ about: text })
        }}
        saving={updateProfile.isPending}
        label={aboutLabel}
        placeholder={aboutPlaceholder}
      />
      <SocialLinksSection />
      <PortfolioSection />
      <ProfileHistory orgId={org?.id} />
    </div>
  )
}
