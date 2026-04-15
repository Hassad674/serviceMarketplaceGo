"use client"

import { useTranslations } from "next-intl"
import { ProfileAboutCard } from "@/shared/components/profile/profile-about-card"
import { ProfileVideoCard } from "@/shared/components/profile/profile-video-card"
import { ProjectHistorySection } from "@/shared/components/profile/project-history-section"
import { LocationDisplayCard } from "@/shared/components/profile/location-display-card"
import { LanguagesDisplayCard } from "@/shared/components/profile/languages-display-card"
import { PricingDisplayCard } from "@/shared/components/profile/pricing-display-card"
import {
  ProfileIdentityHeader,
  type ProfileIdentityHeaderProps,
} from "@/shared/components/ui/profile-identity-header"
import type { Profile } from "../api/profile-api"
import { ExpertiseDisplay } from "./expertise-display"
import { SkillsDisplay } from "./skills-display"
import { PublicPortfolioSection } from "./portfolio-grid"

// AgencyPublicProfile renders the public /agencies/[id] surface with
// the same card order, shells and spacing as FreelancePublicProfile.
// Product scope stays agency-flavored (portfolio section lives here,
// skills card is dedicated, expertise uses the agency taxonomy) but
// the visual shell is harmonized — identity header, about, video,
// pricing/location/languages display cards, project history — all
// pulled from shared/components/profile so the two surfaces drift in
// lockstep from now on.
export interface AgencyPublicProfileProps {
  profile: Profile
  orgId: string
  displayName: string
  rating?: { average: number; count: number }
}

export function AgencyPublicProfile(props: AgencyPublicProfileProps) {
  const { profile, orgId, displayName, rating } = props
  const t = useTranslations("profile")
  const tSkills = useTranslations("profile.skillsDisplay")

  const identity: ProfileIdentityHeaderProps["identity"] = {
    photoUrl: profile.photo_url,
    displayName,
    title: profile.title,
    availabilityStatus: profile.availability_status,
  }

  const directPricing = pickDirectPricing(profile)

  return (
    <div className="space-y-6">
      <ProfileIdentityHeader identity={identity} rating={rating} />

      <ProfileAboutCard
        content={profile.about}
        label={t("aboutAgency")}
        placeholder={t("aboutAgencyPlaceholder")}
        readOnly
      />

      <ProfileVideoCard
        videoUrl={profile.presentation_video_url}
        labels={{
          title: t("videoTitle"),
          emptyLabel: t("noVideo"),
          emptyDescription: t("addVideoDescAgency"),
        }}
        readOnly
      />

      <ExpertiseDisplay
        domains={profile.expertise_domains}
        orgType="agency"
      />

      {profile.skills && profile.skills.length > 0 ? (
        <section
          aria-labelledby="agency-skills-display-title"
          className="bg-card border border-border rounded-xl p-6 shadow-sm"
        >
          <h2
            id="agency-skills-display-title"
            className="text-lg font-semibold text-foreground mb-3"
          >
            {tSkills("sectionTitle")}
          </h2>
          <SkillsDisplay skills={profile.skills} />
        </section>
      ) : null}

      <PricingDisplayCard
        pricing={directPricing}
        titleKey="directSectionTitle"
      />

      <LocationDisplayCard
        city={profile.city ?? ""}
        countryCode={profile.country_code ?? ""}
        workMode={profile.work_mode ?? []}
        travelRadiusKm={profile.travel_radius_km ?? null}
      />

      <LanguagesDisplayCard
        professional={profile.languages_professional ?? []}
        conversational={profile.languages_conversational ?? []}
      />

      <PublicPortfolioSection orgId={orgId} />

      <ProjectHistorySection orgId={orgId} readOnly />
    </div>
  )
}

// pickDirectPricing adapts the legacy agency pricing shape (an array
// of rows keyed by `kind`) to the single-row contract the shared
// PricingDisplayCard expects. Agencies only advertise a direct rate
// on the public page — referral commissions are a persona-scoped
// feature that lives on the referrer profile, not here. Returns null
// when no direct row exists so the card hides itself.
function pickDirectPricing(profile: Profile) {
  const row = profile.pricing?.find((p) => p.kind === "direct")
  if (!row) return null
  return {
    type: row.type,
    min_amount: row.min_amount,
    max_amount: row.max_amount,
    currency: row.currency,
    note: row.note,
    negotiable: row.negotiable,
  }
}
