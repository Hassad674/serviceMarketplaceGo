import type { Metadata } from "next"
import { getTranslations } from "next-intl/server"
import { PublicProfile } from "@/features/provider/components/public-profile"
import { SendMessageButton } from "@/features/messaging/components/send-message-button"
import { fetchAgencyProfileForMetadata } from "@/features/provider/api/agency-profile-server"
import { safeJsonLd } from "@/shared/lib/json-ld"
import type { Profile } from "@/features/provider/api/profile-api"
import {
  buildAlternates,
  absoluteUrl,
  type SupportedLocale,
} from "@/shared/lib/seo/alternates"
import { buildBreadcrumbList } from "@/shared/lib/seo/breadcrumbs"
import {
  buildAggregateRating,
  buildReviewItems,
} from "@/shared/lib/seo/rating"
import {
  fetchPublicAverageRating,
  fetchPublicReviews,
  fetchRelatedProfiles,
} from "@/shared/lib/seo/server-fetchers"
import { BreadcrumbNav } from "@/shared/components/seo/breadcrumb-nav"
import { RelatedProfiles } from "@/shared/components/seo/related-profiles"
import type { Review, AverageRating } from "@/shared/types/review"

type Props = {
  params: Promise<{ id: string; locale: string }>
}

// generateMetadata renders dynamic SEO head tags from the agency
// profile (PERF-W-06 + PERF-W-08). Falls back to generic strings
// only when the fetch fails — listing pages still link here, so the
// URL stays indexable even on transient backend errors.
export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { id, locale } = await params
  const t = await getTranslations({ locale, namespace: "publicProfile" })
  const profile = await fetchAgencyProfileForMetadata(id)

  const displayName = profile?.title || t("agencyProfile")
  const titleSuffix = t("agencyProfile")
  const title = `${displayName} — ${titleSuffix} | Marketplace Service`
  const description =
    profile?.about?.slice(0, 160) || t("agencyProfileDesc")
  const alternates = buildAlternates({
    locale: locale as SupportedLocale,
    path: `/agencies/${id}`,
  })

  return {
    title,
    description,
    alternates,
    openGraph: {
      title,
      description,
      type: "profile",
      url: alternates.canonical,
      locale: locale === "fr" ? "fr_FR" : "en_US",
    },
    twitter: {
      card: "summary_large_image",
      title,
      description,
    },
  }
}

export default async function AgencyProfilePage({ params }: Props) {
  const { id, locale } = await params
  const [profile, rating, reviews, related, tProfile, tSeo] = await Promise.all([
    fetchAgencyProfileForMetadata(id),
    fetchPublicAverageRating(id),
    fetchPublicReviews(id, 5),
    fetchRelatedProfiles({
      type: "agency",
      excludeOrgId: id,
      primaryExpertise: undefined,
      city: undefined,
    }),
    getTranslations({ locale, namespace: "publicProfile" }),
    getTranslations({ locale, namespace: "seo" }),
  ])

  const filteredRelated =
    profile && related.length > 0
      ? related
          .sort((a, b) => {
            const aMatch = primaryMatch(a, profile)
            const bMatch = primaryMatch(b, profile)
            if (aMatch !== bMatch) return bMatch - aMatch
            return b.rating_average - a.rating_average
          })
          .slice(0, 6)
      : related

  const breadcrumbCrumbs = [
    {
      label: tSeo("breadcrumbHome"),
      href: "/",
      item: absoluteUrl(`/${locale}`),
    },
    {
      label: tSeo("breadcrumbAgencies"),
      href: "/agencies",
      item: absoluteUrl(`/${locale}/agencies`),
    },
    {
      label: profile?.title || tProfile("agencyProfile"),
    },
  ]

  return (
    // Wrapper matches /freelancers/[id] and /referrers/[id] — same
    // editorial column width (max-w-5xl ~1024px) and vertical rhythm
    // (space-y-5) so the three public detail pages stay visually
    // identical save for persona-specific content.
    <div className="mx-auto w-full max-w-5xl space-y-5">
      <BreadcrumbNav
        ariaLabel={tSeo("breadcrumbAriaLabel")}
        crumbs={breadcrumbCrumbs.map((c) => ({ label: c.label, href: c.href }))}
      />
      <PublicProfile orgId={id} type="agency" />
      {profile ? (
        <JsonLd
          profileId={id}
          profile={profile}
          rating={rating}
          reviews={reviews}
          anonymousReviewer={tSeo("anonymousReviewer")}
        />
      ) : null}
      <BreadcrumbsJsonLd
        crumbs={breadcrumbCrumbs.map((c) => ({ name: c.label, item: c.item }))}
      />
      <div className="flex justify-center pt-2">
        <SendMessageButton targetOrgId={id} />
      </div>
      <RelatedProfiles
        type="agency"
        documents={filteredRelated}
        labels={{
          heading: tSeo("relatedHeadingAgency"),
          subheading: tSeo("relatedSubheading"),
          viewProfile: tSeo("relatedViewProfile"),
          cityFallback: tSeo("relatedCityFallback"),
        }}
      />
    </div>
  )
}

