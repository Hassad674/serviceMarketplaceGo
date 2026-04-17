"use client"

import { useTranslations } from "next-intl"
import { ProfileAboutCard } from "@/shared/components/profile/profile-about-card"
import { ProfileVideoCard } from "@/shared/components/profile/profile-video-card"
import { ExpertiseDisplay } from "@/shared/components/profile/expertise-display"
import { LocationDisplayCard } from "@/shared/components/profile/location-display-card"
import { LanguagesDisplayCard } from "@/shared/components/profile/languages-display-card"
import { PricingDisplayCard } from "@/shared/components/profile/pricing-display-card"
import {
  AvailabilityEditorCard,
  type AvailabilityStatus,
} from "@/shared/components/profile/availability-editor-card"
import { ExpertiseEditor } from "@/features/provider/components/expertise-editor"
import type { ReferrerProfile } from "../api/referrer-profile-api"
import { ReferrerProfileHeader } from "./referrer-profile-header"
import { ReferrerPricingSection } from "./referrer-pricing-section"
import { ReferrerProjectHistorySection } from "./referrer-project-history-section"

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
// persona) and uses the dedicated apporteur reputation history — the
// rating reflects client reviews on the providers introduced through
// this user's referrals, NOT the user's own freelance rating.
export function ReferrerPublicProfile(props: ReferrerPublicProfileProps) {
  const { profile, displayName, rating, editable } = props
  const t = useTranslations("profile")
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

      {readOnly ? (
        <PricingDisplayCard
          pricing={profile.pricing}
          titleKey="referralSectionTitle"
        />
      ) : (
        <ReferrerPricingSection readOnly={false} />
      )}

      {readOnly ? (
        <LocationDisplayCard
          city={profile.city}
          countryCode={profile.country_code}
          workMode={profile.work_mode}
          travelRadiusKm={profile.travel_radius_km}
        />
      ) : null}

      {readOnly ? (
        <LanguagesDisplayCard
          professional={profile.languages_professional}
          conversational={profile.languages_conversational}
        />
      ) : null}

      {/* Apporteur reputation surface — rating + "projets apportés"
          history. Scope: missions attributed to this user's referrals
          during the exclusivity window. Client identity is never
          exposed; only the provider name and the client's review of
          the provider appear here. */}
      <ReferrerProjectHistorySection orgId={profile.organization_id} />
    </div>
  )
}
