"use client"

import { useTranslations } from "next-intl"
import { useUser, useOrganization } from "@/shared/hooks/use-user"
import { useHasPermission } from "@/shared/hooks/use-permissions"
import { useProfileRating } from "@/shared/hooks/profile/use-profile-rating"
import { useReferrerProfile } from "@/features/referrer-profile/hooks/use-referrer-profile"
import {
  useUpdateReferrerProfile,
  useUpdateReferrerAvailability,
  useUpdateReferrerExpertise,
} from "@/features/referrer-profile/hooks/use-update-referrer-profile"
import {
  useUploadReferrerVideo,
  useDeleteReferrerVideo,
} from "@/features/referrer-profile/hooks/use-referrer-video"
import { ReferrerPublicProfile } from "@/features/referrer-profile/components/referrer-public-profile"
import {
  useUploadOrganizationPhoto,
} from "@/features/organization-shared/hooks/use-update-organization-photo"
import { useOrganizationShared } from "@/features/organization-shared/hooks/use-organization-shared"
import { SharedLocationSection } from "@/features/organization-shared/components/shared-location-section"
import { SharedLanguagesSection } from "@/features/organization-shared/components/shared-languages-section"
import { ReferrerSocialLinksSection } from "@/features/referrer-profile/components/referrer-social-links-section"

// /referral renders the authenticated user's referrer profile in
// editable mode. Shared fields (photo, location, languages) are
// intentionally NOT rendered here — their canonical home is
// /profile, and updating them there invalidates this cache via the
// org-shared mutation fan-out.
//
// Gating: the page is only meaningful for provider accounts. An
// enterprise or agency user landing here sees the "provider only"
// explainer instead.
export default function ReferralPage() {
  const { data: user } = useUser()
  const { data: org } = useOrganization()
  const { data: profile, isLoading, error } = useReferrerProfile()
  useOrganizationShared()
  const { data: rating } = useProfileRating(org?.id)
  const updateProfile = useUpdateReferrerProfile()
  const updateAvailability = useUpdateReferrerAvailability()
  const updateExpertise = useUpdateReferrerExpertise()
  const photoUpload = useUploadOrganizationPhoto()
  const videoUpload = useUploadReferrerVideo()
  const videoDelete = useDeleteReferrerVideo()
  const canEditProfile = useHasPermission("org_profile.edit")
  const t = useTranslations("profile")
  const tReferrer = useTranslations("profile.referrer")

  if (user && user.role !== "provider") {
    return (
      <div className="rounded-xl border border-border bg-card p-8 text-center">
        <p className="text-sm text-muted-foreground">
          {tReferrer("providerOnly")}
        </p>
      </div>
    )
  }

  if (error) {
    return (
      <div className="rounded-xl border border-destructive/30 bg-destructive/5 p-8 text-center">
        <p className="text-sm text-destructive">{t("loadError")}</p>
      </div>
    )
  }
  if (isLoading || !profile) return <ReferralSkeleton />

  const displayName = user
    ? `${user.first_name ?? ""} ${user.last_name ?? ""}`.trim()
    : ""

  return (
    <div className="space-y-6">
      <ReferrerPublicProfile
        profile={profile}
        displayName={displayName}
        rating={
          rating ? { average: rating.average, count: rating.count } : undefined
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
        <>
          <SharedLocationSection />
          <SharedLanguagesSection />
          <ReferrerSocialLinksSection />
        </>
      ) : null}
    </div>
  )
}

function ReferralSkeleton() {
  return (
    <div className="space-y-6" role="status" aria-live="polite">
      <div className="h-32 rounded-xl border border-border bg-muted/40 animate-shimmer" />
      <div className="h-40 rounded-xl border border-border bg-muted/40 animate-shimmer" />
      <div className="h-64 rounded-xl border border-border bg-muted/40 animate-shimmer" />
    </div>
  )
}
