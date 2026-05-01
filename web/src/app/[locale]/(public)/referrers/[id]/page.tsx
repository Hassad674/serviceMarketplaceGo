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
  const canonical = `/referrers/${id}`

  return {
    title,
    description,
    alternates: { canonical },
    openGraph: {
      title,
      description,
      type: "profile",
      images: profile?.photo_url
        ? [{ url: profile.photo_url, width: 400, height: 400, alt: displayName }]
        : undefined,
    },
    twitter: {
      card: "summary",
      title,
      description,
    },
  }
}

export default async function ReferrerProfilePage({ params }: Props) {
  const { id, locale } = await params
  const t = await getTranslations({ locale, namespace: "profile.referrer" })
  const profile = await fetchReferrerProfileForMetadata(id)
  // Localized fallback for the schema.org "name" field — same rationale
  // as the on-page header: the raw organization UUID is unfit for any
  // public surface, including SEO payloads consumed by crawlers.
  const fallbackName = t("displayNameFallback")
  return (
    <div className="space-y-6">
      <ReferrerPublicProfileLoader orgId={id} />
      {profile ? (
        <JsonLd
          profileId={id}
          profile={profile}
          fallbackName={fallbackName}
        />
      ) : null}
      <div className="flex justify-center">
        <SendMessageButton targetOrgId={id} />
      </div>
    </div>
  )
}

interface JsonLdProps {
  profileId: string
  profile: NonNullable<Awaited<ReturnType<typeof fetchReferrerProfileForMetadata>>>
  fallbackName: string
}

function JsonLd({ profileId, profile, fallbackName }: JsonLdProps) {
  // Referrer schema uses Person with jobTitle hinting the "business
  // referrer" role. Skills are intentionally omitted — they live on
  // the freelance persona of the same org, not here.
  const payload = {
    "@context": "https://schema.org",
    "@type": "Person",
    "@id": `/referrers/${profileId}`,
    name: profile.title || fallbackName,
    jobTitle: profile.title || fallbackName,
    description: profile.about || undefined,
    image: profile.photo_url || undefined,
    address: profile.city
      ? {
          "@type": "PostalAddress",
          addressLocality: profile.city,
          addressCountry: profile.country_code || undefined,
        }
      : undefined,
    knowsLanguage: profile.languages_professional,
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
