"use client"

import { useTranslations } from "next-intl"
import { useAuth } from "@/shared/hooks/use-auth"
import { useProfile, useUpdateProfile } from "@/features/provider/hooks/use-profile"
import { useUploadPhoto, useUploadReferrerVideo, useDeleteReferrerVideo } from "@/features/provider/hooks/use-upload"
import { ProfileAbout } from "@/features/provider/components/profile-about"
import { ProfileHeader } from "@/features/provider/components/profile-header"
import { ProfileVideo } from "@/features/provider/components/profile-video"
import { ProfileSkeleton } from "@/features/provider/components/profile-skeleton"

export default function ProviderReferralPage() {
  const { user } = useAuth()
  const { data: profile, isLoading } = useProfile()
  const updateProfile = useUpdateProfile()
  const photoUpload = useUploadPhoto()
  const referrerVideoUpload = useUploadReferrerVideo()
  const referrerVideoDelete = useDeleteReferrerVideo()
  const t = useTranslations("profile")

  if (isLoading) return <ProfileSkeleton />

  const displayName = user
    ? `${user.first_name} ${user.last_name}`
    : ""

  return (
    <div className="space-y-6">
      <ProfileHeader
        profile={profile}
        displayName={displayName}
        roleContext="referrer"
        onUpdateTitle={(title) => updateProfile.mutate({ title })}
        onUploadPhoto={async (file) => { await photoUpload.mutateAsync(file) }}
        uploadingPhoto={photoUpload.isPending}
      />
      <ProfileVideo
        videoUrl={profile?.referrer_video_url}
        title={t("videoTitleReferrer")}
        emptyLabel={t("noVideoReferrer")}
        emptyDescription={t("addVideoDescReferrer")}
        onUploadVideo={async (file) => { await referrerVideoUpload.mutateAsync(file) }}
        uploadingVideo={referrerVideoUpload.isPending}
        onDeleteVideo={() => referrerVideoDelete.mutate()}
        deletingVideo={referrerVideoDelete.isPending}
      />
      <ProfileAbout
        content={profile?.referrer_about || ""}
        onSave={async (text) => {
          await updateProfile.mutateAsync({ referrer_about: text })
        }}
        saving={updateProfile.isPending}
        label={t("aboutReferrer")}
        placeholder={t("aboutReferrerPlaceholder")}
      />
    </div>
  )
}
