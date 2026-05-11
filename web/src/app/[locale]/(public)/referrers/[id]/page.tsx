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
} from "@/shared/lib/seo/server-fetchers"
import { BreadcrumbNav } from "@/shared/components/seo/breadcrumb-nav"

type Props = {
  params: Promise<{ id: string; locale: string }>
}

// PERF-B: ISR revalidation window for the public referrer/apporteur
// profile. Stores the rendered RSC payload + API responses on Vercel's
// edge for 60s. Backend mirror is `Cache-Control: public, max-age=60,
// s-maxage=300` on `/api/v1/referrer-profiles/{orgID}`.
export const revalidate = 60

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
  const [profile, rating, reviews, t, tSeo] = await Promise.all([
    fetchReferrerProfileForMetadata(id),
    fetchPublicAverageRating(id),
    fetchPublicReviews(id, 5),
    getTranslations({ locale, namespace: "profile.referrer" }),
    getTranslations({ locale, namespace: "seo" }),
  ])
  // Localized fallback for the schema.org "name" field — same rationale
  // as the on-page header: the raw organization UUID is unfit for any
  // public surface, including SEO payloads consumed by crawlers.
  const fallbackName = t("displayNameFallback")

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
    // Wrapper matches /freelancers/[id] and /agencies/[id] — same
    // editorial column width (max-w-5xl ~1024px) and vertical rhythm
    // (space-y-5) so the three public detail pages stay visually
    // identical save for persona-specific content.
    <div className="mx-auto w-full max-w-5xl space-y-5">
      <BreadcrumbNav
        ariaLabel={tSeo("breadcrumbAriaLabel")}
        crumbs={breadcrumbCrumbs.map((c) => ({ label: c.label, href: c.href }))}
      />
      <div className="flex justify-end">
        <SendMessageButton targetOrgId={id} persona="referrer" />
      </div>
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
