"use client"

import { useTranslations } from "next-intl"
import { useUser, useOrganization } from "@/shared/hooks/use-user"
import { useHasPermission } from "@/shared/hooks/use-permissions"
import { useProfile, useUpdateProfile } from "@/features/provider/hooks/use-profile"
import { useProfileRating } from "@/features/provider/hooks/use-profile-rating"
import { useUploadPhoto, useUploadVideo, useDeleteVideo } from "@/features/provider/hooks/use-upload"
import { ProfileHeader } from "@/features/provider/components/profile-header"
import { ProfileVideo } from "@/features/provider/components/profile-video"
import { ProfileAbout } from "@/features/provider/components/profile-about"
import { ProfileHistory } from "@/features/provider/components/profile-history"
import { ProfileSkeleton } from "@/features/provider/components/profile-skeleton"
import { SocialLinksSection } from "@/features/provider/components/social-links-section"
import { PortfolioSection } from "@/features/provider/components/portfolio-grid"
import { ExpertiseEditor } from "@/features/provider/components/expertise-editor"
import { SkillsSection } from "@/features/skill/components/skills-section"

function orgTypeToRoleContext(orgType: string | undefined): "agency" | "provider" | "referrer" {
  if (orgType === "agency") return "agency"
  return "provider"
}

export default function ProfilePage() {
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
        <p className="text-sm text-destructive">
          {t("loadError")}
        </p>
      </div>
    )
  }

  const orgType = org?.type ?? "provider_personal"
  const displayName = user?.display_name ?? `${user?.first_name ?? ""} ${user?.last_name ?? ""}`.trim()
  const roleContext = orgTypeToRoleContext(orgType)
  const aboutLabel = orgType === "agency" ? t("aboutAgency") : t("about")
  const aboutPlaceholder = orgType === "agency" ? t("aboutAgencyPlaceholder") : t("aboutPlaceholder")
  const videoDesc = orgType === "agency" ? t("addVideoDescAgency") : undefined

  return (
    <div className="space-y-6">
      <ProfileHeader
        profile={profile}
        displayName={displayName}
        roleContext={roleContext}
        onUpdateTitle={canEditProfile ? (title) => updateProfile.mutate({ title }) : undefined}
        onUploadPhoto={canEditProfile ? async (file) => { await photoUpload.mutateAsync(file) } : undefined}
        uploadingPhoto={photoUpload.isPending}
        readOnly={!canEditProfile}
        averageRating={rating?.average}
        reviewCount={rating?.count}
      />
      <ProfileVideo
        videoUrl={profile?.presentation_video_url}
        emptyDescription={videoDesc}
        onUploadVideo={canEditProfile ? async (file) => { await videoUpload.mutateAsync(file) } : undefined}
        uploadingVideo={videoUpload.isPending}
        onDeleteVideo={canEditProfile ? () => videoDelete.mutate() : undefined}
        deletingVideo={videoDelete.isPending}
        readOnly={!canEditProfile}
      />
      <ProfileAbout
        content={profile?.about || ""}
        onSave={canEditProfile ? async (text) => {
          await updateProfile.mutateAsync({ about: text })
        } : undefined}
        saving={updateProfile.isPending}
        label={aboutLabel}
        placeholder={aboutPlaceholder}
        readOnly={!canEditProfile}
      />
      <ExpertiseEditor
        domains={profile?.expertise_domains}
        orgType={orgType}
        readOnly={!canEditProfile}
      />
      <SkillsSection
        orgType={orgType}
        expertiseKeys={profile?.expertise_domains}
        readOnly={!canEditProfile}
      />
      <SocialLinksSection />
      <PortfolioSection />
      <ProfileHistory orgId={org?.id} />
    </div>
  )
}