interface JsonLdProps {
  profileId: string
  profile: Profile
  rating: AverageRating | null
  reviews: Review[] | null
  anonymousReviewer: string
}

function JsonLd({
  profileId,
  profile,
  rating,
  reviews,
  anonymousReviewer,
}: JsonLdProps) {
  // Schema.org Organization payload — exposes the agency as a B2B
  // entity Google recognises in rich results. `aggregateRating` +
  // `review[]` are emitted only when the org has at least one
  // published review so we never ship a hollow ratings block.
  const knowsAbout = mergeKnowsAbout(
    profile.skills?.map((s) => s.display_text) ?? [],
    profile.expertise_domains ?? [],
  )
  const aggregateRating = buildAggregateRating({ rating })
  const reviewItems = buildReviewItems({
    reviews,
    anonymousAuthorLabel: anonymousReviewer,
  })

  const payload: Record<string, unknown> = {
    "@context": "https://schema.org",
    "@type": "Organization",
    "@id": absoluteUrl(`/agencies/${profileId}`),
    name: profile.title || profileId,
    description: profile.about || undefined,
    image: profile.photo_url ? absoluteUrl(profile.photo_url) : undefined,
    url: absoluteUrl(`/agencies/${profileId}`),
    knowsAbout: knowsAbout.length > 0 ? knowsAbout : undefined,
    address: profile.city
      ? {
          "@type": "PostalAddress",
          addressLocality: profile.city,
          addressCountry: profile.country_code || undefined,
        }
      : undefined,
    aggregateRating,
    review: reviewItems,
  }
  return (
    <script
      type="application/ld+json"
      // Rendered as raw JSON on purpose — React escaping would break
      // the structured-data schema. `profile.about` is user-authored
      // text, so we route through safeJsonLd() to neutralise </script>,
      // --> and U+2028 / U+2029 separators that JSON.stringify leaves
      // intact.
      dangerouslySetInnerHTML={{ __html: safeJsonLd(payload) }}
    />
  )
}

interface BreadcrumbsJsonLdProps {
  crumbs: Array<{ name: string; item?: string }>
}

function BreadcrumbsJsonLd({ crumbs }: BreadcrumbsJsonLdProps) {
  return (
    <script
      type="application/ld+json"
      dangerouslySetInnerHTML={{
        __html: safeJsonLd(buildBreadcrumbList(crumbs)),
      }}
    />
  )
}

function mergeKnowsAbout(skills: string[], expertises: string[]): string[] {
  const combined = [...skills, ...expertises]
    .map((entry) => entry.trim())
    .filter(Boolean)
  return Array.from(new Set(combined))
}

function primaryMatch(
  doc: { expertise_domains: string[]; city?: string },
  profile: { expertise_domains?: string[]; city?: string },
): number {
  let score = 0
  const primary = profile.expertise_domains?.[0]
  if (primary && doc.expertise_domains?.includes(primary)) score += 10
  if (profile.city && doc.city === profile.city) score += 4
  return score
}
