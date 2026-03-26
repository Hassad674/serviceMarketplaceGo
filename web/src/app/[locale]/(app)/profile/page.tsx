"use client"

import { useTranslations } from "next-intl"
import { useUser } from "@/shared/hooks/use-user"
import { useProfile, useUpdateProfile } from "@/features/provider/hooks/use-profile"
import { useUploadPhoto, useUploadVideo, useDeleteVideo } from "@/features/provider/hooks/use-upload"
import { ProfileHeader } from "@/features/provider/components/profile-header"
import { ProfileVideo } from "@/features/provider/components/profile-video"
import { ProfileAbout } from "@/features/provider/components/profile-about"
import { ProfileHistory } from "@/features/provider/components/profile-history"
import { ProfileSkeleton } from "@/features/provider/components/profile-skeleton"

function getDisplayName(
  user: { first_name: string; last_name: string; display_name: string; role: string } | null,
): string {
  if (!user) return ""
  if (user.role === "provider") return `${user.first_name} ${user.last_name}`
  return user.display_name ?? ""
}

function getRoleContext(role: string): "agency" | "provider" | "referrer" {
  if (role === "agency") return "agency"
  if (role === "provider") return "provider"
  return "provider"
}

export default function ProfilePage() {
  const { data: user } = useUser()
  const { data: profile, isLoading, error } = useProfile()
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

  const role = user?.role ?? "provider"
  const displayName = getDisplayName(user ?? null)
  const roleContext = getRoleContext(role)
  const aboutLabel = role === "agency" ? t("aboutAgency") : t("about")
  const aboutPlaceholder = role === "agency" ? t("aboutAgencyPlaceholder") : t("aboutPlaceholder")
  const videoDesc = role === "agency" ? t("addVideoDescAgency") : undefined

  return (
    <div className="space-y-6">
      <ProfileHeader
        profile={profile}
        displayName={displayName}
        roleContext={roleContext}
        onUpdateTitle={(title) => updateProfile.mutate({ title })}
        onUploadPhoto={async (file) => { await photoUpload.mutateAsync(file) }}
        uploadingPhoto={photoUpload.isPending}
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
      <ProfileHistory />
    </div>
  )
}
