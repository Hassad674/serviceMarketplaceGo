"use client"

import { useAuth } from "@/shared/hooks/use-auth"
import { useProfile, useUpdateProfile } from "@/features/provider/hooks/use-profile"
import { useUploadPhoto, useUploadReferrerVideo } from "@/features/provider/hooks/use-upload"
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
        title="Presentation Video — Business Referrer"
        emptyLabel="No referrer video"
        emptyDescription="Add a video to present your business referrer activity"
        onUploadVideo={async (file) => { await referrerVideoUpload.mutateAsync(file) }}
        uploadingVideo={referrerVideoUpload.isPending}
      />
      <ProfileAbout
        content={profile?.referrer_about || ""}
        onSave={async (text) => {
          await updateProfile.mutateAsync({ referrer_about: text })
        }}
        saving={updateProfile.isPending}
        label="About the business referrer"
        placeholder="Describe your business referrer activity..."
      />
    </div>
  )
}
