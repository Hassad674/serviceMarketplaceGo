import type { Metadata } from "next"
import { getTranslations } from "next-intl/server"
import { ArrowLeft } from "lucide-react"
import { Link } from "@i18n/navigation"
import { SendMessageButton } from "@/features/messaging/components/send-message-button"
import {
  FreelancePublicProfileLoader,
} from "@/features/freelance-profile/components/freelance-public-profile-loader"
import {
  fetchFreelanceProfileForMetadata,
} from "@/features/freelance-profile/api/freelance-profile-server"
import { safeJsonLd } from "@/shared/lib/json-ld"

type Props = {
  params: Promise<{ id: string; locale: string }>
}

// generateMetadata runs on the server and populates the SEO head.
// We ask the backend for the freelance aggregate ONCE here; the
// client then re-fetches via TanStack Query when the loader mounts
// — the double-fetch is acceptable because the metadata fetch is
// revalidated on a 2-minute ISR window and the client fetch hits
// the TanStack cache, not the network, on hydration.
export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { id, locale } = await params
  const t = await getTranslations({ locale, namespace: "profile.freelance" })
  const profile = await fetchFreelanceProfileForMetadata(id)

  const displayName = profile?.title || t("publicTitleSuffix")
  const titleSuffix = t("publicTitleSuffix")
  const title = `${displayName} — ${titleSuffix} | Marketplace Service`
  const description = t("publicDescription", {
    name: displayName,
    title: profile?.title ? profile.title : "empty",
  })
  const canonical = `/freelancers/${id}`

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

export default async function FreelancerProfilePage({ params }: Props) {
  const { id, locale } = await params
  const profile = await fetchFreelanceProfileForMetadata(id)
  const t = await getTranslations({ locale, namespace: "profile.freelance" })
  return (
    <div className="mx-auto w-full max-w-5xl space-y-5">
      <Link
        href="/freelancers"
        className="inline-flex items-center gap-1.5 text-xs text-muted-foreground transition-colors hover:text-foreground"
      >
        <ArrowLeft className="h-3 w-3" aria-hidden="true" />
        {t("backToList")}
      </Link>
      <FreelancePublicProfileLoader orgId={id} />
      {profile ? <JsonLd profileId={id} profile={profile} /> : null}
      <div className="flex justify-center pt-2">
        <SendMessageButton targetOrgId={id} />
      </div>
    </div>
  )
}

interface JsonLdProps {
  profileId: string
  profile: NonNullable<Awaited<ReturnType<typeof fetchFreelanceProfileForMetadata>>>
}

function JsonLd({ profileId, profile }: JsonLdProps) {
  const payload = {
    "@context": "https://schema.org",
    "@type": "Person",
    "@id": `/freelancers/${profileId}`,
    name: profile.title || profileId,
    jobTitle: profile.title || undefined,
    description: profile.about || undefined,
    image: profile.photo_url || undefined,
    address: profile.city
      ? {
          "@type": "PostalAddress",
          addressLocality: profile.city,
          addressCountry: profile.country_code || undefined,
        }
      : undefined,
    knowsAbout: profile.skills?.map((s) => s.display_text),
    knowsLanguage: profile.languages_professional,
  }
  return (
    <script
      type="application/ld+json"
      // SEO JSON-LD must be rendered as raw JSON; React escaping would
      // break the schema. `profile.about` is user-authored text, so we
      // route through safeJsonLd() to neutralize </script>, --> and the
      // unicode line/paragraph separators before injecting via
      // dangerouslySetInnerHTML.
      dangerouslySetInnerHTML={{ __html: safeJsonLd(payload) }}
    />
  )
}
