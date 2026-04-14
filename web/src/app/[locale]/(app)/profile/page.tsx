"use client"

import { useTranslations } from "next-intl"
import { useUser, useOrganization } from "@/shared/hooks/use-user"
import { useHasPermission } from "@/shared/hooks/use-permissions"
import { useProfileRating } from "@/shared/hooks/profile/use-profile-rating"
import {
  useFreelanceProfile,
} from "@/features/freelance-profile/hooks/use-freelance-profile"
import {
  useUpdateFreelanceProfile,
} from "@/features/freelance-profile/hooks/use-update-freelance-profile"
import {
  useUploadFreelanceVideo,
  useDeleteFreelanceVideo,
} from "@/features/freelance-profile/hooks/use-freelance-video"
import { FreelancePublicProfile } from "@/features/freelance-profile/components/freelance-public-profile"
import {
  useUploadOrganizationPhoto,
} from "@/features/organization-shared/hooks/use-update-organization-photo"
import { useOrganizationShared } from "@/features/organization-shared/hooks/use-organization-shared"
import { SharedLocationSection } from "@/features/organization-shared/components/shared-location-section"
import { SharedLanguagesSection } from "@/features/organization-shared/components/shared-languages-section"
import { SkillsSection } from "@/features/skill/components/skills-section"

// /profile renders the authenticated user's freelance profile in
// editable mode. Shared fields (location, languages, photo) live in
// the organization-shared feature and are rendered ONCE on this page
// as their canonical home — /referral does not duplicate the UI.
// Writes flow through the org-shared mutation hooks, which fan out
// invalidations to every persona cache so the next render of the
// referrer page sees the fresh values.
export default function ProfilePage() {
  const { data: user } = useUser()
  const { data: org } = useOrganization()
  const { data: profile, isLoading, error } = useFreelanceProfile()
  // Prime the org-shared cache so the sections below read from the
  // same snapshot the freelance profile just delivered. TanStack
  // deduplicates the request so this is essentially free.
  useOrganizationShared()
  const { data: rating } = useProfileRating(org?.id)
  const updateProfile = useUpdateFreelanceProfile()
  const photoUpload = useUploadOrganizationPhoto()
  const videoUpload = useUploadFreelanceVideo()
  const videoDelete = useDeleteFreelanceVideo()
  const canEditProfile = useHasPermission("org_profile.edit")
  const t = useTranslations("profile")

  if (error) {
    return (
      <div className="rounded-xl border border-destructive/30 bg-destructive/5 p-8 text-center">
        <p className="text-sm text-destructive">{t("loadError")}</p>
      </div>
    )
  }
  if (isLoading || !profile) return <ProfileSkeleton />

  const displayName =
    user?.display_name ??
    `${user?.first_name ?? ""} ${user?.last_name ?? ""}`.trim()

  return (
    <div className="space-y-6">
      <FreelancePublicProfile
        profile={profile}
        displayName={displayName}
        rating={
          rating
            ? { average: rating.average, count: rating.count }
            : undefined
        }
        editable={
          canEditProfile
            ? {
                onSaveTitle: (title) =>
                  updateProfile.mutate({
                    title,
                    about: profile.about,
                    video_url: profile.video_url,
                  }),
                onSaveAbout: async (about) => {
                  await updateProfile.mutateAsync({
                    title: profile.title,
                    about,
                    video_url: profile.video_url,
                  })
                },
                savingAbout: updateProfile.isPending,
                onUploadPhoto: async (file) => {
                  await photoUpload.mutateAsync(file)
                },
                uploadingPhoto: photoUpload.isPending,
                onUploadVideo: async (file) => {
                  await videoUpload.mutateAsync(file)
                },
                uploadingVideo: videoUpload.isPending,
                onDeleteVideo: () => videoDelete.mutate(),
                deletingVideo: videoDelete.isPending,
              }
            : undefined
        }
      />

      {canEditProfile ? (
        <>
          <SharedLocationSection />
          <SharedLanguagesSection />
          <SkillsSection orgType={org?.type} readOnly={false} />
        </>
      ) : null}
    </div>
  )
}

function ProfileSkeleton() {
  return (
    <div className="space-y-6" role="status" aria-live="polite">
      <div className="h-32 rounded-xl border border-border bg-muted/40 animate-shimmer" />
      <div className="h-40 rounded-xl border border-border bg-muted/40 animate-shimmer" />
      <div className="h-64 rounded-xl border border-border bg-muted/40 animate-shimmer" />
    </div>
  )
}
