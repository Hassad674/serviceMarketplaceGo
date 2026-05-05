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
import { ExpertiseEditor } from "@/shared/components/expertise/expertise-editor"
import type { FreelanceProfile } from "../api/freelance-profile-api"
import { FreelanceProfileHeader } from "./freelance-profile-header"
import { FreelancePricingSection } from "./freelance-pricing-section"
import { FreelanceSkillsStrip } from "./freelance-skills-strip"

// FreelancePublicProfileProps keeps the public-profile surface below
// the 4-prop cap by grouping the optional owner-side handlers into a
// single `editable` object. When that object is absent the component
// renders in read-only mode — this is the shape used by the public
// /freelancers/[id] route.
export interface FreelancePublicProfileProps {
  profile: FreelanceProfile
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

// FreelancePublicProfile is the single source of truth for both the
// owner-edited /profile page and the public /freelancers/[id] page.
// Gating is explicit via the `editable` prop — when absent, every
// section switches to read-only and any empty section collapses so
// the public viewer sees a tight summary card instead of placeholder
// text.
export function FreelancePublicProfile(props: FreelancePublicProfileProps) {
  const { profile, displayName, rating, editable } = props
  const t = useTranslations("profile")
  const readOnly = !editable

  // Soleil v2 W-16 v3 ordering — sections rendered top-to-bottom:
  //   1. Header (cover band + portrait)
  //   2. À propos
  //   3. Domaines d'expertise
  //   4. Tarifs
  //   5. Disponibilité (private only — public viewers see the pill in the header)
  //   6. Localisation (public only — owners edit via SharedLocationSection)
  //   7. Langues (public only — owners edit via SharedLanguagesSection)
  //   8. Compétences (public only — owners edit via SkillsSection)
  //   9. Vidéo
  // Réseaux sociaux + Historique des projets are appended by the
  // consuming page (loader for public, own-profile-page for editable)
  // so the project history can sit LAST after the persona-specific
  // shared editors land. See brief BATCH-PROFIL-FIX items #2 + #3.
  // The aerated max-width wrapper (max-w-3xl) keeps the editorial
  // column under ~960px even on wide dashboards (item #1).
  return (
    <div className="mx-auto w-full max-w-4xl space-y-5">
      <FreelanceProfileHeader
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
        label={t("about")}
        placeholder={t("aboutPlaceholder")}
        onSave={editable?.onSaveAbout}
        saving={editable?.savingAbout}
        readOnly={readOnly}
      />

      {editable?.expertise ? (
        <ExpertiseEditor
          domains={editable.expertise.value}
          orgType="provider_personal"
          onSave={editable.expertise.onSave}
          saving={editable.expertise.isSaving}
        />
      ) : (
        <ExpertiseDisplay domains={profile.expertise_domains} />
      )}

      {readOnly ? (
        <PricingDisplayCard
          pricing={profile.pricing}
          titleKey="directSectionTitle"
        />
      ) : (
        <FreelancePricingSection readOnly={false} />
      )}

      {editable?.availability ? (
        <AvailabilityEditorCard
          value={editable.availability.value}
          onSave={editable.availability.onSave}
          isSaving={editable.availability.isSaving}
        />
      ) : null}

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

      {readOnly ? (
        <FreelanceSkillsStrip skills={profile.skills} />
      ) : null}

      <ProfileVideoCard
        videoUrl={profile.video_url}
        labels={{
          title: t("videoTitle"),
          emptyLabel: t("noVideo"),
          emptyDescription: t("addVideoDesc"),
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
        showWhenEmpty={readOnly}
      />
    </div>
  )
}
