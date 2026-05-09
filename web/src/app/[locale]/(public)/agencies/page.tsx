import type { Metadata } from "next"
import { getTranslations } from "next-intl/server"
import { SearchPage } from "@/features/provider/components/search-page"
import { fetchListingFirstPage } from "@/features/provider/api/search-server"
import { buildItemList } from "@/features/provider/api/listing-jsonld"
import { safeJsonLd } from "@/shared/lib/json-ld"
import {
  buildAlternates,
  type SupportedLocale,
} from "@/shared/lib/seo/alternates"

// /agencies lists every organization of type `agency`. PERF-W-02:
// the page is now an async Server Component that pre-fetches the
// first 20 documents server-side, renders the JSON-LD ItemList for
// Google, and seeds the client SearchPage with the same results so
// hydration paints the cards immediately (no client refetch).

type Props = {
  params: Promise<{ locale: string }>
  searchParams?: Promise<Record<string, string | string[] | undefined>>
}

// readInitialQuery extracts the `q` URL param when the visitor
// arrives from the landing search bar. Same logic as on the
// freelancers route — kept inline (not extracted) until the rule of
// three is met.
function readInitialQuery(
  searchParams?: Record<string, string | string[] | undefined>,
): string {
  const raw = searchParams?.q
  if (!raw) return ""
  const value = Array.isArray(raw) ? raw[0] : raw
  return typeof value === "string" ? value : ""
}

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { locale } = await params
  const t = await getTranslations({ locale, namespace: "publicListing" })
  const firstPage = await fetchListingFirstPage("agency")
  const count = firstPage?.found ?? 0

  const title = t("agencies.title", { count })
  const description = t("agencies.description", { count })
  const alternates = buildAlternates({
    locale: locale as SupportedLocale,
    path: "/agencies",
  })

  return {
    title,
    description,
    alternates,
    openGraph: {
      type: "website",
      title,
      description,
      url: alternates.canonical,
      locale: locale === "fr" ? "fr_FR" : "en_US",
    },
    twitter: {
      card: "summary",
      title,
      description,
    },
  }
}

export default async function AgenciesDirectoryPage({ searchParams }: Props) {
  const resolvedSearchParams = await searchParams
  const initialQuery = readInitialQuery(resolvedSearchParams)
  const firstPage = await fetchListingFirstPage("agency")
  const itemList = firstPage
    ? buildItemList({
        type: "agency",
        documents: firstPage.documents,
        totalFound: firstPage.found,
      })
    : null

  return (
    <>
      {itemList ? (
        <script
          type="application/ld+json"
          // JSON-LD must be inlined as raw JSON for Google to parse.
          // safeJsonLd neutralises </script>, --> and unicode line
          // separators before injection (XSS hardening).
          dangerouslySetInnerHTML={{ __html: safeJsonLd(itemList) }}
        />
      ) : null}
      <SearchPage
        type="agency"
        initialFirstPage={firstPage ?? undefined}
        initialQuery={initialQuery}
      />
    </>
  )
}
