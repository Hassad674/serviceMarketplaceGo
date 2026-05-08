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
import { ExpertiseEditor } from "@/shared/components/expertise/expertise-editor"
import { useUpdateExpertiseDomains } from "@/features/provider/hooks/use-update-expertise"
import { AvailabilitySection } from "@/features/provider/components/availability-section"
import { PricingSection } from "@/features/provider/components/pricing-section"
import { LocationSection } from "@/features/provider/components/location-section"
import { LanguagesSection } from "@/features/provider/components/languages-section"
import { SkillsSection } from "@/features/skill/components/skills-section"
import { ProfileCompletionBar } from "@/features/profile-completion/components/profile-completion-bar"

// AgencyProfilePage is the editable /profile view for agency orgs.
// Visual shell mirrors the freelance editable page one-for-one — same
// max-w-5xl wrapper, editing-mode hint, completion bar, Soleil v2
// hero header, and section spacing — so the two prestataire personas
// render as a unified "prestataire" profile surface. Hook wiring stays
// legacy on purpose; the agency aggregate has not been migrated to
// the split-profile backend yet.
export function AgencyProfilePage() {
  const { data: user } = useUser()
  const { data: org } = useOrganization()
  const { data: profile, isLoading, error } = useProfile()
  const { data: rating } = useProfileRating(org?.id)
  const updateProfile = useUpdateProfile()
  const photoUpload = useUploadPhoto()
  const videoUpload = useUploadVideo()
  const videoDelete = useDeleteVideo()
  const expertiseUpdate = useUpdateExpertiseDomains()
  const canEditProfile = useHasPermission("org_profile.edit")
  const t = useTranslations("profile")

  if (isLoading) return <ProfileSkeleton />

  if (error || !profile) {
    return (
      <div className="mx-auto w-full max-w-5xl rounded-xl border border-destructive/30 bg-destructive/5 p-8 text-center">
        <p className="text-sm text-destructive">{t("loadError")}</p>
      </div>
    )
  }

  const displayName =
    user?.display_name ??
    `${user?.first_name ?? ""} ${user?.last_name ?? ""}`.trim()

  return (
    <div className="mx-auto w-full max-w-5xl space-y-5">
      <p className="font-serif text-[13px] italic text-muted-foreground">
        {t("editingMode")}
      </p>
      <ProfileCompletionBar variant="page" />
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

      <ExpertiseEditor
        domains={profile.expertise_domains}
        orgType="agency"
        readOnly={!canEditProfile}
        onSave={async (next) => {
          await expertiseUpdate.mutateAsync(next)
        }}
        saving={expertiseUpdate.isPending}
      />

      <PricingSection
        variant="direct"
        orgType="agency"
        referrerEnabled={false}
        readOnly={!canEditProfile}
      />

      <AvailabilitySection
        orgType="agency"
        referrerEnabled={false}
        variant="direct"
        readOnly={!canEditProfile}
      />

      <LocationSection orgType="agency" readOnly={!canEditProfile} />

      <LanguagesSection orgType="agency" readOnly={!canEditProfile} />

      <SkillsSection orgType="agency" readOnly={!canEditProfile} />

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

      <SocialLinksSection />

      <PortfolioSection />

      <ProjectHistorySection orgId={org?.id} />
    </div>
  )
}
