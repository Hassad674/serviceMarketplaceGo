import type { MetadataRoute } from "next"
import { siteConfig } from "@/config/site"
import { fetchListingFirstPage } from "@/features/provider/api/search-server"
import { fetchSitemapJobs } from "@/features/job/api/sitemap-server"

// PERF-W-04 — dynamic sitemap aggregating the SEO-relevant
// surfaces: home, listings, public profiles (agencies, freelancers,
// referrers) and open opportunities. Static dashboard URLs are
// excluded — those go in robots.ts under `disallow`.
//
// Each public-facing entity is fetched server-side with a 60 s
// revalidate window (the underlying server fetcher uses
// `next.revalidate`) so re-rendering the sitemap is cheap.
//
// Note: when the underlying fetch fails, we degrade gracefully and
// emit only the static pages so a transient backend hiccup never
// strands Google with an empty sitemap.

const STATIC_PATHS = [
  { path: "/", changeFrequency: "daily" as const, priority: 1 },
  { path: "/agencies", changeFrequency: "daily" as const, priority: 0.9 },
  { path: "/freelancers", changeFrequency: "daily" as const, priority: 0.9 },
  { path: "/referrers", changeFrequency: "daily" as const, priority: 0.9 },
  { path: "/opportunities", changeFrequency: "daily" as const, priority: 0.9 },
  { path: "/login", changeFrequency: "yearly" as const, priority: 0.3 },
  { path: "/register", changeFrequency: "yearly" as const, priority: 0.3 },
]

export default async function sitemap(): Promise<MetadataRoute.Sitemap> {
  const base = siteConfig.url.replace(/\/$/, "")
  const now = new Date()

  // Parallelise the four search calls — they hit independent
  // endpoints, no need to serialise.
  const [agencies, freelancers, referrers, jobs] = await Promise.all([
    fetchListingFirstPage("agency"),
    fetchListingFirstPage("freelancer"),
    fetchListingFirstPage("referrer"),
    fetchSitemapJobs(),
  ])

  const staticEntries: MetadataRoute.Sitemap = STATIC_PATHS.map((s) => ({
    url: `${base}${s.path}`,
    lastModified: now,
    changeFrequency: s.changeFrequency,
    priority: s.priority,
  }))

  const docToEntry = (
    docs: NonNullable<typeof agencies>["documents"] | undefined,
    pathPrefix: string,
  ): MetadataRoute.Sitemap =>
    (docs ?? []).map((doc) => {
      const id = doc.organization_id ?? doc.id.split(":")[0] ?? doc.id
      const updatedAt = doc.updated_at
        ? new Date(doc.updated_at * 1000)
        : now
      return {
        url: `${base}${pathPrefix}/${id}`,
        lastModified: updatedAt,
        changeFrequency: "weekly" as const,
        priority: 0.8,
      }
    })

  const dynamicEntries: MetadataRoute.Sitemap = [
    ...docToEntry(agencies?.documents, "/agencies"),
    ...docToEntry(freelancers?.documents, "/freelancers"),
    ...docToEntry(referrers?.documents, "/referrers"),
    ...(jobs ?? []).map((j) => ({
      url: `${base}/opportunities/${j.id}`,
      lastModified: j.updated_at ? new Date(j.updated_at) : now,
      changeFrequency: "daily" as const,
      priority: 0.7,
    })),
  ]

  return [...staticEntries, ...dynamicEntries]
}
