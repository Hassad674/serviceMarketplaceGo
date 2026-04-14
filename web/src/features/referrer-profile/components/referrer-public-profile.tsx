"use client"

import { useTranslations } from "next-intl"
import { ProfileAboutCard } from "@/shared/components/profile/profile-about-card"
import { ProfileVideoCard } from "@/shared/components/profile/profile-video-card"
import { ProjectHistorySection } from "@/shared/components/profile/project-history-section"
import { ExpertiseDisplay } from "@/shared/components/profile/expertise-display"
import {
  AvailabilityEditorCard,
  type AvailabilityStatus,
} from "@/shared/components/profile/availability-editor-card"
import { ExpertiseEditor } from "@/features/provider/components/expertise-editor"
import type { ReferrerProfile } from "../api/referrer-profile-api"
import { ReferrerProfileHeader } from "./referrer-profile-header"
import { ReferrerPricingSection } from "./referrer-pricing-section"

export interface ReferrerPublicProfileProps {
  profile: ReferrerProfile
  displayName: string
  rating?: { average: number; count: number }
  editable?: EditableWiring
}

export interface EditableWiring {
  onSaveTitle?: (next: string) => void
  onSaveAbout?: (next: string) => Promise<void>
  savingAbout?: boolean
  onUploadPhoto?: (file: File) => Promise<void>
  uploadingPhoto?: boolean
  onUploadVideo?: (file: File) => Promise<void>
  uploadingVideo?: boolean
  onDeleteVideo?: () => void
  deletingVideo?: boolean
  availability?: {
    value: AvailabilityStatus
    onSave: (next: AvailabilityStatus) => Promise<void>
    isSaving: boolean
  }
  expertise?: {
    value: string[]
    onSave: (next: string[]) => Promise<void>
    isSaving: boolean
  }
}

// ReferrerPublicProfile mirrors the freelance counterpart structurally
// but drops the skills section (skills live with the freelance
// persona) and hands the history card a persona-specific empty state
// — the real "referral deals" history will replace this placeholder
// when a dedicated referral_deals feature ships.
export function ReferrerPublicProfile(props: ReferrerPublicProfileProps) {
  const { profile, displayName, rating, editable } = props
  const t = useTranslations("profile")
  const tReferrer = useTranslations("profile.referrer")
  const readOnly = !editable

  return (
    <div className="space-y-6">
      <ReferrerProfileHeader
        profile={profile}
        displayName={displayName}
        rating={rating}
        editable={
          editable
            ? {
                onSaveTitle: editable.onSaveTitle,
                onUploadPhoto: editable.onUploadPhoto,
                uploadingPhoto: editable.uploadingPhoto,
              }
            : undefined
        }
      />

      <ProfileAboutCard
        content={profile.about}
        label={t("aboutReferrer")}
        placeholder={t("aboutReferrerPlaceholder")}
        onSave={editable?.onSaveAbout}
        saving={editable?.savingAbout}
        readOnly={readOnly}
      />

      <ProfileVideoCard
        videoUrl={profile.video_url}
        labels={{
          title: t("videoTitleReferrer"),
          emptyLabel: t("noVideoReferrer"),
          emptyDescription: t("addVideoDescReferrer"),
        }}
        actions={
          editable
            ? {
                onUpload: editable.onUploadVideo,
                uploading: editable.uploadingVideo,
                onDelete: editable.onDeleteVideo,
                deleting: editable.deletingVideo,
              }
            : undefined
        }
        readOnly={readOnly}
      />

      {editable?.availability ? (
        <AvailabilityEditorCard
          value={editable.availability.value}
          onSave={editable.availability.onSave}
          isSaving={editable.availability.isSaving}
        />
      ) : null}

      {editable?.expertise ? (
        <ExpertiseEditor
          domains={editable.expertise.value}
          orgType="provider_personal"
          onSaveOverride={editable.expertise.onSave}
          savingOverride={editable.expertise.isSaving}
        />
      ) : (
        <ExpertiseDisplay domains={profile.expertise_domains} />
      )}

      <ReferrerPricingSection readOnly={readOnly} />

      {/* History is always empty on referrer profiles — we force the
          empty state so the section reads "no deals yet" instead of
          leaking the freelance project history (both personas share
          the same organization_id). A dedicated referral_deals feed
          will replace this placeholder when that feature ships. */}
      <ProjectHistorySection
        orgId={profile.organization_id}
        readOnly={readOnly}
        forceEmpty
        emptyOverride={{
          title: tReferrer("historyEmptyTitle"),
          description: tReferrer("historyEmptyDescription"),
        }}
      />
    </div>
  )
}
