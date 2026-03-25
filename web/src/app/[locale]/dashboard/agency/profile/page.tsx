"use client"

import { useTranslations } from "next-intl"
import { useAuth } from "@/shared/hooks/use-auth"
import { useProfile, useUpdateProfile } from "@/features/provider/hooks/use-profile"
import { useUploadPhoto, useUploadVideo, useDeleteVideo } from "@/features/provider/hooks/use-upload"
import { ProfileHeader } from "@/features/provider/components/profile-header"
import { ProfileVideo } from "@/features/provider/components/profile-video"
import { ProfileAbout } from "@/features/provider/components/profile-about"
import { ProfileHistory } from "@/features/provider/components/profile-history"
import { ProfileSkeleton } from "@/features/provider/components/profile-skeleton"

export default function AgencyProfilePage() {
  const { user } = useAuth()
  const { data: profile, isLoading } = useProfile()
  const updateProfile = useUpdateProfile()
  const photoUpload = useUploadPhoto()
  const videoUpload = useUploadVideo()
  const videoDelete = useDeleteVideo()
  const t = useTranslations("profile")

  if (isLoading) return <ProfileSkeleton />

  const displayName = user?.display_name ?? ""

  return (
    <div className="space-y-6">
      <ProfileHeader
        profile={profile}
        displayName={displayName}
        roleContext="agency"
        onUpdateTitle={(title) => updateProfile.mutate({ title })}
        onUploadPhoto={async (file) => { await photoUpload.mutateAsync(file) }}
        uploadingPhoto={photoUpload.isPending}
      />
      <ProfileVideo
        videoUrl={profile?.presentation_video_url}
        emptyDescription={t("addVideoDescAgency")}
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
        label={t("aboutAgency")}
        placeholder={t("aboutAgencyPlaceholder")}
      />
      <ProfileHistory />
    </div>
  )
}
