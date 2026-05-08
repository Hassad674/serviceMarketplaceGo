"use client"

import { useTranslations } from "next-intl"
import { ProfileAboutCard } from "@/shared/components/profile/profile-about-card"
import { ProfileVideoCard } from "@/shared/components/profile/profile-video-card"
import { ProjectHistorySection } from "@/shared/components/profile/project-history-section"
import { LocationDisplayCard } from "@/shared/components/profile/location-display-card"
import { LanguagesDisplayCard } from "@/shared/components/profile/languages-display-card"
import { PricingDisplayCard } from "@/shared/components/profile/pricing-display-card"
import { ExpertiseDisplay } from "@/shared/components/profile/expertise-display"
import type { Profile } from "../api/profile-api"
import { AgencyProfileHeader } from "./agency-profile-header"
import { SkillsDisplay } from "./skills-display"
import { PublicPortfolioSection } from "./portfolio-grid"

// AgencyPublicProfile renders the public /agencies/[id] surface. It
// uses the same Soleil v2 shell as FreelancePublicProfile so an
// agency profile reads as a first-class prestataire card — same
// hero, same card spacing, same section ordering. The agency-only
// surfaces (skills card, portfolio) sit at the same nesting level as
// the freelance equivalents so the two pages drift in lockstep.
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

  return (
    <div className="mx-auto w-full max-w-5xl space-y-5">
      <AgencyProfileHeader
        profile={profile}
        displayName={displayName}
        rating={rating}
        photoPriority
      />

      <ProfileAboutCard
        content={profile.about}
        label={t("aboutAgency")}
        placeholder={t("aboutAgencyPlaceholder")}
        readOnly
      />

      <ExpertiseDisplay domains={profile.expertise_domains ?? []} />

      <PricingDisplayCard
        pricing={pickDirectPricing(profile)}
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

      {profile.skills && profile.skills.length > 0 ? (
        <section
          aria-labelledby="agency-skills-display-title"
          className="bg-card border border-border rounded-2xl p-7 shadow-[var(--shadow-card)]"
        >
          <h2
            id="agency-skills-display-title"
            className="font-serif text-xl font-medium tracking-[-0.005em] text-foreground mb-3"
          >
            {tSkills("sectionTitle")}
          </h2>
          <SkillsDisplay skills={profile.skills} />
        </section>
      ) : null}

      <ProfileVideoCard
        videoUrl={profile.presentation_video_url}
        labels={{
          title: t("videoTitle"),
          emptyLabel: t("noVideo"),
          emptyDescription: t("addVideoDescAgency"),
        }}
        readOnly
        showWhenEmpty
      />

      <PublicPortfolioSection orgId={orgId} />

      <ProjectHistorySection orgId={orgId} readOnly />
    </div>
  )
}

// pickDirectPricing collapses the legacy agency pricing rows onto the
// single-row contract the shared PricingDisplayCard expects. Only the
// direct rate appears on the agency card — referral commissions live
// on the apporteur surface.
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
