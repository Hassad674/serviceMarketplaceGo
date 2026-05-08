/**
 * related-profiles.tsx — "Profils similaires" section rendered at the
 * bottom of every public profile page.
 *
 * Goals:
 *   - Internal linking: every card is a real `<a>` to the related
 *     profile, increasing crawl reach for Google.
 *   - SEO: the section emits a JSON-LD `ItemList` so Google can
 *     associate the related entities with the parent page.
 *   - UX: shows up to 6 cards, falls back to nothing when no
 *     candidates were found (the page should not render an awkward
 *     empty heading).
 *
 * The component is a Server Component — it receives the pre-fetched
 * documents and the locale-aware Link helper from next-intl. No client
 * JS needed for an internal-link anchor.
 */

import { Link } from "@i18n/navigation"
import { Portrait } from "@/shared/components/ui/portrait"
import type { RawSearchDocument } from "@/shared/lib/search/typesense-client"
import { safeJsonLd } from "@/shared/lib/json-ld"
import { absoluteUrl } from "@/shared/lib/seo/alternates"

const TYPE_TO_PATH = {
  freelancer: "/freelancers",
  agency: "/agencies",
  referrer: "/referrers",
} as const

export type RelatedProfileType = keyof typeof TYPE_TO_PATH

export interface RelatedProfilesLabels {
  /** Section heading — already translated. */
  heading: string
  /** Subheading / lead paragraph — already translated. */
  subheading: string
  /** Localized "voir le profil" CTA. */
  viewProfile: string
  /** Localized fallback when a card has no city. */
  cityFallback: string
}

export interface RelatedProfilesProps {
  type: RelatedProfileType
  documents: RawSearchDocument[]
  labels: RelatedProfilesLabels
}

export function RelatedProfiles({
  type,
  documents,
  labels,
}: RelatedProfilesProps) {
  if (documents.length === 0) return null
  const path = TYPE_TO_PATH[type]
  const itemList = buildItemList(type, documents)

  return (
    <section
      aria-labelledby="related-profiles-heading"
      className="mt-12 border-t border-border pt-10"
    >
      <header className="mb-6 flex flex-col gap-1">
        <h2
          id="related-profiles-heading"
          className="font-serif text-2xl font-semibold tracking-tight text-foreground"
        >
          {labels.heading}
        </h2>
        <p className="text-sm text-muted-foreground">{labels.subheading}</p>
      </header>
      <ul className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {documents.map((doc, index) => {
          const id = doc.organization_id ?? doc.id.split(":")[0] ?? doc.id
          return (
            <li key={doc.id}>
              <Link
                href={`${path}/${id}`}
                className="group flex h-full flex-col gap-3 rounded-2xl border border-border bg-surface p-4 transition-colors hover:border-primary"
              >
                <div className="flex items-center gap-3">
                  {doc.photo_url ? (
                    // eslint-disable-next-line @next/next/no-img-element
                    <img
                      src={doc.photo_url}
                      alt=""
                      width={56}
                      height={56}
                      loading="lazy"
                      className="h-14 w-14 rounded-full object-cover"
                    />
                  ) : (
                    <Portrait id={index} size={56} />
                  )}
                  <div className="min-w-0 flex-1">
                    <p className="truncate text-base font-medium text-foreground">
                      {doc.display_name}
                    </p>
                    <p className="truncate text-xs text-muted-foreground">
                      {doc.city || labels.cityFallback}
                    </p>
                  </div>
                </div>
                {doc.title ? (
                  <p className="line-clamp-2 text-sm text-muted-foreground">
                    {doc.title}
                  </p>
                ) : null}
                <span className="mt-auto text-sm font-medium text-primary group-hover:underline">
                  {labels.viewProfile}
                </span>
              </Link>
            </li>
          )
        })}
      </ul>
      <script
        type="application/ld+json"
        // ItemList JSON-LD lets Google associate this batch of related
        // profiles with the parent page's entity. safeJsonLd() covers
        // the </script> / U+2028 attack surface — see json-ld.ts.
        dangerouslySetInnerHTML={{ __html: safeJsonLd(itemList) }}
      />
    </section>
  )
}

function buildItemList(
  type: RelatedProfileType,
  documents: RawSearchDocument[],
): Record<string, unknown> {
  const path = TYPE_TO_PATH[type]
  const itemType = type === "agency" ? "Organization" : "Person"
  return {
    "@context": "https://schema.org",
    "@type": "ItemList",
    itemListElement: documents.map((doc, index) => {
      const id = doc.organization_id ?? doc.id.split(":")[0] ?? doc.id
      return {
        "@type": "ListItem",
        position: index + 1,
        item: {
          "@type": itemType,
          "@id": absoluteUrl(`${path}/${id}`),
          name: doc.display_name,
          url: absoluteUrl(`${path}/${id}`),
        },
      }
    }),
  }
}
