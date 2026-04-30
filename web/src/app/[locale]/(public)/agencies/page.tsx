import type { Metadata } from "next"
import { getTranslations } from "next-intl/server"
import { SearchPage } from "@/features/provider/components/search-page"
import { fetchListingFirstPage } from "@/features/provider/api/search-server"
import { buildItemList } from "@/features/provider/api/listing-jsonld"
import { safeJsonLd } from "@/shared/lib/json-ld"

// /agencies lists every organization of type `agency`. PERF-W-02:
// the page is now an async Server Component that pre-fetches the
// first 20 documents server-side, renders the JSON-LD ItemList for
// Google, and seeds the client SearchPage with the same results so
// hydration paints the cards immediately (no client refetch).

type Props = {
  params: Promise<{ locale: string }>
}

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { locale } = await params
  const t = await getTranslations({ locale, namespace: "publicListing" })
  const firstPage = await fetchListingFirstPage("agency")
  const count = firstPage?.found ?? 0

  const title = t("agencies.title", { count })
  const description = t("agencies.description", { count })

  return {
    title,
    description,
    alternates: { canonical: "/agencies" },
    openGraph: {
      type: "website",
      title,
      description,
    },
    twitter: {
      card: "summary",
      title,
      description,
    },
  }
}

export default async function AgenciesDirectoryPage() {
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
      <SearchPage type="agency" initialFirstPage={firstPage ?? undefined} />
    </>
  )
}
