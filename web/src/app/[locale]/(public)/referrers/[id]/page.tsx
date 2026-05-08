import type { Metadata } from "next"
import { getTranslations } from "next-intl/server"
import { SendMessageButton } from "@/features/messaging/components/send-message-button"
import {
  ReferrerPublicProfileLoader,
} from "@/features/referrer-profile/components/referrer-public-profile-loader"
import {
  fetchReferrerProfileForMetadata,
} from "@/features/referrer-profile/api/referrer-profile-server"
import { safeJsonLd } from "@/shared/lib/json-ld"
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

type Props = {
  params: Promise<{ id: string; locale: string }>
}

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { id, locale } = await params
  const t = await getTranslations({ locale, namespace: "profile.referrer" })
  const profile = await fetchReferrerProfileForMetadata(id)

  const displayName = profile?.title || t("publicTitleSuffix")
  const titleSuffix = t("publicTitleSuffix")
  const title = `${displayName} — ${titleSuffix} | Marketplace Service`
  const description = t("publicDescription", {
    name: displayName,
    title: profile?.title ? profile.title : "empty",
  })
  const alternates = buildAlternates({
    locale: locale as SupportedLocale,
    path: `/referrers/${id}`,
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

export default async function ReferrerProfilePage({ params }: Props) {
  const { id, locale } = await params
  const [profile, rating, reviews, related, t, tSeo] = await Promise.all([
    fetchReferrerProfileForMetadata(id),
    fetchPublicAverageRating(id),
    fetchPublicReviews(id, 5),
    fetchRelatedProfiles({
      type: "referrer",
      excludeOrgId: id,
      primaryExpertise: undefined,
      city: undefined,
    }),
    getTranslations({ locale, namespace: "profile.referrer" }),
    getTranslations({ locale, namespace: "seo" }),
  ])
  // Localized fallback for the schema.org "name" field — same rationale
  // as the on-page header: the raw organization UUID is unfit for any
  // public surface, including SEO payloads consumed by crawlers.
  const fallbackName = t("displayNameFallback")

  const filteredRelated =
    profile && related.length > 0
      ? related
          .sort((a, b) => {
            // Referrers don't have expertise; rank by city + rating.
            const aMatch =
              profile.city && a.city === profile.city ? 4 : 0
            const bMatch =
              profile.city && b.city === profile.city ? 4 : 0
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
      label: tSeo("breadcrumbReferrers"),
      href: "/referrers",
      item: absoluteUrl(`/${locale}/referrers`),
    },
    {
      label: profile?.title || fallbackName,
    },
  ]

  return (
    <div className="space-y-6">
      <BreadcrumbNav
        ariaLabel={tSeo("breadcrumbAriaLabel")}
        crumbs={breadcrumbCrumbs.map((c) => ({ label: c.label, href: c.href }))}
      />
      <ReferrerPublicProfileLoader orgId={id} />
      {profile ? (
        <JsonLd
          profileId={id}
          profile={profile}
          fallbackName={fallbackName}
          rating={rating}
          reviews={reviews}
          anonymousReviewer={tSeo("anonymousReviewer")}
        />
      ) : null}
      <BreadcrumbsJsonLd
        crumbs={breadcrumbCrumbs.map((c) => ({ name: c.label, item: c.item }))}
      />
      <div className="flex justify-center">
        <SendMessageButton targetOrgId={id} />
      </div>
      <RelatedProfiles
        type="referrer"
        documents={filteredRelated}
        labels={{
          heading: tSeo("relatedHeadingReferrer"),
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
  profile: NonNullable<Awaited<ReturnType<typeof fetchReferrerProfileForMetadata>>>
  fallbackName: string
  rating: Awaited<ReturnType<typeof fetchPublicAverageRating>>
  reviews: Awaited<ReturnType<typeof fetchPublicReviews>>
  anonymousReviewer: string
}

function JsonLd({
  profileId,
  profile,
  fallbackName,
  rating,
  reviews,
  anonymousReviewer,
}: JsonLdProps) {
  // Referrer schema uses Person with jobTitle hinting the "business
  // referrer" role. Skills are intentionally omitted — they live on
  // the freelance persona of the same org, not here. Reviews are the
  // recommendation signal that matters for an apporteur d'affaires.
  const aggregateRating = buildAggregateRating({ rating })
  const reviewItems = buildReviewItems({
    reviews,
    anonymousAuthorLabel: anonymousReviewer,
  })

  const payload: Record<string, unknown> = {
    "@context": "https://schema.org",
    "@type": "Person",
    "@id": absoluteUrl(`/referrers/${profileId}`),
    name: profile.title || fallbackName,
    jobTitle: profile.title || fallbackName,
    description: profile.about || undefined,
    image: profile.photo_url ? absoluteUrl(profile.photo_url) : undefined,
    url: absoluteUrl(`/referrers/${profileId}`),
    address: profile.city
      ? {
          "@type": "PostalAddress",
          addressLocality: profile.city,
          addressCountry: profile.country_code || undefined,
        }
      : undefined,
    knowsLanguage: profile.languages_professional,
    aggregateRating,
    review: reviewItems,
  }
  return (
    <script
      type="application/ld+json"
      // SEO JSON-LD must be rendered as raw JSON; React escaping would
      // break the schema. `profile.about` is user-authored, so we route
      // through safeJsonLd() to neutralize </script>, --> and U+2028 /
      // U+2029 separators that JSON.stringify leaves intact.
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
