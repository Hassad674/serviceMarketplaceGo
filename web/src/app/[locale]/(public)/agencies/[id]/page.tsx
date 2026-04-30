import type { Metadata } from "next"
import { getTranslations } from "next-intl/server"
import { PublicProfile } from "@/features/provider/components/public-profile"
import { SendMessageButton } from "@/features/messaging/components/send-message-button"
import { fetchAgencyProfileForMetadata } from "@/features/provider/api/agency-profile-server"
import { safeJsonLd } from "@/shared/lib/json-ld"
import type { Profile } from "@/features/provider/api/profile-api"

type Props = {
  params: Promise<{ id: string; locale: string }>
}

// generateMetadata renders dynamic SEO head tags from the agency
// profile (PERF-W-06). Falls back to generic strings only when the
// fetch fails — listing pages still link here, so the URL stays
// indexable even on transient backend errors.
export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { id, locale } = await params
  const t = await getTranslations({ locale, namespace: "publicProfile" })
  const profile = await fetchAgencyProfileForMetadata(id)

  const displayName = profile?.title || t("agencyProfile")
  const titleSuffix = t("agencyProfile")
  const title = `${displayName} — ${titleSuffix} | Marketplace Service`
  const description =
    profile?.about?.slice(0, 160) || t("agencyProfileDesc")
  const canonical = `/agencies/${id}`

  return {
    title,
    description,
    alternates: { canonical },
    openGraph: {
      title,
      description,
      type: "profile",
      images: profile?.photo_url
        ? [
            {
              url: profile.photo_url,
              width: 400,
              height: 400,
              alt: displayName,
            },
          ]
        : undefined,
    },
    twitter: {
      card: "summary",
      title,
      description,
    },
  }
}

export default async function AgencyProfilePage({ params }: Props) {
  const { id } = await params
  const profile = await fetchAgencyProfileForMetadata(id)
  return (
    <div className="space-y-6">
      <PublicProfile orgId={id} type="agency" />
      {profile ? <JsonLd profileId={id} profile={profile} /> : null}
      <div className="flex justify-center">
        <SendMessageButton targetOrgId={id} />
      </div>
    </div>
  )
}

interface JsonLdProps {
  profileId: string
  profile: Profile
}

function JsonLd({ profileId, profile }: JsonLdProps) {
  // Schema.org Organization payload — exposes the agency as a B2B
  // entity Google recognises in rich results. `aggregateRating` is
  // omitted when the org has zero reviews to avoid skewing the
  // schema's required minima.
  const payload: Record<string, unknown> = {
    "@context": "https://schema.org",
    "@type": "Organization",
    "@id": `/agencies/${profileId}`,
    name: profile.title || profileId,
    description: profile.about || undefined,
    image: profile.photo_url || undefined,
    knowsAbout: profile.skills?.map((s) => s.display_text),
    address: profile.city
      ? {
          "@type": "PostalAddress",
          addressLocality: profile.city,
          addressCountry: profile.country_code || undefined,
        }
      : undefined,
  }
  return (
    <script
      type="application/ld+json"
      // SEO JSON-LD must be rendered as raw JSON; React escaping
      // would break the schema. `profile.about` is user-authored
      // text, so we route through safeJsonLd() to neutralise
      // </script>, --> and the unicode line/paragraph separators
      // before injecting via dangerouslySetInnerHTML.
      dangerouslySetInnerHTML={{ __html: safeJsonLd(payload) }}
    />
  )
}
