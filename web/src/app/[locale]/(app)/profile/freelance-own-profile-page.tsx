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
  useUpdateFreelanceAvailability,
  useUpdateFreelanceExpertise,
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
import { FreelanceSocialLinksSection } from "@/features/freelance-profile/components/freelance-social-links-section"

// Editable /profile view for provider_personal users. Renders the
// freelance persona: split pricing/availability/expertise on top of
// the shared identity fields (location, languages, photo) which live
// on the organization and are mirrored onto /referral via cache
// invalidation.
export function FreelanceOwnProfilePage() {
  const { data: user } = useUser()
  const { data: org } = useOrganization()
  const { data: profile, isLoading, error } = useFreelanceProfile()
  useOrganizationShared()
  const { data: rating } = useProfileRating(org?.id)
  const updateProfile = useUpdateFreelanceProfile()
  const updateAvailability = useUpdateFreelanceAvailability()
  const updateExpertise = useUpdateFreelanceExpertise()
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
    <div className="space-y-5">
      <p className="font-serif text-[13px] italic text-muted-foreground">
        {t("editingMode")}
      </p>
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
                availability: {
                  value: profile.availability_status,
                  onSave: async (next) => {
                    await updateAvailability.mutateAsync(next)
                  },
                  isSaving: updateAvailability.isPending,
                },
                expertise: {
                  value: profile.expertise_domains ?? [],
                  onSave: async (next) => {
                    await updateExpertise.mutateAsync(next)
                  },
                  isSaving: updateExpertise.isPending,
                },
              }
            : undefined
        }
      />

      {canEditProfile ? (
        <div className="space-y-5">
          <SharedLocationSection />
          <SharedLanguagesSection />
          <SkillsSection orgType={org?.type} readOnly={false} />
          <FreelanceSocialLinksSection />
        </div>
      ) : null}
    </div>
  )
}

function ProfileSkeleton() {
  return (
    <div className="space-y-5" role="status" aria-live="polite">
      <div className="gradient-warm h-40 rounded-2xl" aria-hidden="true" />
      <div className="-mt-16 mx-4 h-40 rounded-2xl border border-border bg-card shadow-[0_4px_24px_rgba(42,31,21,0.04)] sm:mx-6" />
      <div className="h-40 rounded-xl border border-border bg-card shadow-[0_4px_24px_rgba(42,31,21,0.04)] animate-shimmer" />
      <div className="h-64 rounded-xl border border-border bg-card shadow-[0_4px_24px_rgba(42,31,21,0.04)] animate-shimmer" />
    </div>
  )
}
