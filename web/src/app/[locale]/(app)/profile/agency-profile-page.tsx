"use client"

import { useTranslations } from "next-intl"
import { useUser, useOrganization } from "@/shared/hooks/use-user"
import { useHasPermission } from "@/shared/hooks/use-permissions"
import {
  useProfile,
  useUpdateProfile,
} from "@/features/provider/hooks/use-profile"
import { useProfileRating } from "@/shared/hooks/profile/use-profile-rating"
import {
  useUploadPhoto,
  useUploadVideo,
  useDeleteVideo,
} from "@/features/provider/hooks/use-upload"
import { ProfileAboutCard } from "@/shared/components/profile/profile-about-card"
import { ProfileVideoCard } from "@/shared/components/profile/profile-video-card"
import { ProjectHistorySection } from "@/shared/components/profile/project-history-section"
import { AgencyProfileHeader } from "@/features/provider/components/agency-profile-header"
import { ProfileSkeleton } from "@/features/provider/components/profile-skeleton"
import { SocialLinksSection } from "@/features/provider/components/social-links-section"
import { PortfolioSection } from "@/features/provider/components/portfolio-grid"
import { ExpertiseEditor } from "@/features/provider/components/expertise-editor"
import { AvailabilitySection } from "@/features/provider/components/availability-section"
import { PricingSection } from "@/features/provider/components/pricing-section"
import { LocationSection } from "@/features/provider/components/location-section"
import { LanguagesSection } from "@/features/provider/components/languages-section"
import { SkillsSection } from "@/features/skill/components/skills-section"

// Agency editable profile page — uses the legacy /api/v1/profile
// endpoints via the restored provider hooks. Visual shell is now
// harmonized with the freelance editable page: shared profile
// header, ProfileAboutCard and ProfileVideoCard shells, plus the
// same card spacing so the two surfaces drift in lockstep. Hook
// wiring stays legacy on purpose — agencies have not been migrated
// to the split-profile backend yet.
export function AgencyProfilePage() {
  const { data: user } = useUser()
  const { data: org } = useOrganization()
  const { data: profile, isLoading, error } = useProfile()
  const { data: rating } = useProfileRating(org?.id)
  const updateProfile = useUpdateProfile()
  const photoUpload = useUploadPhoto()
  const videoUpload = useUploadVideo()
  const videoDelete = useDeleteVideo()
  const canEditProfile = useHasPermission("org_profile.edit")
  const t = useTranslations("profile")

  if (isLoading) return <ProfileSkeleton />

  if (error || !profile) {
    return (
      <div className="rounded-xl border border-destructive/30 bg-destructive/5 p-8 text-center">
        <p className="text-sm text-destructive">{t("loadError")}</p>
      </div>
    )
  }

  const displayName =
    user?.display_name ??
    `${user?.first_name ?? ""} ${user?.last_name ?? ""}`.trim()

  return (
    <div className="space-y-6">
      <AgencyProfileHeader
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
                onSaveTitle: (title) => updateProfile.mutate({ title }),
                onUploadPhoto: async (file) => {
                  await photoUpload.mutateAsync(file)
                },
                uploadingPhoto: photoUpload.isPending,
              }
            : undefined
        }
      />

      <AvailabilitySection
        orgType="agency"
        referrerEnabled={false}
        variant="direct"
        readOnly={!canEditProfile}
      />

      <PricingSection
        variant="direct"
        orgType="agency"
        referrerEnabled={false}
        readOnly={!canEditProfile}
      />

      <LocationSection orgType="agency" readOnly={!canEditProfile} />

      <LanguagesSection orgType="agency" readOnly={!canEditProfile} />

      <ProfileAboutCard
        content={profile.about ?? ""}
        label={t("aboutAgency")}
        placeholder={t("aboutAgencyPlaceholder")}
        onSave={
          canEditProfile
            ? async (text) => {
                await updateProfile.mutateAsync({ about: text })
              }
            : undefined
        }
        saving={updateProfile.isPending}
        readOnly={!canEditProfile}
      />

      <ProfileVideoCard
        videoUrl={profile.presentation_video_url ?? ""}
        labels={{
          title: t("videoTitle"),
          emptyLabel: t("noVideo"),
          emptyDescription: t("addVideoDescAgency"),
        }}
        actions={
          canEditProfile
            ? {
                onUpload: async (file) => {
                  await videoUpload.mutateAsync(file)
                },
                uploading: videoUpload.isPending,
                onDelete: () => videoDelete.mutate(),
                deleting: videoDelete.isPending,
              }
            : undefined
        }
        readOnly={!canEditProfile}
      />

      <ExpertiseEditor
        domains={profile.expertise_domains}
        orgType="agency"
        readOnly={!canEditProfile}
      />

      <SkillsSection orgType="agency" readOnly={!canEditProfile} />

      <SocialLinksSection />

      <PortfolioSection />

      <ProjectHistorySection orgId={org?.id} />
    </div>
  )
}
