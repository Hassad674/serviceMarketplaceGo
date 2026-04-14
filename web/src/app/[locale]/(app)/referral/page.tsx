"use client"

import { useTranslations } from "next-intl"
import { useUser, useOrganization } from "@/shared/hooks/use-user"
import { useHasPermission } from "@/shared/hooks/use-permissions"
import { useProfile, useUpdateProfile } from "@/features/provider/hooks/use-profile"
import { useUploadPhoto, useUploadReferrerVideo, useDeleteReferrerVideo } from "@/features/provider/hooks/use-upload"
import { ProfileAbout } from "@/features/provider/components/profile-about"
import { ProfileHeader } from "@/features/provider/components/profile-header"
import { ProfileVideo } from "@/features/provider/components/profile-video"
import { ProfileSkeleton } from "@/features/provider/components/profile-skeleton"
import { PricingSection } from "@/features/provider/components/pricing-section"

export default function ReferralPage() {
  const { data: user } = useUser()
  const { data: org } = useOrganization()
  const { data: profile, isLoading, error } = useProfile()
  const updateProfile = useUpdateProfile()
  const photoUpload = useUploadPhoto()
  const referrerVideoUpload = useUploadReferrerVideo()
  const referrerVideoDelete = useDeleteReferrerVideo()
  const canEditProfile = useHasPermission("org_profile.edit")
  const t = useTranslations("profile")

  // Referral page is only meaningful for providers
  if (user && user.role !== "provider") {
    return (
      <div className="rounded-xl border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 p-8 text-center">
        <p className="text-sm text-gray-500 dark:text-gray-400">
          {t("referralProviderOnly")}
        </p>
      </div>
    )
  }

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

  const displayName = user
    ? `${user.first_name} ${user.last_name}`
    : ""

  return (
    <div className="space-y-6">
      <ProfileHeader
        profile={profile}
        displayName={displayName}
        roleContext="referrer"
        onUpdateTitle={canEditProfile ? (title) => updateProfile.mutate({ title }) : undefined}
        onUploadPhoto={canEditProfile ? async (file) => { await photoUpload.mutateAsync(file) } : undefined}
        uploadingPhoto={photoUpload.isPending}
        readOnly={!canEditProfile}
      />
      <PricingSection
        variant="referral"
        orgType={org?.type}
        referrerEnabled={user?.referrer_enabled}
        readOnly={!canEditProfile}
      />
      <ProfileVideo
        videoUrl={profile?.referrer_video_url}
        title={t("videoTitleReferrer")}
        emptyLabel={t("noVideoReferrer")}
        emptyDescription={t("addVideoDescReferrer")}
        onUploadVideo={canEditProfile ? async (file) => { await referrerVideoUpload.mutateAsync(file) } : undefined}
        uploadingVideo={referrerVideoUpload.isPending}
        onDeleteVideo={canEditProfile ? () => referrerVideoDelete.mutate() : undefined}
        deletingVideo={referrerVideoDelete.isPending}
        readOnly={!canEditProfile}
      />
      <ProfileAbout
        content={profile?.referrer_about || ""}
        onSave={canEditProfile ? async (text) => {
          await updateProfile.mutateAsync({ referrer_about: text })
        } : undefined}
        saving={updateProfile.isPending}
        label={t("aboutReferrer")}
        placeholder={t("aboutReferrerPlaceholder")}
        readOnly={!canEditProfile}
      />
    </div>
  )
}
